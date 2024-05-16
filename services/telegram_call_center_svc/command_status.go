package telegram_call_center_svc

import (
	chainreg "github.com/bcdevtools/validator-health-check/registry/chain_registry"
	"strings"
	"time"
)

// processCommandStatus processes command /status
func (e *employee) processCommandStatus(updateCtx *telegramUpdateCtx) error {
	var sb strings.Builder

	pausedChainsSubscribed := make(map[string]time.Time)
	pausedChainsNotSubscribed := make(map[string]time.Time)
	pausedValidatorsSubscribed := make(map[string]time.Time)
	pausedValidatorsNotSubscribed := make(map[string]time.Time)

	allChains := chainreg.GetCopyAllChainConfigsRL()
	for _, chain := range allChains {
		chainName := chain.GetChainName()
		pausedChain, expiryC := chainreg.IsChainPausedRL(chainName)
		var subscribedChain bool

		for _, val := range chain.GetValidators() {
			pausedValidator, expiryV := chainreg.IsValidatorPausedRL(val.ValidatorOperatorAddress)

			var subscribedValidator bool
			for _, watcherIdentity := range val.WatchersIdentity {
				if watcherIdentity == updateCtx.identity {
					subscribedValidator = true
					subscribedChain = true
					break
				}
			}

			if pausedValidator {
				if subscribedValidator {
					pausedValidatorsSubscribed[val.ValidatorOperatorAddress] = expiryV
				} else {
					pausedValidatorsNotSubscribed[val.ValidatorOperatorAddress] = expiryV
				}
			}
		}

		if pausedChain {
			if subscribedChain {
				pausedChainsSubscribed[chainName] = expiryC
			} else {
				pausedChainsNotSubscribed[chainName] = expiryC
			}
		}
	}

	sb.WriteString("Paused chains you subscribed:")
	if len(pausedChainsSubscribed) == 0 {
		sb.WriteString(" None")
	} else {
		for chainName, expiry := range pausedChainsSubscribed {
			sb.WriteString("\n- ")
			sb.WriteString(chainName)
			sb.WriteString(" until ")
			sb.WriteString(expiry.Format(time.DateTime))
		}
	}

	sb.WriteString("\n\nPaused validators you subscribed:")
	if len(pausedValidatorsSubscribed) == 0 {
		sb.WriteString(" None")
	} else {
		for valoper, expiry := range pausedValidatorsSubscribed {
			sb.WriteString("\n- ")
			sb.WriteString(valoper)
			sb.WriteString(" until ")
			sb.WriteString(expiry.Format(time.DateTime))
		}
	}

	if updateCtx.isRootUser {
		sb.WriteString("\n\n(Root) Paused chains you not subscribed:")
		if len(pausedChainsNotSubscribed) == 0 {
			sb.WriteString(" None")
		} else {
			for chainName, expiry := range pausedChainsNotSubscribed {
				sb.WriteString("\n- ")
				sb.WriteString(chainName)
				sb.WriteString(" until ")
				sb.WriteString(expiry.Format(time.DateTime))
			}
		}

		sb.WriteString("\n\n(Root) Paused validators you not subscribed:")
		if len(pausedValidatorsNotSubscribed) == 0 {
			sb.WriteString(" None")
		} else {
			for valoper, expiry := range pausedValidatorsNotSubscribed {
				sb.WriteString("\n- ")
				sb.WriteString(valoper)
				sb.WriteString(" until ")
				sb.WriteString(expiry.Format(time.DateTime))
			}
		}
	}

	return e.sendResponse(updateCtx, sb.String())
}
