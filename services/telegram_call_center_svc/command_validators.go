package telegram_call_center_svc

import (
	"fmt"
	chainreg "github.com/bcdevtools/validator-health-check/registry/chain_registry"
	"strings"
)

// processCommandValidators processes command /validators
func (e *employee) processCommandValidators(updateCtx *telegramUpdateCtx) error {
	watchValidatorOnChains := make(map[string]map[string]bool)
	watchChainsSorted := make([]string, 0)
	notWatchChainsSorted := make([]string, 0)

	allChainsSorted := chainreg.GetCopyAllChainConfigsRL().Sort()
	for _, chain := range allChainsSorted {
		chainName := chain.GetChainName()
		for _, val := range chain.GetValidators() {
			for _, watcherIdentity := range val.WatchersIdentity {
				if watcherIdentity == updateCtx.identity {
					watchValidators, exists := watchValidatorOnChains[chainName]
					if !exists {
						watchValidators = make(map[string]bool)
						watchValidatorOnChains[chainName] = watchValidators
						watchChainsSorted = append(watchChainsSorted, chainName)
					}
					watchValidators[val.ValidatorOperatorAddress] = true
					break
				}
			}
		}
		if updateCtx.isRootUser {
			if _, watch := watchValidatorOnChains[chainName]; !watch {
				notWatchChainsSorted = append(notWatchChainsSorted, chainName)
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("Validators you subscribed:")
	if len(watchValidatorOnChains) == 0 {
		sb.WriteString(" None")
	} else {
		for _, chainName := range watchChainsSorted {
			for validator := range watchValidatorOnChains[chainName] {
				sb.WriteString("\n\n- ")
				if paused, _ := chainreg.IsValidatorPausedRL(validator); paused {
					sb.WriteString("(PAUSED) ")
				}
				sb.WriteString(validator)
				if paused, _ := chainreg.IsChainPausedRL(chainName); paused {
					sb.WriteString(fmt.Sprintf(" (%s - PAUSED)", chainName))
				} else {
					sb.WriteString(fmt.Sprintf(" (%s)", chainName))
				}
			}
		}
	}

	if updateCtx.isRootUser && len(notWatchChainsSorted) > 0 {
		sb.WriteString("\n\n(Root) Chains you not subscribed:")
		if len(notWatchChainsSorted) == 0 {
			sb.WriteString(" None")
		} else {
			for _, chainName := range notWatchChainsSorted {
				sb.WriteString("\n- ")
				if paused, _ := chainreg.IsChainPausedRL(chainName); paused {
					sb.WriteString("(PAUSED) ")
				}
				sb.WriteString(chainName)
			}
		}
	}

	return e.sendResponse(updateCtx, sb.String())
}
