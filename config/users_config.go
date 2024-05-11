package config

import (
	"fmt"
	"github.com/bcdevtools/validator-health-check/constants"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"os"
	"path"
	"regexp"
)

type UsersConfig struct {
	Users map[string]UserRecord `mapstructure:"users"`
}

type UserRecord struct {
	Identity       string              `mapstructure:"-"`
	Root           bool                `mapstructure:"root"`
	TelegramConfig *UserTelegramConfig `mapstructure:"telegram,omitempty"`
}

type UserRecords []UserRecord

type UserTelegramConfig struct {
	Username string `mapstructure:"username"`
	UserId   int64  `mapstructure:"id"`
	Token    string `mapstructure:"token"` // token used to send message to this user
}

// LoadUsersConfig load the configuration from `users.yaml` file within the specified application's home directory
func LoadUsersConfig(homeDir string) (*UsersConfig, error) {
	usersCfgFile := path.Join(homeDir, constants.USERS_FILE_NAME)

	fileStats, err := os.Stat(usersCfgFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("users file %s could not be found", usersCfgFile)
		}

		return nil, err
	}

	if fileStats.Mode().Perm() != constants.FILE_PERMISSION && fileStats.Mode().Perm() != 0o700 {
		//goland:noinspection GoBoolExpressions
		if constants.FILE_PERMISSION == 0o700 {
			panic(fmt.Errorf("incorrect permission of %s, must be %s", constants.USERS_FILE_NAME, constants.FILE_PERMISSION_STR))
		} else {
			panic(fmt.Errorf("incorrect permission of %s, must be %s or 700", constants.USERS_FILE_NAME, constants.FILE_PERMISSION_STR))
		}
	}

	viper.SetConfigType(constants.CONFIG_TYPE)
	viper.SetConfigFile(usersCfgFile)

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		return nil, errors.Wrap(err, "unable to read users conf file")
	}

	conf := &UsersConfig{}
	err = viper.Unmarshal(conf)
	if err != nil {
		return nil, errors.Wrap(err, "unable to deserialize users conf file")
	}

	return conf, nil
}

func (c UsersConfig) ToUserRecords() UserRecords {
	records := make(UserRecords, 0, len(c.Users))
	for identity, userRecord := range c.Users {
		userRecord.Identity = identity
		records = append(records, userRecord)
	}
	return records
}

func (r UserRecord) Validate() error {
	//
	if r.Identity == "" {
		return fmt.Errorf("identity is must be set after reading from config")
	}
	if //goland:noinspection RegExpSimplifiable
	!regexp.MustCompile(`^[a-zA-Z\d_]+$`).MatchString(r.Identity) {
		return fmt.Errorf("identity must be alphanumeric and underscore only")
	}

	// telegram section
	if r.TelegramConfig == nil {
		return fmt.Errorf("telegram config must be set")
	}
	if r.TelegramConfig.Username == "" {
		return fmt.Errorf("telegram username must be set")
	}
	if //goland:noinspection RegExpSimplifiable
	!regexp.MustCompile(`^[a-zA-Z\d_]+$`).MatchString(r.TelegramConfig.Username) {
		return fmt.Errorf("telegram username must be alphanumeric and underscore only")
	}
	if r.TelegramConfig.UserId == 0 {
		return fmt.Errorf("telegram user ID must be set")
	}
	if r.TelegramConfig.Token == "" {
		return fmt.Errorf("telegram token must be set")
	}

	return nil
}

func (r UserRecords) Validate() error {
	if len(r) < 1 {
		return fmt.Errorf("no user record")
	}

	uniqueIdentities := make(map[string]bool)
	uniqueTelegramUsername := make(map[string]bool)
	uniqueTelegramUserId := make(map[int64]bool)

	for _, userRecord := range r {
		if err := userRecord.Validate(); err != nil {
			return errors.Wrapf(err, "invalid user record [%s]", userRecord.Identity)
		}
		if _, found := uniqueIdentities[userRecord.Identity]; found {
			return fmt.Errorf("duplicate user record identity: %s", userRecord.Identity)
		}
		uniqueIdentities[userRecord.Identity] = true
		if userRecord.TelegramConfig != nil {
			if _, found := uniqueTelegramUsername[userRecord.TelegramConfig.Username]; found {
				return fmt.Errorf("duplicate telegram username: %s", userRecord.TelegramConfig.Username)
			}
			uniqueTelegramUsername[userRecord.TelegramConfig.Username] = true
			if _, found := uniqueTelegramUserId[userRecord.TelegramConfig.UserId]; found {
				return fmt.Errorf("duplicate telegram user ID: %d", userRecord.TelegramConfig.UserId)
			}
			uniqueTelegramUserId[userRecord.TelegramConfig.UserId] = true
		}
	}

	return nil
}
