package chain_registry

import (
	"github.com/bcdevtools/validator-health-check/config"
	"strings"
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
	GetLastHealthCheckUtcRL() time.Time
	SetLastHealthCheckUtcWL()
}

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
	lastHealthCheckUtc time.Time
}

func newRegisteredChainConfig(chainConfig config.ChainConfig) RegisteredChainConfig {
	normalizeRPC := func(rpc string) string {
		if rpc != "" {
			rpc = strings.TrimSuffix(rpc, "/")
		}
		return rpc
	}

	return &registeredChainConfig{
		chainName: chainConfig.ChainName,
		chainId:   chainConfig.ChainId,
		priority:  chainConfig.Priority,
		rpc: func(rpc ...string) []string {
			normalizedRPCs := make([]string, len(rpc))
			for i, r := range rpc {
				normalizedRPCs[i] = normalizeRPC(r)
			}
			return normalizedRPCs
		}(),
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
