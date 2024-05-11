package types

import (
	"github.com/EscanBE/go-lib/logging"
	"github.com/EscanBE/go-lib/telegram/bot"
	"github.com/bcdevtools/validator-health-check/config"
)

// WorkerContext hold the working context for each of the worker.
// In here, we can save identity, config, caches,...
type WorkerContext struct {
	WorkerID string
	AppCtx   config.AppContext
	Logger   logging.Logger
	RoCfg    WorkerReadonlyConfig
	RwCache  *WorkerWritableCache
}

// WorkerReadonlyConfig contains readonly configuration options
type WorkerReadonlyConfig struct {
}

// WorkerWritableCache contains caches, resources shared across workers, or local init & use, depends on implementation
type WorkerWritableCache struct {
}

// GetTelegramBot returns bot.TelegramBot instance
func (wc WorkerContext) GetTelegramBot() *bot.TelegramBot {
	return wc.AppCtx.Bot
}
