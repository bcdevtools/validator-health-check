package health_check_worker

import (
	"context"
	"fmt"
	libapp "github.com/EscanBE/go-lib/app"
	"github.com/EscanBE/go-lib/logging"
	"github.com/bcdevtools/validator-health-check/codec"
	"github.com/bcdevtools/validator-health-check/config"
	"github.com/bcdevtools/validator-health-check/constants"
	chainreg "github.com/bcdevtools/validator-health-check/registry/chain_registry"
	rpcreg "github.com/bcdevtools/validator-health-check/registry/rpc_client_registry"
	"github.com/bcdevtools/validator-health-check/registry/user_registry"
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
	"time"
)

// Worker represents for a worker, itself holds things needed for doing business logic, especially its own `HcwContext`
type Worker struct {
	ctx               *workertypes.HcwContext
	telegramPusherSvc tpsvc.TelegramPusher
}

// NewHcWorker creates new health-check worker
func NewHcWorker(wCtx *workertypes.HcwContext, telegramPusherSvc tpsvc.TelegramPusher) Worker {
	return Worker{
		ctx:               wCtx,
		telegramPusherSvc: telegramPusherSvc,
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
				if len(validator.WatchersIdentity) == 0 {
					continue
				}
				for _, identity := range validator.WatchersIdentity {
					userRecord, found := user_registry.GetUserRecordByIdentityRL(identity)
					if !found {
						panic(fmt.Sprintf("user not found, weird! identity: %s", identity))
					}
					if userRecord.TelegramConfig.IsEmptyOrIncompleteConfig() {
						panic(fmt.Sprintf("telegram config is empty or incomplete, weird! identity: %s", identity))
					}
					allWatchersIdentity = append(allWatchersIdentity, identity)
					watchersIdentityToUserRecord[identity] = userRecord
				}
			}

			enqueueTelegramMessageByIdentity := func(message string, identities ...string) {
				for _, identity := range identities {
					userRecord, found := watchersIdentityToUserRecord[identity]
					if !found {
						panic(fmt.Sprintf("user not found, weird! identity: %s", identity))
					}
					w.telegramPusherSvc.EnqueueMessageWL(tptypes.QueueMessage{
						ReceiverID: userRecord.TelegramConfig.UserId,
						Priority:   userRecord.Root,
						Message:    message,
					})
				}
			}

			chainName := registeredChainConfig.GetChainName()
			logger.Debug("health-checking chain", "chain", chainName, "wid", w.ctx.WorkerID)

			var healthCheckError error
			defer func() {
				if healthCheckError == nil {
					logger.Debug("health-check successfully")
					return
				}

				logger.Error("failed to health-check chain", "chain", chainName, "error", healthCheckError.Error())
				enqueueTelegramMessageByIdentity(
					fmt.Sprintf("failed to health-check chain %s, error: %s", chainName, healthCheckError.Error()),
					allWatchersIdentity...,
				)
			}()

			// get the most healthy RPC
			rpcClient, mostHealthyEndpoint, latestBlockTime, errFetchSigningInfo := getMostHealthyRpc(registeredChainConfig.GetRPCs(), registeredChainConfig.GetChainId(), logger)
			if errFetchSigningInfo != nil {
				healthCheckError = errors.Wrap(errFetchSigningInfo, "failed to get most healthy RPC")
				return
			}

			logger.Debug("most healthy RPC", "chain", chainName, "endpoint", mostHealthyEndpoint, "latest_block_time", latestBlockTime)
			if time.Since(latestBlockTime) > constants.INFORM_TELEGRAM_IF_BLOCK_OLDER_THAN {
				enqueueTelegramMessageByIdentity(
					fmt.Sprintf("latest block time of the most health RPC of %s is too old: %s, diff %s", chainName, latestBlockTime, time.Since(latestBlockTime)),
					allWatchersIdentity...,
				)
			}

			registeredChainConfig.InformPriorityLatestHealthyRpcWL(mostHealthyEndpoint)

			// fetch all validators
			validators, errFetchSigningInfo := getAllValidators(rpcClient)
			if errFetchSigningInfo != nil {
				healthCheckError = errors.Wrap(errFetchSigningInfo, "failed to get all validators")
				return
			}
			stakingValidatorByValoper := make(map[string]stakingtypes.Validator)
			for _, validator := range validators {
				stakingValidatorByValoper[validator.OperatorAddress] = validator
			}

			// reload mapping
			w.reloadMappingValAddressIfNeeded(registeredChainConfig, validators)

			// fetch all signingInfos
			signingInfos, errFetchSigningInfo := getAllSigningInfos(rpcClient)
			if errFetchSigningInfo != nil {
				enqueueTelegramMessageByIdentity(
					fmt.Sprintf("failed to get all validator signing infos on %s, for uptime-check, error: %s", chainName, errFetchSigningInfo.Error()),
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
					fmt.Sprintf("failed to get all slashing params on %s, for uptime-check, error: %s", chainName, errFetchSlashingParams.Error()),
					allWatchersIdentity...,
				)
			}

			// health-check each validator

			for _, validator := range registeredChainConfig.GetValidators() {
				valoperAddr := validator.ValidatorOperatorAddress
				stakingValidator, found := stakingValidatorByValoper[valoperAddr]
				if !found {
					enqueueTelegramMessageByIdentity(fmt.Sprintf("validator %s not found, chain %s", valoperAddr, chainName), validator.WatchersIdentity...)
					continue
				}

				switch stakingValidator.Status {
				case stakingtypes.Bonded:
					// all good
				case stakingtypes.Unbonded:
					enqueueTelegramMessageByIdentity(fmt.Sprintf("validator %s on %s is unbonded! Tombstoned? Contact to unsubscribe this validator", valoperAddr, chainName), validator.WatchersIdentity...)
				case stakingtypes.Unbonding:
					enqueueTelegramMessageByIdentity(fmt.Sprintf("validator %s on %s is unbonding! Out of active set? Jailed?", valoperAddr, chainName), validator.WatchersIdentity...)
				default:
					enqueueTelegramMessageByIdentity(fmt.Sprintf("validator %s on %s is in unknown status %s", valoperAddr, chainName, stakingValidator.Status), validator.WatchersIdentity...)
				}

				// TODO health-check slashing
				if errFetchSigningInfo == nil { // skip check if error on fetch, error message informed before
					valconsAddr, found := valaddreg.GetValconsByValoperRL(chainName, valoperAddr)
					if found {
						signingInfo, found := valconsToSigningInfo[valconsAddr]
						if found {
							if signingInfo.Tombstoned {
								enqueueTelegramMessageByIdentity(
									fmt.Sprintf("FATAL: validator %s on %s is tombstoned! Contact to unsubscribe this validator", valoperAddr, chainName),
									validator.WatchersIdentity...,
								)
							} else if now := time.Now().UTC(); signingInfo.JailedUntil.After(now) {
								enqueueTelegramMessageByIdentity(
									fmt.Sprintf("FATAL: validator %s on %s is jailed until %s, %f minutes left", valoperAddr, chainName, signingInfo.JailedUntil, signingInfo.JailedUntil.Sub(now).Minutes()),
									validator.WatchersIdentity...,
								)
							} else if signingInfo.MissedBlocksCounter > 0 {
								if slashingParams != nil {
									if slashingParams.MinSignedPerWindow.IsPositive() {
										uptime := sdk.NewDec(signingInfo.MissedBlocksCounter).Mul(sdk.NewDec(100)).Quo(slashingParams.MinSignedPerWindow).RoundInt().Int64()
										if uptime <= 90 {
											enqueueTelegramMessageByIdentity(
												fmt.Sprintf("validator %s on %s low uptime %d%", valoperAddr, chainName, uptime),
												validator.WatchersIdentity...,
											)
										}

										if signingInfo.MissedBlocksCounter > slashingParams.MinSignedPerWindow.RoundInt64()/2 {
											enqueueTelegramMessageByIdentity(
												fmt.Sprintf("FATAL: validator %s on %s missed more than half of the blocks in the window, beware of being Jailed", valoperAddr, chainName),
												validator.WatchersIdentity...,
											)
										}

										logger.Debug(
											"validator health-check information",
											"uptime", fmt.Sprintf("%d%%", uptime),
											"missed-block", fmt.Sprintf("%d/%d", signingInfo.MissedBlocksCounter, slashingParams.MinSignedPerWindow.RoundInt64()),
											"valoper", valoperAddr,
											"chain", chainName,
										)
									}
								} else {
									enqueueTelegramMessageByIdentity(
										fmt.Sprintf("skipped uptime health-check for %s on %s because missing slashing params", valoperAddr, chainName),
										validator.WatchersIdentity...,
									)
								}
							}
						} else {
							enqueueTelegramMessageByIdentity(
								fmt.Sprintf("validator signing info of %s on %s not found from result", valoperAddr, chainName),
								validator.WatchersIdentity...,
							)
						}
					} else {
						enqueueTelegramMessageByIdentity(
							fmt.Sprintf("validator consensus address of %s on %s not found in mapping", valoperAddr, chainName),
							validator.WatchersIdentity...,
						)
					}
				}

				// TODO health-check the optional RPC
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

		valaddreg.RegisterPairValAddressWL(chainName, validator.OperatorAddress, consAddr.String())
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