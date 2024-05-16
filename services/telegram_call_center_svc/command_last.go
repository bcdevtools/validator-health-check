package telegram_call_center_svc

import (
	"fmt"
	hcw "github.com/bcdevtools/validator-health-check/work/health_check_worker"
	"strings"
	"time"
)

// processCommandLast processes command /last
func (e *employee) processCommandLast(updateCtx *telegramUpdateCtx) error {
	var sb strings.Builder

	args := updateCtx.commandArgs()
	if len(args) == 0 {
		sb.WriteString("Please provide a validator operator address")
		return e.sendResponse(updateCtx, sb.String())
	}

	cache, has := hcw.GetCacheValidatorHealthCheckRL(args)
	if !has {
		sb.WriteString("No health-check data found for the validator, reason maybe:")
		sb.WriteString("\n- Bot have just restarted and no health-check data yet")
		sb.WriteString("\n- The validator is not registered")
		return e.sendResponse(updateCtx, sb.String())
	}

	if cache.TomeStoned != nil && *cache.TomeStoned {
		sb.WriteString("** TomeStoned **\n")
	}
	if cache.Jailed != nil && *cache.Jailed {
		sb.WriteString("** Jailed **\n")
		if cache.JailedUntil != nil {
			sb.WriteString("(until:")
			sb.WriteString(cache.JailedUntil.Format(time.DateTime))
			sb.WriteString(")\n")
		}
	}

	sb.WriteString("Moniker: ")
	sb.WriteString(cache.Moniker)
	sb.WriteString("\nValoper: ")
	sb.WriteString(cache.Valoper)
	sb.WriteString("\nValcons: ")
	sb.WriteString(cache.Valcons)
	if cache.Uptime != nil {
		sb.WriteString("\nUptime: ")
		sb.WriteString(fmt.Sprintf("%.2f%%", *cache.Uptime))
	}
	if cache.BondStatus != nil {
		sb.WriteString("\nBondStatus: ")
		sb.WriteString(cache.BondStatus.String())
	}
	if cache.MissedBlockCount != nil && cache.DowntimeSlashingWhenMissedExcess != nil {
		sb.WriteString("\nMissedBlockCount: ")
		sb.WriteString(fmt.Sprintf("%d/%d", *cache.MissedBlockCount, *cache.DowntimeSlashingWhenMissedExcess))
	} else if cache.MissedBlockCount != nil {
		sb.WriteString("\nMissedBlockCount: ")
		sb.WriteString(fmt.Sprintf("%d", *cache.MissedBlockCount))
	} else if cache.DowntimeSlashingWhenMissedExcess != nil {
		sb.WriteString("\nDowntimeSlashingWhenMissedExcess: ")
		sb.WriteString(fmt.Sprintf("%d", *cache.DowntimeSlashingWhenMissedExcess))
	}

	sb.WriteString("\nLast updated: ")
	sb.WriteString(cache.TimeOccurs.Format(time.DateTime))
	sb.WriteString(fmt.Sprintf(" (%s ago)", time.Since(cache.TimeOccurs).String()))

	return e.sendResponse(updateCtx, sb.String())
}
