package health_check_worker

//goland:noinspection SpellCheckingInspection
import (
	"context"
	"encoding/hex"
	"fmt"
	libapp "github.com/EscanBE/go-lib/app"
	"github.com/EscanBE/go-lib/logging"
	"github.com/bcdevtools/validator-health-check/codec"
	"github.com/bcdevtools/validator-health-check/config"
	"github.com/bcdevtools/validator-health-check/constants"
	chainreg "github.com/bcdevtools/validator-health-check/registry/chain_registry"
	rpcreg "github.com/bcdevtools/validator-health-check/registry/rpc_client_registry"
	usereg "github.com/bcdevtools/validator-health-check/registry/user_registry"
	valaddreg "github.com/bcdevtools/validator-health-check/registry/validator_address_registry"
	tpsvc "github.com/bcdevtools/validator-health-check/services/telegram_push_message_svc"
	tptypes "github.com/bcdevtools/validator-health-check/services/telegram_push_message_svc/types"
	"github.com/bcdevtools/validator-health-check/utils"
	workertypes "github.com/bcdevtools/validator-health-check/work/health_check_worker/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/pkg/errors"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"sort"
	"strings"
	"time"
)

// Worker represents for a worker, itself holds things needed for doing business logic, especially its own `HcwContext`
type Worker struct {
	ctx *workertypes.HcwContext
}

// NewHcWorker creates new health-check worker
func NewHcWorker(wCtx *workertypes.HcwContext) Worker {
	return Worker{
		ctx: wCtx,
	}
}

