package chain_registry

import (
	"github.com/bcdevtools/validator-health-check/config"
	"sync"
	"time"
)

type RegisteredChainConfig interface {
	GetChainName() string
	IsPriority() bool
	GetRPCs() []string
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
	priority           bool
	rpc                []string
	validators         []ValidatorOfRegisteredChainConfig
	lastHealthCheckUtc time.Time
}

func newRegisteredChainConfig(chainConfig config.ChainConfig) RegisteredChainConfig {
	return &registeredChainConfig{
		chainName: chainConfig.ChainName,
		priority:  chainConfig.Priority,
		rpc:       chainConfig.RPCs,
		validators: func() []ValidatorOfRegisteredChainConfig {
			var validators []ValidatorOfRegisteredChainConfig
			for _, chainValidatorConfig := range chainConfig.Validators {
				validators = append(validators, ValidatorOfRegisteredChainConfig{
					ValidatorOperatorAddress: chainValidatorConfig.ValidatorOperatorAddress,
					WatchersIdentity:         chainValidatorConfig.Watchers,
					OptionalHealthCheckRPC:   chainValidatorConfig.OptionalHealthCheckRPC,
				})
			}
			return validators
		}(),
	}
}

func (r *registeredChainConfig) GetChainName() string {
	return r.chainName
}

func (r *registeredChainConfig) IsPriority() bool {
	return r.priority
}

func (r *registeredChainConfig) GetRPCs() []string {
	return r.rpc[:]
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
