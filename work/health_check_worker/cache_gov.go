package health_check_worker

import "sync"

var cacheGovMutex sync.RWMutex
var cacheVotedGov map[string]uint64

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
	cacheVotedGov = make(map[string]uint64)
}