// Start performs business logic of worker
func (w Worker) Start() {
	logger := w.ctx.AppCtx.Logger
	defer libapp.TryRecoverAndExecuteExitFunctionIfRecovered(logger)

	healthCheckInterval := w.ctx.AppCtx.AppConfig.General.HealthCheckInterval
	if healthCheckInterval < 30*time.Second {
		healthCheckInterval = 30 * time.Second
	}

	for {
		time.Sleep(30 * time.Millisecond)

		registeredChainConfig := chainreg.GetFirstChainConfigForHealthCheckRL(healthCheckInterval)
		if registeredChainConfig == nil {
			//logger.Debug("no chain to health-check", "wid", w.Ctx.WorkerID)
			continue
		}

		func(registeredChainConfig chainreg.RegisteredChainConfig) {
			allWatchersIdentity := make([]string, 0)
			watchersIdentityToUserRecord := make(map[string]config.UserRecord)
			for _, validator := range registeredChainConfig.GetValidators() {
				for _, identity := range validator.WatchersIdentity {
					userRecord, found := usereg.GetUserRecordByIdentityRL(identity)
					if !found {
						continue
					}
					if userRecord.TelegramConfig.IsEmptyOrIncompleteConfig() {
						panic(fmt.Sprintf("telegram config is empty or incomplete, weird! identity: %s", identity))
					}
					allWatchersIdentity = append(allWatchersIdentity, identity)
					watchersIdentityToUserRecord[identity] = userRecord
				}
			}

			chainName := registeredChainConfig.GetChainName()
			logger.Debug("health-checking chain", "chain", chainName, "wid", w.ctx.WorkerID)

			var countEnqueuedTelegramMessages int

			defer func() {
				logger.Info("enqueued telegram messages", "count", countEnqueuedTelegramMessages, "chain", chainName)
			}()

			enqueueTelegramMessageByIdentity := func(validator, message string, fatal bool, identities ...string) {
				countEnqueuedTelegramMessages++
				for _, identity := range identities {
					userRecord, found := watchersIdentityToUserRecord[identity]
					if !found {
						logger.Error("can not enqueue telegram message, user not found", "validator", validator, "chain", chainName, "fatal", fatal, "identity", identity, "message", message)
						continue
					}

					var messagePrefix string
					if fatal {
						messagePrefix += "*FATAL!!*"
					}
					messagePrefix += fmt.Sprintf("[%s]", chainName)
					if validator != "" {
						messagePrefix += fmt.Sprintf("[%s]", validator)
					}
					message = fmt.Sprintf("%s %s", messagePrefix, message)

					tpsvc.EnqueueMessageWL(tptypes.QueueMessage{
						ReceiverID: userRecord.TelegramConfig.UserId,
						Priority:   userRecord.Root,
						Message:    message,
					})

					logger.Debug("enqueued telegram message by identity", "message", message, "identity", identity)
				}
			}

			var healthCheckError error
			defer func() {
				if healthCheckError == nil {
					logger.Debug("health-check successfully", "chain", chainName)
					return
				}

				logger.Error("failed to health-check chain", "chain", chainName, "error", healthCheckError.Error())
				enqueueTelegramMessageByIdentity(
					"",
					fmt.Sprintf("failed to health-check, error: %s", healthCheckError.Error()),
					false,
					allWatchersIdentity...,
				)
			}()

			// get the most healthy RPC
			rpcClient, mostHealthyEndpoint, latestBlockTime, errFetchHealthyRpc := getMostHealthyRpc(registeredChainConfig.GetRPCs(), registeredChainConfig.GetChainId(), logger)
			if errFetchHealthyRpc != nil {
				healthCheckError = errors.Wrap(errFetchHealthyRpc, "failed to get most healthy RPC")
				return
			}

			logger.Debug("most healthy RPC", "chain", chainName, "endpoint", mostHealthyEndpoint, "latest_block_time", latestBlockTime)
			if time.Since(latestBlockTime) > constants.INFORM_TELEGRAM_IF_BLOCK_OLDER_THAN {
				enqueueTelegramMessageByIdentity(
					"",
					fmt.Sprintf("latest block time of the most health RPC is too old: %s, diff %s", latestBlockTime, time.Since(latestBlockTime)),
					false,
					allWatchersIdentity...,
				)
			}

			registeredChainConfig.InformPriorityLatestHealthyRpcWL(mostHealthyEndpoint)

			// fetch all validators
			stakingValidators, errFetchStakingValidators := getAllValidators(rpcClient)
			if errFetchStakingValidators != nil {
				healthCheckError = errors.Wrap(errFetchStakingValidators, "failed to get all validators")
				return
			}
			stakingValidatorByValoper := make(map[string]stakingtypes.Validator)
			for _, validator := range stakingValidators {
				stakingValidatorByValoper[validator.OperatorAddress] = validator
			}

			// reload mapping
			w.reloadMappingValAddressIfNeeded(registeredChainConfig, stakingValidators)

			// fetch all signingInfos
			signingInfos, errFetchSigningInfo := getAllSigningInfos(rpcClient)
			if errFetchSigningInfo != nil {
				enqueueTelegramMessageByIdentity(
					"",
					fmt.Sprintf("failed to get all validator signing infos, for uptime-check, error: %s", errFetchSigningInfo.Error()),
					false,
					allWatchersIdentity...,
				)
			}
			valconsToSigningInfo := make(map[string]slashingtypes.ValidatorSigningInfo)
			for _, signingInfo := range signingInfos {
				valconsToSigningInfo[signingInfo.Address] = signingInfo
			}

			// fetch slashing params
			slashingParams, errFetchSlashingParams := getSlashingParams(rpcClient)
			if errFetchSlashingParams != nil {
				enqueueTelegramMessageByIdentity(
					"",
					fmt.Sprintf("failed to get all slashing params, for uptime-check, error: %s", errFetchSlashingParams.Error()),
					false,
					allWatchersIdentity...,
				)
			}

			// prepare ranking
			sort.Slice(stakingValidators, func(i, j int) bool {
				left := stakingValidators[i]
				right := stakingValidators[j]

				if left.IsBonded() == right.IsBonded() {
					return left.Tokens.GT(right.Tokens)
				}

				if left.IsBonded() {
					return true
				} else {
					return false
				}
			})
			valoperToRank := make(map[string]int)
			for i, validator := range stakingValidators {
				valoperToRank[validator.OperatorAddress] = i + 1
			}

			// health-check each validator

			for _, validator := range registeredChainConfig.GetValidators() {
				valoperAddr := validator.ValidatorOperatorAddress

				if paused, _ := chainreg.IsValidatorPausedRL(valoperAddr); paused {
					logger.Info("validator paused, skipping health-check", "chain", chainName, "valoper", valoperAddr)
					continue
				}

				cacheHc := CacheValidatorHealthCheck{
					Valoper: valoperAddr,
				}

				stakingValidator, found := stakingValidatorByValoper[valoperAddr]
				if !found {
					enqueueTelegramMessageByIdentity(
						valoperAddr,
						"validator not found",
						false,
						validator.WatchersIdentity...,
					)
					continue
				}

				rank, found := valoperToRank[valoperAddr]
				if found {
					cacheHc.Rank = rank
				}

				moniker := stakingValidator.Description.Moniker
				cacheHc.Moniker = moniker

				switch stakingValidator.Status {
				case stakingtypes.Bonded:
					// all good
				case stakingtypes.Unbonded:
					enqueueTelegramMessageByIdentity(
						valoperAddr,
						fmt.Sprintf("validator %s is un-bonded! Tombstoned? Contact to unsubscribe this validator", moniker),
						true,
						validator.WatchersIdentity...,
					)
				case stakingtypes.Unbonding:
					enqueueTelegramMessageByIdentity(
						valoperAddr,
						fmt.Sprintf("validator %s is unbonding! Fall-out of active set? Was jailed?%s", moniker, func() string {
							if rank == 0 {
								return ""
							}
							return fmt.Sprintf(" Rank %d.", rank)
						}()),
						true,
						validator.WatchersIdentity...,
					)
				default:
					enqueueTelegramMessageByIdentity(
						valoperAddr,
						fmt.Sprintf("unknown bond status %s", stakingValidator.Status),
						true,
						validator.WatchersIdentity...,
					)
				}

				cacheHc.BondStatus = &stakingValidator.Status

				if errFetchSigningInfo == nil { // skip check if error on fetch, error message informed before
					valconsAddr, found := valaddreg.GetValconsByValoperRL(chainName, valoperAddr)
					cacheHc.Valcons = valconsAddr
					if found {
						signingInfo, found := valconsToSigningInfo[valconsAddr]
						if found {
							if signingInfo.Tombstoned {
								sendToWatchers := tpsvc.ShouldSendMessageWL(
									tpsvc.PreventSpammingCaseTomeStoned,
									validator.WatchersIdentity,
									1*time.Hour,
								)
								if len(sendToWatchers) > 0 {
									enqueueTelegramMessageByIdentity(
										valoperAddr,
										fmt.Sprintf("%s is TOMBSTONED! Contact to unsubscribe this validator", moniker),
										true,
										sendToWatchers...,
									)
								}
								bTrue := true
								cacheHc.TomeStoned = &bTrue
							} else if now := time.Now().UTC(); signingInfo.JailedUntil.After(now) {
								sendToWatchers := tpsvc.ShouldSendMessageWL(
									tpsvc.PreventSpammingCaseJailed,
									validator.WatchersIdentity,
									30*time.Minute,
								)
								if len(sendToWatchers) > 0 {
									enqueueTelegramMessageByIdentity(
										valoperAddr,
										fmt.Sprintf("%s was Jailed until %s, %f minutes left", moniker, signingInfo.JailedUntil, signingInfo.JailedUntil.Sub(now).Minutes()),
										true,
										sendToWatchers...,
									)
								}
								bTrue := true
								cacheHc.Jailed = &bTrue
								cacheHc.JailedUntil = &signingInfo.JailedUntil
							} else {
								if signingInfo.MissedBlocksCounter > 0 {
									if slashingParams != nil {
										if slashingParams.MinSignedPerWindow.IsPositive() && slashingParams.SignedBlocksWindow > 0 {
											var downtimeSlashingWhenMissedExcess int64
											if slashingParams.MinSignedPerWindow.Equal(sdk.OneDec()) {
												downtimeSlashingWhenMissedExcess = 0
											} else {
												downtimeSlashingWhenMissedExcess =
													slashingParams.SignedBlocksWindow - slashingParams.MinSignedPerWindow.Mul(sdk.NewDec(slashingParams.SignedBlocksWindow)).Ceil().RoundInt64()
											}
											cacheHc.DowntimeSlashingWhenMissedExcess = &downtimeSlashingWhenMissedExcess

											missedBlocksOverDowntimeSlashingRatio := utils.RatioOfInt64(signingInfo.MissedBlocksCounter, downtimeSlashingWhenMissedExcess)
											if missedBlocksOverDowntimeSlashingRatio > 50.0 {
												sendToWatchers := tpsvc.ShouldSendMessageWL(
													tpsvc.PreventSpammingCaseMissedBlocksOverDangerousThreshold,
													validator.WatchersIdentity,
													15*time.Minute,
												)
												if len(sendToWatchers) > 0 {
													enqueueTelegramMessageByIdentity(
														valoperAddr,
														fmt.Sprintf(
															"%s has missed more than half of the allowed blocks in the window, beware of being Jailed. Missed %d/%d, ratio %f%%, window %d blocks",
															moniker,
															signingInfo.MissedBlocksCounter,
															downtimeSlashingWhenMissedExcess,
															missedBlocksOverDowntimeSlashingRatio,
															slashingParams.SignedBlocksWindow,
														),
														true,
														sendToWatchers...,
													)
												}
											} else if missedBlocksOverDowntimeSlashingRatio > 10.0 {
												sendToWatchers := tpsvc.ShouldSendMessageWL(
													tpsvc.PreventSpammingCaseMissedBlocksOverDangerousThreshold,
													validator.WatchersIdentity,
													2*time.Hour,
												)
												if len(sendToWatchers) > 0 {
													enqueueTelegramMessageByIdentity(
														valoperAddr,
														fmt.Sprintf(
															"%s has high missed-block-ratio. Missed %d/%d, ratio %f%%, window %d blocks",
															moniker,
															signingInfo.MissedBlocksCounter,
															downtimeSlashingWhenMissedExcess,
															missedBlocksOverDowntimeSlashingRatio,
															slashingParams.SignedBlocksWindow,
														),
														false,
														sendToWatchers...,
													)
												}
											}

											uptime := 100.0 - utils.RatioOfInt64(signingInfo.MissedBlocksCounter, slashingParams.SignedBlocksWindow)
											if uptime <= 90.0 {
												var ignoreIfLastSentLessThan time.Duration
												if uptime <= 65.0 {
													ignoreIfLastSentLessThan = 15 * time.Minute
												} else if uptime <= 75.0 {
													ignoreIfLastSentLessThan = 30 * time.Minute
												} else {
													ignoreIfLastSentLessThan = 1 * time.Hour
												}
												sendToWatchers := tpsvc.ShouldSendMessageWL(
													tpsvc.PreventSpammingCaseLowUptime,
													validator.WatchersIdentity,
													ignoreIfLastSentLessThan,
												)
												fatal := uptime <= 70.0
												if len(sendToWatchers) > 0 {
													enqueueTelegramMessageByIdentity(
														valoperAddr,
														fmt.Sprintf("%s has low uptime %f%%", moniker, uptime),
														fatal,
														sendToWatchers...,
													)
												}
											}
											cacheHc.Uptime = &uptime

											logger.Debug(
												"validator health-check information",
												"uptime", fmt.Sprintf("%f%%", uptime),
												"missed-block", fmt.Sprintf("%d/%d", signingInfo.MissedBlocksCounter, downtimeSlashingWhenMissedExcess),
												"valoper", valoperAddr,
												"chain", chainName,
											)
										}
									} else {
										enqueueTelegramMessageByIdentity(
											valoperAddr,
											fmt.Sprintf("skipped uptime health-check for %s because missing slashing params", moniker),
											false,
											validator.WatchersIdentity...,
										)
									}
									cacheHc.MissedBlockCount = &signingInfo.MissedBlocksCounter
								} else {
									logger.Debug("no missed block", "chain", chainName, "valoper", valoperAddr, "signing-info", signingInfo)
								}
							}
						} else {
							enqueueTelegramMessageByIdentity(
								valoperAddr,
								fmt.Sprintf("validator %s signing info could not be found, valcons: %s", moniker, valconsAddr),
								false,
								validator.WatchersIdentity...,
							)
							logger.Debug("validator signing info could not be found", "chain", chainName, "valcons", valconsAddr, "valoper", valoperAddr, "signing-info-size", len(signingInfos))
						}
					} else {
						enqueueTelegramMessageByIdentity(
							valoperAddr,
							fmt.Sprintf("validator %s consensus address not found in mapping", moniker),
							false,
							validator.WatchersIdentity...,
						)
					}
				}

				if validator.OptionalHealthCheckRPC != "" {
					func(validator chainreg.ValidatorOfRegisteredChainConfig, valoperAddr string) {
						var errorToReport error
						var fatal bool
						ignoreIfLastSentLessThan := 15 * time.Minute

						defer func() {
							if errorToReport != nil {
								sendToWatchers := tpsvc.ShouldSendMessageWL(
									tpsvc.PreventSpammingCaseDirectHealthCheckOptionalRPC,
									validator.WatchersIdentity,
									ignoreIfLastSentLessThan,
								)
								if len(sendToWatchers) > 0 {
									enqueueTelegramMessageByIdentity(
										valoperAddr,
										errorToReport.Error(),
										fatal,
										sendToWatchers...,
									)
								}
							}
						}()

						rpcClient, err := rpcreg.GetRpcClientByEndpointWL(validator.OptionalHealthCheckRPC, logger)
						if err != nil {
							errorToReport = errors.Wrapf(err, "failed to get RPC client to direct health-check validator %s: %s", moniker, validator.OptionalHealthCheckRPC)
						} else {
							resultStatus, err := utils.Retry(func() (*coretypes.ResultStatus, error) {
								return rpcClient.GetWebsocketClient().Status(context.Background())
							})
							if err != nil {
								errorToReport = errors.Wrapf(err, "failed to get status from direct health-check validator %s: %s", moniker, validator.OptionalHealthCheckRPC)
							} else {
								if resultStatus.SyncInfo.CatchingUp {
									errorToReport = fmt.Errorf("validator %s is catching up, block %d, time %v", moniker, resultStatus.SyncInfo.LatestBlockHeight, resultStatus.SyncInfo.LatestBlockTime)
									fatal = true
								} else if diff := time.Since(resultStatus.SyncInfo.LatestBlockTime.UTC()); diff > 30*time.Second {
									errorToReport = fmt.Errorf("validator %s is out dated %d, time %v, server time %v", moniker, int64(diff.Seconds()), resultStatus.SyncInfo.LatestBlockTime, time.Now().UTC())
									ignoreIfLastSentLessThan = 10 * time.Minute
									fatal = true
								}
							}
						}
					}(validator, valoperAddr)
				}

				putCacheValidatorHealthCheckWL(cacheHc)
			}

			// health-check managed RPCs
			if len(registeredChainConfig.GetHealthCheckRPCs()) > 0 {
				rootUsersIdentity := usereg.GetRootUsersIdentityRL()
				rootUsersIdentityWatchingThisChain := utils.Collisions(rootUsersIdentity, allWatchersIdentity)
				if len(rootUsersIdentityWatchingThisChain) == 0 {
					logger.Info("no root user watching this chain to report, skipping health-check managed RPCs", "chain", chainName)
				} else {
					for _, managedRPC := range registeredChainConfig.GetHealthCheckRPCs() {
						func(managedRPC string, rootUsersIdentityWatchingThisChain []string) {
							var errorToReport error

							defer func() {
								if errorToReport != nil {
									logger.Error("health-check managed RPC failed", "chain", chainName, "managed_rpc", managedRPC, "error", errorToReport.Error())
									sendToWatchers := tpsvc.ShouldSendMessageWL(
										tpsvc.PreventSpammingCaseHealthCheckManagedRPC,
										rootUsersIdentityWatchingThisChain,
										30*time.Minute,
									)
									if len(sendToWatchers) > 0 {
										enqueueTelegramMessageByIdentity(
											"",
											errorToReport.Error(),
											false,
											sendToWatchers...,
										)
									}
								}
							}()

							rpcClient, err := rpcreg.GetRpcClientByEndpointWL(managedRPC, logger)
							if err != nil {
								errorToReport = errors.Wrapf(err, "failed to get RPC client to health-check managed RPC %s", managedRPC)
								return
							}

							resultStatus, err := utils.Retry(func() (*coretypes.ResultStatus, error) {
								return rpcClient.GetWebsocketClient().Status(context.Background())
							})
							if err != nil {
								errorToReport = errors.Wrapf(err, "failed to get status for health-check managed RPC %s", managedRPC)
								return
							}

							if resultStatus.SyncInfo.CatchingUp {
								errorToReport = fmt.Errorf("managed RPC node is catching up, block %d, time %v, RPC %s", resultStatus.SyncInfo.LatestBlockHeight, resultStatus.SyncInfo.LatestBlockTime, managedRPC)
								return
							}

							if diff := time.Since(resultStatus.SyncInfo.LatestBlockTime.UTC()); diff >= 180*time.Second {
								errorToReport = fmt.Errorf("managed RPC node is out dated %d, time %v, server time %v, RPC %s", int64(diff.Seconds()), resultStatus.SyncInfo.LatestBlockTime, time.Now().UTC(), managedRPC)
							}
						}(managedRPC, rootUsersIdentityWatchingThisChain)
					}
				}
			}
		}(registeredChainConfig)
	}
}

