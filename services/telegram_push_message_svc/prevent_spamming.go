package telegram_push_message_svc

import (
	"sync"
	"time"
)

type PreventSpammingCase int8

const (
	PreventSpammingCaseTomeStoned PreventSpammingCase = iota
	PreventSpammingCaseJailed
	PreventSpammingCaseLowUptime
	PreventSpammingCaseMissedBlocksOverDangerousThreshold
	PreventSpammingCaseDirectHealthCheckOptionalRPC
)

var mutexRwPreventSpamming sync.RWMutex
var globalPreventSpamming map[PreventSpammingCase]map[string]time.Time

func ShouldSendMessageWL(_case PreventSpammingCase, identities []string, ignoreIfLastSentLessThan time.Duration) (shouldSendToIdentities []string) {
	mutexRwPreventSpamming.Lock()
	defer mutexRwPreventSpamming.Unlock()

	perCaseRegistry, found := globalPreventSpamming[_case]
	if !found {
		perCaseRegistry = make(map[string]time.Time)
		globalPreventSpamming[_case] = perCaseRegistry
	}

	for _, identity := range identities {
		lastSent, found := perCaseRegistry[identity]
		if found && time.Since(lastSent) < ignoreIfLastSentLessThan {
			// ignore
		} else {
			shouldSendToIdentities = append(shouldSendToIdentities, identity)
			perCaseRegistry[identity] = time.Now().UTC()
		}
	}

	return
}

func init() {
	globalPreventSpamming = make(map[PreventSpammingCase]map[string]time.Time)
}
