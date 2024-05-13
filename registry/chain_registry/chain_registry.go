package chain_registry

import (
	"github.com/bcdevtools/validator-health-check/config"
	"sync"
	"time"
)

var mutex sync.RWMutex
var globalChainNameToChainConfig map[string]RegisteredChainConfig
var globalWatcherIdentityToChainNameToValoper map[string]map[string][]string

func UpdateChainsConfigWL(chainConfigs config.ChainsConfig, usersConfig *config.UsersConfig) error {
	if err := chainConfigs.Validate(usersConfig); err != nil {
		return err
	}

	mutex.Lock()
	defer mutex.Unlock()

	// prune old data
	globalChainNameToChainConfig = make(map[string]RegisteredChainConfig)
	globalWatcherIdentityToChainNameToValoper = make(map[string]map[string][]string)

	// put new data
	for _, chainConfig := range chainConfigs {
		if chainConfig.Disable {
			continue
		}
		globalChainNameToChainConfig[chainConfig.ChainName] = newRegisteredChainConfig(chainConfig)
		for _, validator := range chainConfig.Validators {
			for _, watcher := range validator.Watchers {
				if _, found := globalWatcherIdentityToChainNameToValoper[watcher]; !found {
					globalWatcherIdentityToChainNameToValoper[watcher] = make(map[string][]string)
				}
				globalWatcherIdentityToChainNameToValoper[watcher][chainConfig.ChainName] = append(globalWatcherIdentityToChainNameToValoper[watcher][chainConfig.ChainName], validator.ValidatorOperatorAddress)
			}
		}
	}

	return nil
}

// GetFirstChainConfigForHealthCheckRL returns the first chain config that needs to be health-checked.
//
// NOTICE: this update the last health-check of the chain to recent time, so that the next health-check worker will not pick it up.
func GetFirstChainConfigForHealthCheckRL(minDurationSinceLastHealthCheck time.Duration) RegisteredChainConfig {
	mutex.RLock()
	defer mutex.RUnlock()

	for _, chainConfig := range globalChainNameToChainConfig {
		lastHealthCheckUTC := chainConfig.GetLastHealthCheckUtcRL()
		if time.Since(lastHealthCheckUTC) >= minDurationSinceLastHealthCheck {
			chainConfig.SetLastHealthCheckUtcWL() // prevent double health-check
			return chainConfig
		}
	}

	return nil
}