func (w Worker) reloadMappingValAddressIfNeeded(registeredChainConfig chainreg.RegisteredChainConfig, stakingValidators []stakingtypes.Validator) {
	logger := w.ctx.AppCtx.Logger

	chainName := registeredChainConfig.GetChainName()
	validators := registeredChainConfig.GetValidators()
	var reloadMappingValAddress bool
	for _, validator := range validators {
		_, found := valaddreg.GetValconsByValoperRL(chainName, validator.ValidatorOperatorAddress)
		if !found {
			logger.Info("validator not found in mapping, going to reload validator address mapping", "chain", chainName, "valoper", validator.ValidatorOperatorAddress)
			reloadMappingValAddress = true
			break
		}
	}

	if !reloadMappingValAddress {
		return
	}

	for _, validator := range stakingValidators {
		consAddr, success := utils.FromAnyPubKeyToConsensusAddress(validator.ConsensusPubkey, codec.CryptoCodec)
		if !success {
			logger.Error("failed to convert pubkey to consensus address", "chain", chainName, "valoper", validator.OperatorAddress, "consensus_pubkey", validator.ConsensusPubkey)
			continue
		}

		valconsHrp, success := utils.GetValconsHrpFromValoperHrp(validator.OperatorAddress)
		if !success {
			panic(fmt.Sprintf("failed to get valcons hrp from valoper hrp, weird! valoper: %s", validator.OperatorAddress))
		}

		valconsAddrStr, err := sdk.Bech32ifyAddressBytes(valconsHrp, consAddr.Bytes())
		if err != nil {
			logger.Error("failed to bech32ify consensus address", "hrp", valconsHrp, "chain", chainName, "valoper", validator.OperatorAddress, "consensus_address", strings.ToUpper(hex.EncodeToString(consAddr)))
			continue
		}

		valaddreg.RegisterPairValAddressWL(chainName, validator.OperatorAddress, valconsAddrStr)
	}
}

