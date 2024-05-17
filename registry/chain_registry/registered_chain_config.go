package chain_registry

import (
	"github.com/bcdevtools/validator-health-check/config"
	"github.com/bcdevtools/validator-health-check/utils"
	"sort"
	"sync"
	"time"
)

type RegisteredChainConfig interface {
	GetChainName() string
	GetChainId() string
	IsPriority() bool
	GetRPCs() []string
	InformPriorityLatestHealthyRpcWL(string)
	GetValidators() []ValidatorOfRegisteredChainConfig
	GetHealthCheckRPCs() []string
	GetLastHealthCheckUtcRL() time.Time
	SetLastHealthCheckUtcWL()
}

type RegisteredChainsConfig []RegisteredChainConfig

var _ RegisteredChainConfig = &registeredChainConfig{}

type ValidatorOfRegisteredChainConfig struct {
	ValidatorOperatorAddress string
	WatchersIdentity         []string
	OptionalHealthCheckRPC   string
}

type registeredChainConfig struct {
	sync.RWMutex
	chainName          string
	chainId            string
	priority           bool
	rpc                []string
	validators         []ValidatorOfRegisteredChainConfig
	healthCheckRPC     []string
	lastHealthCheckUtc time.Time
}

func newRegisteredChainConfig(chainConfig config.ChainConfig) RegisteredChainConfig {
	normalizeRPC := func(rpc string) string {
		if rpc != "" {
			rpc = utils.ReplaceAnySchemeWithHttp(rpc)
			rpc = utils.NormalizeRpcEndpoint(rpc)
		}
		return rpc
	}
	normalizeRPCs := func(rpc ...string) []string {
		normalizedRPCs := make([]string, len(rpc))
		for i, r := range rpc {
			normalizedRPCs[i] = normalizeRPC(r)
		}
		return normalizedRPCs
	}

	return &registeredChainConfig{
		chainName: chainConfig.ChainName,
		chainId:   chainConfig.ChainId,
		priority:  chainConfig.Priority,
		rpc:       normalizeRPCs(utils.Distinct[string](append(chainConfig.RPCs, chainConfig.HealthCheckRPC...)...)...),
		validators: func() []ValidatorOfRegisteredChainConfig {
			var validators []ValidatorOfRegisteredChainConfig
			for _, chainValidatorConfig := range chainConfig.Validators {
				validators = append(validators, ValidatorOfRegisteredChainConfig{
					ValidatorOperatorAddress: chainValidatorConfig.ValidatorOperatorAddress,
					WatchersIdentity:         chainValidatorConfig.Watchers,
					OptionalHealthCheckRPC:   normalizeRPC(chainValidatorConfig.OptionalHealthCheckRPC),
				})
			}
			return validators
		}(),
		healthCheckRPC: normalizeRPCs(chainConfig.HealthCheckRPC...),
	}
}

func (r *registeredChainConfig) GetChainName() string {
	return r.chainName
}

func (r *registeredChainConfig) GetChainId() string {
	return r.chainId
}

func (r *registeredChainConfig) IsPriority() bool {
	return r.priority
}

func (r *registeredChainConfig) GetRPCs() []string {
	return r.rpc[:]
}

func (r *registeredChainConfig) InformPriorityLatestHealthyRpcWL(rpc string) {
	if len(r.rpc) == 0 {
		return
	}

	r.Lock()
	defer r.Unlock()

	reorderedRpc := []string{rpc}
	for _, r := range r.rpc {
		if r == rpc {
			continue
		}
		reorderedRpc = append(reorderedRpc, r)
	}

	r.rpc = reorderedRpc
}

func (r *registeredChainConfig) GetValidators() []ValidatorOfRegisteredChainConfig {
	return r.validators[:]
}

func (r *registeredChainConfig) GetHealthCheckRPCs() []string {
	return r.healthCheckRPC
}

func (r *registeredChainConfig) GetLastHealthCheckUtcRL() time.Time {
	r.RLock()
	defer r.RUnlock()

	return r.lastHealthCheckUtc
}

func (r *registeredChainConfig) SetLastHealthCheckUtcWL() {
	r.Lock()
	defer r.Unlock()

	r.lastHealthCheckUtc = time.Now().UTC()
}

func (rs RegisteredChainsConfig) Sort() RegisteredChainsConfig {
	sort.Slice(rs, func(i, j int) bool {
		left := rs[i]
		right := rs[j]

		if left.IsPriority() == right.IsPriority() {
			return left.GetChainName() < right.GetChainName()
		}

		if left.IsPriority() {
			return true
		} else {
			return false
		}
	})
	return rs
}
