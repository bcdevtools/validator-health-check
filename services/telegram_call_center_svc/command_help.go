package telegram_call_center_svc

import (
	"fmt"
	"strings"
)

// processCommandHelp processes command /help
func (e *employee) processCommandHelp(updateCtx *telegramUpdateCtx) error {

	var sb strings.Builder
	sb.WriteString("Available commands:")
	sb.WriteString(fmt.Sprintf("\n/%s - Show your user information", commandMe))
	sb.WriteString(fmt.Sprintf("\n/%s - Show chains you subscribed", commandChains))
	sb.WriteString(fmt.Sprintf("\n/%s - Show validators you subscribed", commandValidators))
	if updateCtx.isRootUser {
		sb.WriteString(fmt.Sprintf("\n/%s <chain or valoper> <duration> - Pause a chain or a validator", commandPause))
	} else {
		sb.WriteString(fmt.Sprintf("\n/%s <valoper> <duration> - Pause a validator", commandPause))
	}
	sb.WriteString(fmt.Sprintf("\n/%s - Show paused chains and validators", commandStatus))
	sb.WriteString(fmt.Sprintf("\n/%s - Show this help message", commandHelp))

	return e.sendResponse(updateCtx, sb.String())
}
