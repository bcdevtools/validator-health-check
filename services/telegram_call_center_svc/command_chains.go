package telegram_call_center_svc

import (
	chainreg "github.com/bcdevtools/validator-health-check/registry/chain_registry"
	"strings"
)

// processCommandChains processes command /chain
func (e *employee) processCommandChains(updateCtx *telegramUpdateCtx) error {
	watchChainsSorted := make([]string, 0)
	notWatchChainsSorted := make([]string, 0)

	allChainsSorted := chainreg.GetCopyAllChainConfigsRL().Sort()
	for _, chain := range allChainsSorted {
		chainName := chain.GetChainName()
		var watch bool
		for _, val := range chain.GetValidators() {
			for _, watcherIdentity := range val.WatchersIdentity {
				if watcherIdentity == updateCtx.identity {
					watch = true
					break
				}
			}

			if watch {
				break
			}
		}

		if watch {
			watchChainsSorted = append(watchChainsSorted, chainName)
		} else {
			notWatchChainsSorted = append(notWatchChainsSorted, chainName)
		}
	}

	var sb strings.Builder
	sb.WriteString("Chains you subscribed:")
	if len(watchChainsSorted) == 0 {
		sb.WriteString(" None")
	} else {
		for _, chainName := range watchChainsSorted {
			sb.WriteString("\n- ")
			if paused, _ := chainreg.IsChainPausedRL(chainName); paused {
				sb.WriteString("(PAUSED) ")
			}
			sb.WriteString(chainName)
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
