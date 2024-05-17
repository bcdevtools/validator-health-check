package telegram_call_center_svc

import (
	"fmt"
	tpsvc "github.com/bcdevtools/validator-health-check/services/telegram_push_message_svc"
	"strings"
	"time"
)

// processCommandSilent processes command /silent
func (e *employee) processCommandSilent(updateCtx *telegramUpdateCtx) error {
	var sb strings.Builder

	args := strings.TrimSpace(updateCtx.commandArgs())
	if len(args) == 0 {
		existingPatterns := tpsvc.GetSilentPatternsByChatIdRL(updateCtx.chatId())
		if len(existingPatterns) == 0 {
			sb.WriteString("(none)")
		} else {
			sb.WriteString("Current effective patterns:")
			for pattern, expiry := range existingPatterns {
				sb.WriteString(fmt.Sprintf("\n\n- [%s] %s", expiry.Format(time.DateTime), pattern))
			}
		}
	} else {
		var duration *time.Duration
		var pattern string

		spl := strings.SplitN(args, " ", 2)
		if len(spl) != 2 {
			sb.WriteString("Invalid arguments!")
			sb.WriteString(fmt.Sprintf("\n\nUsage: /%s <duration> <pattern>", commandSilent))
			return e.sendResponse(updateCtx, sb.String())
		}

		pattern = strings.TrimSpace(spl[1])

		part := spl[1]
		switch part {
		case "0", "0s":
			duration = nil // un-silent
		default:
			dur, err := time.ParseDuration(spl[1])
			if err != nil {
				sb.WriteString("Invalid duration format!")
				return e.sendResponse(updateCtx, sb.String())
			}
			if dur < 0 || dur > 12*time.Hour {
				sb.WriteString("Duration must be positive and less than 12 hours!")
				return e.sendResponse(updateCtx, sb.String())
			}
			duration = &dur
		}

		if duration == nil {
			err := tpsvc.RemoveSilencePatternWL(updateCtx.chatId(), pattern)
			if err != nil {
				sb.WriteString("Failed to remove the silent pattern:")
				sb.WriteString(fmt.Sprintf("\n\n%s", err.Error()))
			} else {
				sb.WriteString("Removed the silent pattern")
			}
		} else {
			updated, err := tpsvc.SetSilencePatternWL(updateCtx.chatId(), pattern, *duration)
			if err != nil {
				sb.WriteString("Failed to set the silent pattern:")
				sb.WriteString(fmt.Sprintf("\n\n%s", err.Error()))
			} else if updated {
				sb.WriteString("Successfully updated expiration for the silent pattern")
			} else {
				sb.WriteString("Successfully set new silent pattern")
			}
		}
	}

	return e.sendResponse(updateCtx, sb.String())
}
