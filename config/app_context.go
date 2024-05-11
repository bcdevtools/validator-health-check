package config

import (
	"github.com/EscanBE/go-lib/logging"
	libutils "github.com/EscanBE/go-lib/utils"
)

// AppContext hold the working context of the application entirely.
// It contains application configuration, logger,...
type AppContext struct {
	AppConfig AppConfig
	Logger    logging.Logger
}

// NewAppContext inits `AppContext` with needed information to be stored within and return it
func NewAppContext(conf *AppConfig) *AppContext {
	logger := logging.NewDefaultLogger()

	ctx := &AppContext{
		AppConfig: *conf,
		Logger:    logger,
	}

	err := logger.ApplyConfig(conf.Logging)
	libutils.ExitIfErr(err, "failed to apply logging config")

	return ctx
}
