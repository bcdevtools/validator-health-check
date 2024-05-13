package config

import (
	"fmt"
	logtypes "github.com/EscanBE/go-lib/logging/types"
	"github.com/bcdevtools/validator-health-check/constants"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"os"
	"path"
	"time"
)

// AppConfig is the structure representation of configuration from `config.yaml` file
type AppConfig struct {
	General      GeneralConfig          `mapstructure:"general"`
	WorkerConfig WorkerConfig           `mapstructure:"worker"`
	Logging      logtypes.LoggingConfig `mapstructure:"logging"`
}

type GeneralConfig struct {
	HotReloadInterval   time.Duration `mapstructure:"hot-reload"`
	HealthCheckInterval time.Duration `mapstructure:"health-check"`
}

type WorkerConfig struct {
	HealthCheckCount int `mapstructure:"health-check-count"`
}

// LoadAppConfig load the configuration from `config.yaml` file within the specified application's home directory
func LoadAppConfig(homeDir string) (*AppConfig, error) {
	cfgFile := path.Join(homeDir, constants.CONFIG_FILE_NAME)

	fileStats, err := os.Stat(cfgFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("conf file %s could not be found", cfgFile)
		}

		return nil, err
	}

	if fileStats.Mode().Perm() != constants.FILE_PERMISSION && fileStats.Mode().Perm() != 0o700 {
		//goland:noinspection GoBoolExpressions
		if constants.FILE_PERMISSION == 0o700 {
			panic(fmt.Errorf("incorrect permission of %s, must be %s", constants.CONFIG_FILE_NAME, constants.FILE_PERMISSION_STR))
		} else {
			panic(fmt.Errorf("incorrect permission of %s, must be %s or 700", constants.CONFIG_FILE_NAME, constants.FILE_PERMISSION_STR))
		}
	}

	viper.SetConfigType(constants.CONFIG_TYPE)
	viper.SetConfigFile(cfgFile)

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		return nil, errors.Wrap(err, "unable to read conf file")
	}

	conf := &AppConfig{}
	err = viper.Unmarshal(conf)
	if err != nil {
		return nil, errors.Wrap(err, "unable to deserialize conf file")
	}

	return conf, nil
}

// PrintOptions prints the configuration in the `config.yaml` in a nice way, human-readable
func (c AppConfig) PrintOptions() {
	headerPrintln("- General:")
	headerPrintf("  + Hot-reload: %s\n", c.General.HotReloadInterval)
	headerPrintf("  + Health-check: %s\n", c.General.HealthCheckInterval)

	headerPrintln("- Worker's behavior:")
	headerPrintf("  + Health-check count: %d\n", c.WorkerConfig.HealthCheckCount)

	headerPrintln("- Logging:")
	if len(c.Logging.Level) < 1 {
		headerPrintf("  + Level: %s\n", logtypes.LOG_LEVEL_DEFAULT)
	} else {
		headerPrintf("  + Level: %s\n", c.Logging.Level)
	}

	if len(c.Logging.Format) < 1 {
		headerPrintf("  + Format: %s\n", logtypes.LOG_FORMAT_DEFAULT)
	} else {
		headerPrintf("  + Format: %s\n", c.Logging.Format)
	}
}

// headerPrintf prints text with prefix
func headerPrintf(format string, a ...any) {
	fmt.Printf("[HCFG]"+format, a...)
}

// headerPrintln prints text with prefix
func headerPrintln(a string) {
	fmt.Println("[HCFG]" + a)
}

// Validate performs validation on the configuration specified in the `config.yaml` within application's home directory
func (c AppConfig) Validate() error {
	if c.General.HotReloadInterval < 1*time.Minute {
		return fmt.Errorf("hot-reload interval must be at least 1 minute")
	}
	if c.General.HealthCheckInterval < 30*time.Second {
		return fmt.Errorf("health-check interval must be at least 30 seconds")
	}

	// validate Worker section
	if c.WorkerConfig.HealthCheckCount < constants.MINIMUM_WORKER_HEALTH_CHECK {
		return fmt.Errorf("workers health-check must be at least %d", constants.MINIMUM_WORKER_HEALTH_CHECK)
	}

	// validate Logging section
	errLogCfg := c.Logging.Validate()
	if errLogCfg != nil {
		return errLogCfg
	}

	return nil
}
