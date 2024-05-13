package types

import (
	"github.com/bcdevtools/validator-health-check/config"
)

type HcwContext struct {
	WorkerID int
	AppCtx   config.AppContext

	// TODO remove if not used
	RoCfg   WorkerReadonlyConfig
	RwCache *WorkerWritableCache
}

// WorkerReadonlyConfig contains readonly configuration options
type WorkerReadonlyConfig struct {
}

// WorkerWritableCache contains caches, resources shared across workers, or local init & use, depends on implementation
type WorkerWritableCache struct {
}
