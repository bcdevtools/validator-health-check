package health_check_worker

import (
	"sync"
	"time"
)

var cacheGovMutex sync.RWMutex
var lastCheckGovByChain map[string]time.Time
var cacheVotedGov map[string]uint64

func putCacheLastCheckGovByChainWL(chainName string) {
	cacheGovMutex.Lock()
	defer cacheGovMutex.Unlock()

	lastCheckGovByChain[chainName] = time.Now().UTC()
}

func getLastCheckGovByChainRL(chainName string) time.Time {
	cacheGovMutex.RLock()
	defer cacheGovMutex.RUnlock()

	time, _ := lastCheckGovByChain[chainName]
	return time
}

func putCacheVotedGovWL(valoper string, proposalId uint64) {
	cacheGovMutex.Lock()
	defer cacheGovMutex.Unlock()

	if cachedValue, found := cacheVotedGov[valoper]; found && cachedValue >= proposalId {
		return
	}
	cacheVotedGov[valoper] = proposalId
}

func isVotedGovLessThan(valoper string, proposalId uint64) bool {
	cacheGovMutex.RLock()
	defer cacheGovMutex.RUnlock()

	cachedValue, found := cacheVotedGov[valoper]
	if !found {
		return true
	}
	return cachedValue < proposalId
}

func init() {
	lastCheckGovByChain = make(map[string]time.Time)
	cacheVotedGov = make(map[string]uint64)
}
