package telegram_call_center_svc

import (
	"fmt"
	"strings"
)

// processCommandMe processes command /help
func (e *employee) processCommandHelp(updateCtx *telegramUpdateCtx) error {

	var sb strings.Builder
	sb.WriteString("Available commands:")
	sb.WriteString(fmt.Sprintf("\n/%s - Show your user information", commandMe))
	sb.WriteString(fmt.Sprintf("\n/%s - Show chains you subscribed", commandChains))
	sb.WriteString(fmt.Sprintf("\n/%s - Show validators you subscribed", commandValidators))
	sb.WriteString(fmt.Sprintf("\n/%s - Show this help message", commandHelp))

	return e.sendResponse(updateCtx, sb.String())
}
