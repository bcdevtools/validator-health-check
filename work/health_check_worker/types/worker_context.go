package types

import (
	"github.com/bcdevtools/validator-health-check/config"
)

type HcwContext struct {
	WorkerID int
	AppCtx   config.AppContext
}
