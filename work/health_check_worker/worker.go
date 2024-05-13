package health_check_worker

import (
	"context"
	"fmt"
	libapp "github.com/EscanBE/go-lib/app"
	"github.com/EscanBE/go-lib/logging"
	"github.com/bcdevtools/validator-health-check/codec"
	chainreg "github.com/bcdevtools/validator-health-check/registry/chain_registry"
	rpcreg "github.com/bcdevtools/validator-health-check/registry/rpc_client_registry"
	valaddreg "github.com/bcdevtools/validator-health-check/registry/validator_address_registry"
	"github.com/bcdevtools/validator-health-check/utils"
	workertypes "github.com/bcdevtools/validator-health-check/work/health_check_worker/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/pkg/errors"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"sort"
	"time"
)

// Worker represents for a worker, itself holds things needed for doing business logic, especially its own `HcwContext`
type Worker struct {
	Ctx *workertypes.HcwContext
}

// NewHcWorker creates new health-check worker
func NewHcWorker(wCtx *workertypes.HcwContext) Worker {
	return Worker{
		Ctx: wCtx,
	}
}

// Start performs business logic of worker
func (w Worker) Start() {
	logger := w.Ctx.AppCtx.Logger
	defer libapp.TryRecoverAndExecuteExitFunctionIfRecovered(logger)

	healthCheckInterval := w.Ctx.AppCtx.AppConfig.General.HealthCheckInterval
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
			var healthCheckError error
			defer func() {
				if healthCheckError == nil {
					return
				}

				logger.Error("failed to health-check chain", "chain", registeredChainConfig.GetChainName(), "error", healthCheckError.Error())
				// TODO inform telegram
			}()

			chainName := registeredChainConfig.GetChainName()
			logger.Debug("health-checking chain", "chain", chainName, "wid", w.Ctx.WorkerID)

			// get the most healthy RPC
			rpcClient, mostHealthyEndpoint, err := getMostHealthyRpc(registeredChainConfig.GetRPCs(), registeredChainConfig.GetChainId(), logger)
			if err != nil {
				healthCheckError = errors.Wrap(err, "failed to get most healthy RPC")
				return
			}

			registeredChainConfig.InformPriorityLatestHealthyRpcWL(mostHealthyEndpoint)

			err = w.reloadMappingValAddressIfNeeded(registeredChainConfig, rpcClient)
			if err != nil {
				healthCheckError = errors.Wrap(err, "failed to reload mapping validator address")
				return
			}

			logger.Debug("health-check successfully")
		}(registeredChainConfig)
	}
}

func (w Worker) reloadMappingValAddressIfNeeded(registeredChainConfig chainreg.RegisteredChainConfig, rpcClient rpcreg.RpcClient) error {
	logger := w.Ctx.AppCtx.Logger

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
		return nil
	}

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

		var resultABCIQuery *coretypes.ResultABCIQuery

		queryValidatorsResponse, err := utils.Retry[*stakingtypes.QueryValidatorsResponse](func() (*stakingtypes.QueryValidatorsResponse, error) {
			resultABCIQuery, err = rpcClient.GetWebsocketClient().ABCIQuery(context.Background(), "/cosmos.staking.v1beta1.Query/Validators", bz)
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
			return errors.Wrap(err, "failed to query validators")
		}

		nextKey = queryValidatorsResponse.Pagination.NextKey
		stop = len(queryValidatorsResponse.Pagination.NextKey) == 0
		stakingValidators = append(stakingValidators, queryValidatorsResponse.Validators...)
		page++
	}

	for _, validator := range stakingValidators {
		consAddr, success := utils.FromAnyPubKeyToConsensusAddress(validator.ConsensusPubkey, codec.CryptoCodec)
		if !success {
			logger.Info("failed to convert pubkey to consensus address", "chain", chainName, "valoper", validator.OperatorAddress, "consensus_pubkey", validator.ConsensusPubkey)
			continue
		}

		valaddreg.RegisterPairValAddressWL(chainName, validator.OperatorAddress, consAddr.String())
	}

	return nil
}

func getMostHealthyRpc(rpc []string, chainId string, logger logging.Logger) (rpcreg.RpcClient, string, error) {
	if len(rpc) == 0 {
		panic("no rpc to health-check")
	}
	type scoredRPC struct {
		latestBlock int64
		endpoint    string
		err         error
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
		return nil, "", mostHealthyRPC.err
	}

	rpcClient, err := rpcreg.GetRpcClientByEndpointWL(mostHealthyRPC.endpoint, logger)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to get RPC client of the most healthy RPC")
	}

	return rpcClient, mostHealthyRPC.endpoint, nil
}
