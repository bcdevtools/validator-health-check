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
		sb.WriteString(fmt.Sprintf("\n- The provided address is invalid, use /%s to list or /%s by part of address", commandValidators, commandSearch))
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
	if cache.Rank > 0 {
		sb.WriteString("\nRank: ")
		sb.WriteString(fmt.Sprintf("%d", cache.Rank))
	}
	sb.WriteString("\nValoper: ")
	sb.WriteString(cache.Valoper)
	sb.WriteString("\nValcons: ")
	sb.WriteString(cache.Valcons)
	if cache.Uptime != nil {
		sb.WriteString("\nUptime: ")
		sb.WriteString(fmt.Sprintf("%.2f%%", *cache.Uptime))
	}
	if cache.BondStatus != nil {
		sb.WriteString("\nBond status: ")
		sb.WriteString(cache.BondStatus.String())
	}
	if cache.MissedBlockCount != nil {
		sb.WriteString("\nMissed blocks: ")
		if cache.DowntimeSlashingWhenMissedExcess != nil {
			sb.WriteString(fmt.Sprintf("%d/%d", *cache.MissedBlockCount, *cache.DowntimeSlashingWhenMissedExcess))
		} else {
			sb.WriteString(fmt.Sprintf("%d", *cache.MissedBlockCount))
		}
	}

	sb.WriteString("\nLast updated: ")
	sb.WriteString(cache.TimeOccurs.Format(time.DateTime))
	sb.WriteString(fmt.Sprintf(" (%.2f ago)", time.Since(cache.TimeOccurs).Seconds()))

	return e.sendResponse(updateCtx, sb.String())
}
