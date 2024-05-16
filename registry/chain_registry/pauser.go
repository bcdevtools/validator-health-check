package chain_registry

import (
	"sync"
	"time"
)

var pauserMutex sync.RWMutex
var pausedChains map[string]time.Time
var pausedValidators map[string]time.Time

func PauseChainWL(chainName string, duration time.Duration) time.Time {
	pauserMutex.Lock()
	defer pauserMutex.Unlock()

	expiry := time.Now().UTC().Add(duration)
	pausedChains[chainName] = expiry
	return expiry
}

func UnpauseChainWL(chainName string) {
	pauserMutex.Lock()
	defer pauserMutex.Unlock()

	delete(pausedChains, chainName)
}

func IsChainPausedRL(chainName string) bool {
	pauserMutex.RLock()
	defer pauserMutex.RUnlock()

	expiry, paused := pausedChains[chainName]
	return paused && time.Now().UTC().Before(expiry)
}

func PauseValidatorWL(valoper string, duration time.Duration) time.Time {
	pauserMutex.Lock()
	defer pauserMutex.Unlock()

	expiry := time.Now().UTC().Add(duration)
	pausedValidators[valoper] = expiry
	return expiry
}

func UnpauseValidatorWL(valoper string) {
	pauserMutex.Lock()
	defer pauserMutex.Unlock()

	delete(pausedValidators, valoper)
}

func IsValidatorPausedRL(valoper string) bool {
	pauserMutex.RLock()
	defer pauserMutex.RUnlock()

	expiry, paused := pausedValidators[valoper]
	return paused && time.Now().UTC().Before(expiry)
}

func init() {
	pausedChains = make(map[string]time.Time)
	pausedValidators = make(map[string]time.Time)
}
