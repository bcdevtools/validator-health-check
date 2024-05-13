package config

import (
	"fmt"
	"github.com/bcdevtools/validator-health-check/constants"
	"github.com/bcdevtools/validator-health-check/utils"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type ChainConfig struct {
	ChainName  string                           `mapstructure:"chain-name"`
	ChainId    string                           `mapstructure:"chain-id"`
	Disable    bool                             `mapstructure:"disable,omitempty"`
	Priority   bool                             `mapstructure:"priority,omitempty"`
	RPCs       []string                         `mapstructure:"rpc"`
	Validators map[string]*ChainValidatorConfig `mapstructure:"validators"`
}

type ChainsConfig []ChainConfig

type ChainValidatorConfig struct {
	ValidatorOperatorAddress string   `mapstructure:"-"`
	Watchers                 []string `mapstructure:"watchers"`
	OptionalHealthCheckRPC   string   `mapstructure:"health-check-rpc,omitempty"` // if provided, do health-check directly to this endpoint
}

// LoadChainsConfig load the configuration from `chain.*.yaml` file within the specified application's home directory
func LoadChainsConfig(homeDir string) (ChainsConfig, error) {
	var chainsConfig ChainsConfig
	err := filepath.Walk(homeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}
		if !strings.HasPrefix(info.Name(), constants.CHAIN_FILE_NAME_PREFIX) {
			return nil
		}
		if !strings.HasSuffix(info.Name(), "."+constants.CONFIG_TYPE) {
			return nil
		}

		fileStats, err := os.Stat(path)
		if err != nil {
			return errors.Wrapf(err, "unable to stat chain conf file %s", path)
		}

		if fileStats.Mode().Perm() != constants.FILE_PERMISSION && fileStats.Mode().Perm() != 0o700 {
			//goland:noinspection GoBoolExpressions
			if constants.FILE_PERMISSION == 0o700 {
				panic(fmt.Errorf("incorrect permission of %s, must be %s", path, constants.FILE_PERMISSION_STR))
			} else {
				panic(fmt.Errorf("incorrect permission of %s, must be %s or 700", path, constants.FILE_PERMISSION_STR))
			}
		}

		viper.SetConfigType(constants.CONFIG_TYPE)
		viper.SetConfigFile(path)

		viper.AutomaticEnv() // read in environment variables that match

		// If a config file is found, read it in.
		if err := viper.ReadInConfig(); err != nil {
			return errors.Wrap(err, "unable to read chain conf file")
		}

		conf := &ChainConfig{}
		err = viper.Unmarshal(conf)
		if err != nil {
			return errors.Wrap(err, "unable to deserialize chain conf file")
		}

		for valoper, chainValidatorConf := range conf.Validators {
			chainValidatorConf.ValidatorOperatorAddress = valoper
		}

		chainsConfig = append(chainsConfig, *conf)

		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "unable to read chains conf files")
	}

	if len(chainsConfig) == 0 {
		return nil, fmt.Errorf("no chain config found")
	}

	return chainsConfig, nil
}

func (c ChainConfig) Validate() error {
	if c.ChainName == "" {
		return fmt.Errorf("chain name is missing")
	}

	if !regexp.MustCompile(`^[\w-]+$`).MatchString(c.ChainName) {
		return fmt.Errorf("chain name must be alphanumeric, underscore, and dash only")
	}

	if c.ChainId == "" {
		return fmt.Errorf("chain id is missing")
	}

	if !regexp.MustCompile(`^[\w-]+$`).MatchString(c.ChainId) {
		return fmt.Errorf("chain id must be alphanumeric, underscore, and dash only")
	}

	if c.Disable {
		return nil
	}

	if len(c.RPCs) == 0 {
		return fmt.Errorf("RPCs are missing")
	}
	for _, rpc := range c.RPCs {
		if rpc == "" {
			return fmt.Errorf("RPCs contains empty string")
		}
	}

	if len(c.Validators) == 0 {
		return fmt.Errorf("validators are missing")
	}

	for _, validator := range c.Validators {
		if validator.ValidatorOperatorAddress == "" {
			return fmt.Errorf("validator operator address must be set")
		}
		if !utils.IsValoperAddressFormat(validator.ValidatorOperatorAddress) {
			return fmt.Errorf("validator operator address %s is invalid", validator.ValidatorOperatorAddress)
		}
		if len(validator.Watchers) == 0 {
			return fmt.Errorf("watchers for %s are missing", validator.ValidatorOperatorAddress)
		}
		for _, watcher := range validator.Watchers {
			if watcher == "" {
				return fmt.Errorf("watchers for %s contains empty string", validator.ValidatorOperatorAddress)
			}
		}
	}

	return nil
}

func (c ChainsConfig) Validate(usersConfig *UsersConfig) error {
	if len(c) == 0 {
		return fmt.Errorf("no chain config")
	}

	uniqueChainNames := make(map[string]bool)
	notDisabledCount := 0
	for _, chain := range c {
		if err := chain.Validate(); err != nil {
			return errors.Wrapf(err, "invalid chain config [%s]", chain.ChainName)
		}
		if _, found := uniqueChainNames[chain.ChainName]; found {
			return fmt.Errorf("duplicate chain name: %s", chain.ChainName)
		}
		uniqueChainNames[chain.ChainName] = true
		if !chain.Disable {
			notDisabledCount++
		}
		for _, validator := range chain.Validators {
			for _, watcher := range validator.Watchers {
				if _, found := usersConfig.Users[watcher]; !found {
					return fmt.Errorf("watcher identity %s for chain %s validator %s does not exists", watcher, chain.ChainName, validator.ValidatorOperatorAddress)
				}
			}
		}
	}

	if notDisabledCount == 0 {
		return fmt.Errorf("no enabled chain config")
	}

	return nil
}

func (c ChainsConfig) SortByPriority() ChainsConfig {
	sort.Slice(c, func(i, j int) bool {
		return c[i].Priority && !c[j].Priority
	})
	return c
}
