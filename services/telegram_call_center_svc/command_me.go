package telegram_call_center_svc

import (
	"fmt"
	"strings"
)

// processCommandMe processes command /me
func (e *employee) processCommandMe(updateCtx *telegramUpdateCtx) error {
	var sb strings.Builder
	sb.WriteString("Username: ")
	sb.WriteString(updateCtx.username)
	if updateCtx.isRootUser {
		sb.WriteString("\n(Root)")
	}
	sb.WriteString("\nUser ID: ")
	sb.WriteString(fmt.Sprintf("%d", updateCtx.userId()))
	sb.WriteString("\n")
	sb.WriteString("Chat ID: ")
	sb.WriteString(fmt.Sprintf("%d", updateCtx.chatId()))

	return e.sendResponse(updateCtx, sb.String())
}
