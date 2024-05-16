package telegram_call_center_svc

import (
	chainreg "github.com/bcdevtools/validator-health-check/registry/chain_registry"
	"strings"
)

// processCommandMe processes command /chain
func (e *employee) processCommandChains(updateCtx *telegramUpdateCtx) error {
	watchChains := make(map[string]bool)
	notWatchChains := make(map[string]bool)

	allChains := chainreg.GetCopyAllChainConfigsRL()
	for _, chain := range allChains {
		chainName := chain.GetChainName()
		for _, val := range chain.GetValidators() {
			for _, watcherIdentity := range val.WatchersIdentity {
				if watcherIdentity == updateCtx.identity {
					watchChains[chainName] = true
					break
				}
			}
		}
		if updateCtx.isRootUser {
			if _, watch := watchChains[chain.GetChainName()]; !watch {
				notWatchChains[chainName] = true
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("Chains you subscribed:")
	if len(watchChains) == 0 {
		sb.WriteString(" None")
	} else {
		for chainName := range watchChains {
			sb.WriteString("\n- ")
			sb.WriteString(chainName)
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
