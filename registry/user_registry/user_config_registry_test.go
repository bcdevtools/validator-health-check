package user_registry

import (
	"github.com/bcdevtools/validator-health-check/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUpdateUsersConfigWL(t *testing.T) {
	require.Error(t, UpdateUsersConfigWL(config.UserRecords{}))

	require.NoError(t, UpdateUsersConfigWL(config.UserRecords{
		{
			Identity: "1i",
			TelegramConfig: &config.UserTelegramConfig{
				Username: "1u",
				UserId:   1,
				Token:    "1t",
			},
		},
		{
			Identity: "2i",
			TelegramConfig: &config.UserTelegramConfig{
				Username: "2u",
				UserId:   2,
				Token:    "2t",
			},
		},
	}))

	userRecord, found := GetUserRecordByIdentityRL("1i")
	require.True(t, found)
	require.Equal(t, "1i", userRecord.Identity)
	require.Equal(t, "1u", userRecord.TelegramConfig.Username)
	require.Equal(t, int64(1), userRecord.TelegramConfig.UserId)
	require.Equal(t, "1t", userRecord.TelegramConfig.Token)

	userRecord, found = GetUserRecordByIdentityRL("2i")
	require.True(t, found)
	require.Equal(t, "2i", userRecord.Identity)
	require.Equal(t, "2u", userRecord.TelegramConfig.Username)
	require.Equal(t, int64(2), userRecord.TelegramConfig.UserId)
	require.Equal(t, "2t", userRecord.TelegramConfig.Token)

	userRecord, found = GetUserRecordByTelegramUserIdRL(1)
	require.True(t, found)
	require.Equal(t, "1i", userRecord.Identity)
	require.Equal(t, "1u", userRecord.TelegramConfig.Username)
	require.Equal(t, int64(1), userRecord.TelegramConfig.UserId)
	require.Equal(t, "1t", userRecord.TelegramConfig.Token)

	userRecord, found = GetUserRecordByTelegramUserIdRL(2)
	require.True(t, found)
	require.Equal(t, "2i", userRecord.Identity)
	require.Equal(t, "2u", userRecord.TelegramConfig.Username)
	require.Equal(t, int64(2), userRecord.TelegramConfig.UserId)
	require.Equal(t, "2t", userRecord.TelegramConfig.Token)

	userName, found := GetTelegramUserNameByTelegramUserIdRL(1)
	require.True(t, found)
	require.Equal(t, "1u", userName)

	userName, found = GetTelegramUserNameByTelegramUserIdRL(2)
	require.True(t, found)
	require.Equal(t, "2u", userName)
}