func getAllValidators(rpcClient rpcreg.RpcClient) ([]stakingtypes.Validator, error) {
	const limit uint64 = 200 // luckily, this endpoint support large page size. 500 is no problem.

	var stakingValidators []stakingtypes.Validator
	var nextKey []byte
	var stop = false
	page := 1

	for !stop {
		req := stakingtypes.QueryValidatorsRequest{
			Pagination: &query.PageRequest{
				Limit: limit,
				Key:   nextKey,
			},
		}

		bz, err := req.Marshal()
		if err != nil {
			panic(errors.Wrap(err, "failed to marshal request, weird!"))
		}

		queryValidatorsResponse, err := utils.Retry[*stakingtypes.QueryValidatorsResponse](func() (*stakingtypes.QueryValidatorsResponse, error) {
			resultABCIQuery, err := rpcClient.GetWebsocketClient().ABCIQuery(context.Background(), "/cosmos.staking.v1beta1.Query/Validators", bz)
			if err != nil {
				return nil, err
			}

			if len(resultABCIQuery.Response.Value) == 0 {
				return nil, fmt.Errorf("empty response value, weird")
			}

			queryValidatorsResponse := &stakingtypes.QueryValidatorsResponse{}
			err = queryValidatorsResponse.Unmarshal(resultABCIQuery.Response.Value)
			if err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal response, weird!")
			}

			return queryValidatorsResponse, nil
		})

		if err != nil {
			return nil, errors.Wrap(err, "failed to query validators")
		}

		nextKey = queryValidatorsResponse.Pagination.NextKey
		stop = len(queryValidatorsResponse.Pagination.NextKey) == 0
		stakingValidators = append(stakingValidators, queryValidatorsResponse.Validators...)
		page++
	}

	return stakingValidators, nil
}

