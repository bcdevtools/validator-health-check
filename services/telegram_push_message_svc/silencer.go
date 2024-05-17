package telegram_push_message_svc

import (
	"fmt"
	"github.com/bcdevtools/validator-health-check/constants"
	"strings"
	"sync"
	"time"
)

var mutexSilencer sync.RWMutex
var silencePatternByChatID = make(map[int64]map[string]time.Time)

func SetSilencePatternWL(chatID int64, pattern string, duration time.Duration) (update bool, err error) {
	pattern = strings.TrimSpace(pattern)
	if len(pattern) < constants.SILENT_PATTERN_MINIMUM_LENGTH {
		return false, fmt.Errorf(
			"pattern must be at least %d characters long, got: %d",
			constants.SILENT_PATTERN_MINIMUM_LENGTH,
			len(pattern),
		)
	}

	mutexSilencer.Lock()
	defer mutexSilencer.Unlock()

	nowUTC := time.Now().UTC()

	silencePatternsOfChatID, existsChatId := silencePatternByChatID[chatID]
	if existsChatId {
		// remove expired patterns
		for existingPattern, expiry := range silencePatternsOfChatID {
			if expiry.Before(nowUTC) {
				delete(silencePatternsOfChatID, existingPattern)
			}
		}

		// check if maximum patterns allowed per chat-ID
		const maximumPatternsAllowedPerChatID = 50
		if len(silencePatternsOfChatID) > maximumPatternsAllowedPerChatID {
			return false, fmt.Errorf("maximum %d patterns are allowed per chat, please wait expiry first", maximumPatternsAllowedPerChatID)
		}
	} else {
		silencePatternsOfChatID = make(map[string]time.Time)
		silencePatternByChatID[chatID] = silencePatternsOfChatID
	}

	currentExpiry, exists := silencePatternsOfChatID[pattern]
	exists = exists && currentExpiry.After(nowUTC)

	silencePatternsOfChatID[pattern] = nowUTC.Add(duration)
	return exists, nil
}

func RemoveSilencePatternWL(chatID int64, pattern string) error {
	pattern = strings.TrimSpace(pattern)
	if len(pattern) < 1 {
		return fmt.Errorf("missing pattern")
	}

	mutexSilencer.Lock()
	defer mutexSilencer.Unlock()

	silencePatternsOfChatID, existsChatId := silencePatternByChatID[chatID]
	if !existsChatId {
		return fmt.Errorf("not silent any")
	}

	if _, existsPattern := silencePatternsOfChatID[pattern]; !existsPattern {
		return fmt.Errorf("pattern does not exists")
	}

	delete(silencePatternsOfChatID, pattern)
	return nil
}

func GetSilentPatternsByChatIdRL(chatID int64) map[string]time.Time {
	copied := make(map[string]time.Time)

	mutexSilencer.RLock()
	defer mutexSilencer.RUnlock()

	silencePatternsOfChatID, existsChatId := silencePatternByChatID[chatID]
	if existsChatId {
		nowUTC := time.Now().UTC()
		for pattern, expiry := range silencePatternsOfChatID {
			if expiry.Before(nowUTC) {
				continue
			}
			copied[pattern] = expiry
		}
	}

	return silencePatternsOfChatID
}

func shouldSilentByChatIdRWL(chatID int64, message string) bool {
	nowUTC := time.Now().UTC()

	var needCleanup bool
	defer func() {
		if needCleanup {
			mutexSilencer.Lock()
			defer mutexSilencer.Unlock()

			silencePatternsOfChatID := silencePatternByChatID[chatID]
			for pattern, expiry := range silencePatternsOfChatID {
				if expiry.Before(nowUTC) {
					delete(silencePatternsOfChatID, pattern)
				}
			}
		}
	}()

	mutexSilencer.RLock()
	defer mutexSilencer.RUnlock()

	if _, exists := silencePatternByChatID[chatID]; !exists {
		return false
	}

	patterns, exists := silencePatternByChatID[chatID]
	if !exists {
		return false
	}

	for pattern, expiry := range patterns {
		if expiry.Before(nowUTC) {
			needCleanup = true
			continue
		}

		if strings.Contains(message, pattern) {
			return true
		}
	}
	return false
}
