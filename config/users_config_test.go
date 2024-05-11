package config

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUserRecord_Validate(t *testing.T) {
	tests := []struct {
		name            string
		userRecord      UserRecord
		wantErr         bool
		wantErrContains string
	}{
		{
			name: "pass",
			userRecord: UserRecord{
				Identity: "1",
				TelegramConfig: &UserTelegramConfig{
					Username: "1",
					UserId:   1,
					Token:    "1",
				},
			},
			wantErr: false,
		},
		{
			name: "identity must be set",
			userRecord: UserRecord{
				Identity: "",
				TelegramConfig: &UserTelegramConfig{
					Username: "1",
					UserId:   1,
					Token:    "1",
				},
			},
			wantErr:         true,
			wantErrContains: "identity is must be set",
		},
		{
			name: "identity contains special character",
			userRecord: UserRecord{
				Identity: "$",
				TelegramConfig: &UserTelegramConfig{
					Username: "1",
					UserId:   1,
					Token:    "1",
				},
			},
			wantErr:         true,
			wantErrContains: "identity must be alphanumeric",
		},
		{
			name: "telegram must be set",
			userRecord: UserRecord{
				Identity:       "1",
				TelegramConfig: nil,
			},
			wantErr:         true,
			wantErrContains: "telegram config must be set",
		},
		{
			name: "telegram username must be set",
			userRecord: UserRecord{
				Identity: "1",
				TelegramConfig: &UserTelegramConfig{
					Username: "",
					UserId:   1,
					Token:    "1",
				},
			},
			wantErr:         true,
			wantErrContains: "telegram username must be set",
		},
		{
			name: "telegram username validation",
			userRecord: UserRecord{
				Identity: "1",
				TelegramConfig: &UserTelegramConfig{
					Username: "$",
					UserId:   1,
					Token:    "1",
				},
			},
			wantErr:         true,
			wantErrContains: "telegram username must be alphanumeric",
		},
		{
			name: "telegram user id must be set",
			userRecord: UserRecord{
				Identity: "1",
				TelegramConfig: &UserTelegramConfig{
					Username: "1",
					UserId:   0,
					Token:    "1",
				},
			},
			wantErr:         true,
			wantErrContains: "telegram user ID must be set",
		},
		{
			name: "telegram token must be set",
			userRecord: UserRecord{
				Identity: "1",
				TelegramConfig: &UserTelegramConfig{
					Username: "1",
					UserId:   1,
					Token:    "",
				},
			},
			wantErr:         true,
			wantErrContains: "telegram token must be set",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := UserRecord{
				Identity:       tt.userRecord.Identity,
				Root:           tt.userRecord.Root,
				TelegramConfig: tt.userRecord.TelegramConfig,
			}
			if tt.wantErr {
				err := r.Validate()
				require.Error(t, err)
				require.NotEmpty(t, tt.wantErrContains, "need setup")
				require.Contains(t, err.Error(), tt.wantErrContains)
			} else {
				require.NoError(t, r.Validate())
			}
		})
	}
}
