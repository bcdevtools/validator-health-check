package telegram_call_center_svc

import (
	"fmt"
	"github.com/bcdevtools/validator-health-check/constants"
	"strings"
)

// processCommandHelp processes command /help
func (e *employee) processCommandHelp(updateCtx *telegramUpdateCtx) error {
	var sb strings.Builder

	sb.WriteString("Available commands:")
	sb.WriteString(fmt.Sprintf("\n/%s - Show your user information", constants.CommandMe))
	sb.WriteString(fmt.Sprintf("\n/%s - Show chains you subscribed", constants.CommandChains))
	sb.WriteString(fmt.Sprintf("\n/%s - Show validators you subscribed", constants.CommandValidators))
	sb.WriteString(fmt.Sprintf("\n/%s <valoper> - Show last health-check statistic of a validator", constants.CommandLast))
	if updateCtx.isRootUser {
		sb.WriteString(fmt.Sprintf("\n/%s <chain or valoper> <duration> - Pause a chain or a validator", constants.CommandPause))
	} else {
		sb.WriteString(fmt.Sprintf("\n/%s <valoper> <duration> - Pause a validator", constants.CommandPause))
	}
	sb.WriteString(fmt.Sprintf("\n/%s - Show paused chains and validators", constants.CommandStatus))
	sb.WriteString(fmt.Sprintf("\n/%s - Search for a validator by part of it address", constants.CommandSearch))
	// do not show /silent command
	sb.WriteString(fmt.Sprintf("\n/%s - Show this help message", constants.CommandHelp))

	return e.sendResponse(updateCtx, sb.String())
}