func getAllSigningInfos(rpcClient rpcreg.RpcClient) ([]slashingtypes.ValidatorSigningInfo, error) {
	const limit uint64 = 200 // luckily, this endpoint support large page size. 500 is no problem.

	var validatorSigningInfos []slashingtypes.ValidatorSigningInfo
	var nextKey []byte
	var stop = false
	page := 1

	for !stop {
		req := slashingtypes.QuerySigningInfosRequest{
			Pagination: &query.PageRequest{
				Limit: limit,
				Key:   nextKey,
			},
		}

		bz, err := req.Marshal()
		if err != nil {
			panic(errors.Wrap(err, "failed to marshal request, weird!"))
		}

		querySigningInfosResponse, err := utils.Retry[*slashingtypes.QuerySigningInfosResponse](func() (*slashingtypes.QuerySigningInfosResponse, error) {
			resultABCIQuery, err := rpcClient.GetWebsocketClient().ABCIQuery(context.Background(), "/cosmos.slashing.v1beta1.Query/SigningInfos", bz)
			if err != nil {
				return nil, err
			}

			if len(resultABCIQuery.Response.Value) == 0 {
				return nil, fmt.Errorf("empty response value, weird")
			}

			querySigningInfosResponse := &slashingtypes.QuerySigningInfosResponse{}
			err = querySigningInfosResponse.Unmarshal(resultABCIQuery.Response.Value)
			if err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal response, weird!")
			}

			return querySigningInfosResponse, nil
		})

		if err != nil {
			return nil, errors.Wrap(err, "failed to query validators signing infos")
		}

		nextKey = querySigningInfosResponse.Pagination.NextKey
		stop = len(querySigningInfosResponse.Pagination.NextKey) == 0
		validatorSigningInfos = append(validatorSigningInfos, querySigningInfosResponse.Info...)
		page++
	}

	return validatorSigningInfos, nil
}

