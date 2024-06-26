package health_check_worker

import (
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"sync"
	"time"
)

var cacheHcMutex sync.RWMutex
var cacheValidatorHealthCheck map[string]CacheValidatorHealthCheck

type CacheValidatorHealthCheck struct {
	Valoper                          string
	Valcons                          string
	Moniker                          string
	Rank                             int
	BondStatus                       *stakingtypes.BondStatus
	TomeStoned                       *bool
	Jailed                           *bool
	JailedUntil                      *time.Time
	MissedBlockCount                 *int64
	DowntimeSlashingWhenMissedExcess *int64
	Uptime                           *float64

	TimeOccurs time.Time
}

func putCacheValidatorHealthCheckWL(cache CacheValidatorHealthCheck) {
	cacheHcMutex.Lock()
	defer cacheHcMutex.Unlock()

	cache.TimeOccurs = time.Now().UTC()
	cacheValidatorHealthCheck[cache.Valoper] = cache
}

func GetCacheValidatorHealthCheckRL(valoper string) (CacheValidatorHealthCheck, bool) {
	cacheHcMutex.RLock()
	defer cacheHcMutex.RUnlock()

	cache, found := cacheValidatorHealthCheck[valoper]
	return cache, found
}

func init() {
	cacheValidatorHealthCheck = make(map[string]CacheValidatorHealthCheck)
}
