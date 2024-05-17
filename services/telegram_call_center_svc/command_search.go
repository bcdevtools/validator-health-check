package telegram_call_center_svc

import (
	"fmt"
	chainreg "github.com/bcdevtools/validator-health-check/registry/chain_registry"
	"strings"
)

// processCommandSearch processes command /search
func (e *employee) processCommandSearch(updateCtx *telegramUpdateCtx) error {
	var sb strings.Builder

	args := strings.TrimSpace(strings.ToLower(updateCtx.commandArgs()))
	if len(args) == 0 {
		sb.WriteString("Please provide a part of the validator operator address you want to search for!")
		sb.WriteString(fmt.Sprintf("\n? Use /%s to list all validators", commandValidators))
		return e.sendResponse(updateCtx, sb.String())
	}

	if len(args) < 3 {
		sb.WriteString("Search query must be at least 3 characters long!")
		return e.sendResponse(updateCtx, sb.String())
	}

	validators := make(map[string]bool)
	for _, chain := range chainreg.GetCopyAllChainConfigsRL() {
		for _, val := range chain.GetValidators() {
			if !strings.Contains(val.ValidatorOperatorAddress, args) {
				continue
			}

			for _, watcherIdentity := range val.WatchersIdentity {
				if watcherIdentity == updateCtx.identity {
					validators[val.ValidatorOperatorAddress] = true
					break
				}
			}

			if updateCtx.isRootUser {
				if _, watch := validators[val.ValidatorOperatorAddress]; !watch {
					validators[val.ValidatorOperatorAddress] = false
				}
			}
		}
	}

	if len(validators) == 0 {
		sb.WriteString("Not match any, try longer query!")
	} else {
		var cnt int
		for val, watch := range validators {
			cnt++

			if cnt > 1 {
				sb.WriteString("\n\n")
			}

			sb.WriteString(val)

			if !watch {
				sb.WriteString(" (not subscribed)")
			}
			break
		}
	}

	return e.sendResponse(updateCtx, sb.String())
}
