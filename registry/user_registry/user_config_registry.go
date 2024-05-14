package user_registry

import (
	"github.com/bcdevtools/validator-health-check/config"
	"sync"
)

var mutex sync.RWMutex
var globalIdentityToUsersConfig map[string]config.UserRecord
var globalTelegramIdToIdentity map[int64]string

func UpdateUsersConfigWL(userRecords config.UserRecords) error {
	if err := userRecords.Validate(); err != nil {
		return err
	}

	mutex.Lock()
	defer mutex.Unlock()

	// prune old data
	globalIdentityToUsersConfig = make(map[string]config.UserRecord)
	globalTelegramIdToIdentity = make(map[int64]string)

	// put new data
	for _, userRecord := range userRecords {
		globalIdentityToUsersConfig[userRecord.Identity] = userRecord
		if userRecord.TelegramConfig != nil {
			globalTelegramIdToIdentity[userRecord.TelegramConfig.UserId] = userRecord.Identity
		}
	}

	return nil
}

func GetUserRecordByIdentityRL(identity string) (userRecord config.UserRecord, found bool) {
	mutex.RLock()
	defer mutex.RUnlock()

	userRecord, found = globalIdentityToUsersConfig[identity]
	return
}

func GetUserRecordByTelegramUserIdRL(telegramUserId int64) (userRecord config.UserRecord, found bool) {
	mutex.RLock()
	defer mutex.RUnlock()

	identity, found := globalTelegramIdToIdentity[telegramUserId]
	if !found {
		return
	}

	userRecord, found = globalIdentityToUsersConfig[identity]
	return
}

func GetRootUsersIdentityRL() []string {
	mutex.RLock()
	defer mutex.RUnlock()

	var rootUsersIdentity []string
	for identity, userRecord := range globalIdentityToUsersConfig {
		if userRecord.Root {
			rootUsersIdentity = append(rootUsersIdentity, identity)
		}
	}
	return rootUsersIdentity
}
