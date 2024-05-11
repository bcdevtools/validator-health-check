package user_registry

import (
	"github.com/bcdevtools/validator-health-check/config"
	"sync"
)

var mutexUserConfig = sync.RWMutex{}
var globalIdentityToUsersConfig map[string]config.UserRecord
var globalTelegramIdToIdentity map[int64]string

func UpdateUsersConfigWL(userRecords config.UserRecords) error {
	if err := userRecords.Validate(); err != nil {
		return err
	}

	mutexUserConfig.Lock()
	defer mutexUserConfig.Unlock()

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
	mutexUserConfig.RLock()
	defer mutexUserConfig.RUnlock()

	userRecord, found = globalIdentityToUsersConfig[identity]
	return
}

func GetUserRecordByTelegramUserIdRL(telegramUserId int64) (userRecord config.UserRecord, found bool) {
	mutexUserConfig.RLock()
	defer mutexUserConfig.RUnlock()

	identity, found := globalTelegramIdToIdentity[telegramUserId]
	if !found {
		return
	}

	userRecord, found = globalIdentityToUsersConfig[identity]
	return
}

func GetTelegramUserNameByTelegramUserIdRL(telegramUserId int64) (telegramUserName string, found bool) {
	userRecord, found := GetUserRecordByTelegramUserIdRL(telegramUserId)
	if !found {
		return
	}

	telegramUserName = userRecord.TelegramConfig.Username
	found = telegramUserName != ""
	return
}
