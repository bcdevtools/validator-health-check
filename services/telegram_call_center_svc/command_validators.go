package telegram_call_center_svc

import (
	"fmt"
	chainreg "github.com/bcdevtools/validator-health-check/registry/chain_registry"
	"strings"
)

// processCommandMe processes command /validators
func (e *employee) processCommandValidators(updateCtx *telegramUpdateCtx) error {
	watchChains := make(map[string]map[string]bool)
	notWatchChains := make(map[string]bool)

	allChains := chainreg.GetCopyAllChainConfigsRL()
	for _, chain := range allChains {
		chainName := chain.GetChainName()
		for _, val := range chain.GetValidators() {
			for _, watcherIdentity := range val.WatchersIdentity {
				if watcherIdentity == updateCtx.identity {
					watchValidators, exists := watchChains[chainName]
					if !exists {
						watchValidators = make(map[string]bool)
						watchChains[chainName] = watchValidators
					}
					watchValidators[val.ValidatorOperatorAddress] = true
					break
				}
			}
		}
		if updateCtx.isRootUser {
			if _, watch := watchChains[chainName]; !watch {
				notWatchChains[chainName] = true
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("Validators you subscribed:")
	if len(watchChains) == 0 {
		sb.WriteString(" None")
	} else {
		for chainName, validators := range watchChains {
			for validator := range validators {
				sb.WriteString("\n- ")
				sb.WriteString(validator)
				sb.WriteString(fmt.Sprintf(" (%s)", chainName))
			}
		}
	}

	if updateCtx.isRootUser && len(notWatchChains) > 0 {
		sb.WriteString("\n\n(Root) Chains you not subscribed:")
		if len(notWatchChains) == 0 {
			sb.WriteString(" None")
		} else {
			for chainName := range notWatchChains {
				sb.WriteString("\n- ")
				sb.WriteString(chainName)
			}
		}
	}

	return e.sendResponse(updateCtx, sb.String())
}
