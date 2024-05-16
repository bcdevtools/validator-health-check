package telegram_call_center_svc

import (
	chainreg "github.com/bcdevtools/validator-health-check/registry/chain_registry"
	"strings"
)

// processCommandStatus processes command /status
func (e *employee) processCommandStatus(updateCtx *telegramUpdateCtx) error {
	var sb strings.Builder

	pausedChainsSubscribed := make(map[string]bool)
	pausedChainsNotSubscribed := make(map[string]bool)
	pausedValidatorsSubscribed := make(map[string]bool)
	pausedValidatorsNotSubscribed := make(map[string]bool)

	allChains := chainreg.GetCopyAllChainConfigsRL()
	for _, chain := range allChains {
		chainName := chain.GetChainName()
		pausedChain := chainreg.IsChainPausedRL(chainName)
		var subscribedChain bool

		for _, val := range chain.GetValidators() {
			pausedValidator := chainreg.IsValidatorPausedRL(val.ValidatorOperatorAddress)
			if !pausedValidator {
				continue
			}

			var subscribedValidator bool
			for _, watcherIdentity := range val.WatchersIdentity {
				if watcherIdentity == updateCtx.identity {
					subscribedValidator = true
					subscribedChain = true
					break
				}
			}

			if subscribedValidator {
				pausedValidatorsSubscribed[val.ValidatorOperatorAddress] = true
			} else {
				pausedValidatorsNotSubscribed[val.ValidatorOperatorAddress] = true
			}
		}

		if pausedChain {
			if subscribedChain {
				pausedChainsSubscribed[chainName] = true
			} else {
				pausedChainsNotSubscribed[chainName] = true
			}
		}
	}

	sb.WriteString("Paused chains you subscribed:")
	if len(pausedChainsSubscribed) == 0 {
		sb.WriteString(" None")
	} else {
		for chainName := range pausedChainsSubscribed {
			sb.WriteString("\n- ")
			sb.WriteString(chainName)
		}
	}

	sb.WriteString("\n\nPaused validators you subscribed:")
	if len(pausedValidatorsSubscribed) == 0 {
		sb.WriteString(" None")
	} else {
		for valoper := range pausedValidatorsSubscribed {
			sb.WriteString("\n- ")
			sb.WriteString(valoper)
		}
	}

	if updateCtx.isRootUser {
		sb.WriteString("\n\n(Root) Paused chains you not subscribed:")
		if len(pausedChainsNotSubscribed) == 0 {
			sb.WriteString(" None")
		} else {
			for chainName := range pausedChainsNotSubscribed {
				sb.WriteString("\n- ")
				sb.WriteString(chainName)
			}
		}

		sb.WriteString("\n\n(Root) Paused validators you not subscribed:")
		if len(pausedValidatorsNotSubscribed) == 0 {
			sb.WriteString(" None")
		} else {
			for valoper := range pausedValidatorsNotSubscribed {
				sb.WriteString("\n- ")
				sb.WriteString(valoper)
			}
		}
	}

	return e.sendResponse(updateCtx, sb.String())
}