func getSlashingParams(rpcClient rpcreg.RpcClient) (*slashingtypes.Params, error) {
	req := slashingtypes.QueryParamsRequest{}

	bz, err := req.Marshal()
	if err != nil {
		panic(errors.Wrap(err, "failed to marshal request, weird!"))
	}

	querySigningInfosResponse, err := utils.Retry[*slashingtypes.QueryParamsResponse](func() (*slashingtypes.QueryParamsResponse, error) {
		resultABCIQuery, err := rpcClient.GetWebsocketClient().ABCIQuery(context.Background(), "/cosmos.slashing.v1beta1.Query/Params", bz)
		if err != nil {
			return nil, err
		}

		if len(resultABCIQuery.Response.Value) == 0 {
			return nil, fmt.Errorf("empty response value, weird")
		}

		queryParamResponse := &slashingtypes.QueryParamsResponse{}
		err = queryParamResponse.Unmarshal(resultABCIQuery.Response.Value)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal response, weird!")
		}

		return queryParamResponse, nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to query slashing params")
	}

	if querySigningInfosResponse == nil {
		return nil, errors.New("empty response, weird!")
	}

	return &querySigningInfosResponse.Params, nil
}

func getMostHealthyRpc(rpc []string, chainId string, logger logging.Logger) (rpcreg.RpcClient, string, time.Time, error) {
	if len(rpc) == 0 {
		panic("no rpc to health-check")
	}
	type scoredRPC struct {
		latestBlock     int64
		latestBlockTime time.Time
		endpoint        string
		err             error
	}
	chanScoredRPC := make(chan scoredRPC, len(rpc))
	for _, r := range rpc {
		go func(r string) {
			scoredRPC := scoredRPC{
				endpoint: r,
			}
			defer func() {
				r := recover()
				if r != nil {
					scoredRPC.err = fmt.Errorf("panic: %v", r)
				}

				chanScoredRPC <- scoredRPC
			}()

			rpcClient, err := rpcreg.GetRpcClientByEndpointWL(r, logger)
			if err != nil {
				scoredRPC.err = errors.Wrap(err, "failed to get RPC client")
				return
			}

			wsClient := rpcClient.GetWebsocketClient()
			if wsClient == nil {
				scoredRPC.err = errors.New("websocket client is nil")
				return
			}

			resultStatus, err := utils.Retry(func() (*coretypes.ResultStatus, error) {
				return wsClient.Status(context.Background())
			})

			if err != nil {
				scoredRPC.err = errors.Wrap(err, "failed to get status")
				return
			}

			if resultStatus.NodeInfo.Network != chainId {
				scoredRPC.err = fmt.Errorf("network mismatch, expected %s, got %s", chainId, resultStatus.NodeInfo.Network)
				return
			}

			scoredRPC.latestBlock = resultStatus.SyncInfo.LatestBlockHeight
			scoredRPC.latestBlockTime = resultStatus.SyncInfo.LatestBlockTime
			return
		}(r)
	}

	var scoredRPCs = make([]scoredRPC, len(rpc))
	for i := 0; i < len(rpc); i++ {
		scoredRPC := <-chanScoredRPC
		scoredRPCs[i] = scoredRPC
	}

	sort.Slice(scoredRPCs, func(i, j int) bool {
		left := scoredRPCs[i]
		right := scoredRPCs[j]

		if left.err != nil && right.err != nil {
			return left.latestBlock > right.latestBlock
		}

		if left.err != nil {
			return false
		}

		if right.err != nil {
			return true
		}

		return left.latestBlock > right.latestBlock
	})

	mostHealthyRPC := scoredRPCs[0]
	if mostHealthyRPC.err != nil {
		return nil, "", time.Time{}, mostHealthyRPC.err
	}

	rpcClient, err := rpcreg.GetRpcClientByEndpointWL(mostHealthyRPC.endpoint, logger)
	if err != nil {
		return nil, "", time.Time{}, errors.Wrap(err, "failed to get RPC client of the most healthy RPC")
	}

	return rpcClient, mostHealthyRPC.endpoint, mostHealthyRPC.latestBlockTime, nil
}
