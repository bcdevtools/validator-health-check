package telegram_call_center_svc

import (
	"fmt"
	"github.com/bcdevtools/validator-health-check/constants"
	chainreg "github.com/bcdevtools/validator-health-check/registry/chain_registry"
	"strings"
	"time"
)

// processCommandPause processes command /pause
func (e *employee) processCommandPause(updateCtx *telegramUpdateCtx) error {
	var sb strings.Builder

	args := updateCtx.commandArgs()
	if len(args) == 0 {
		if updateCtx.isRootUser {
			sb.WriteString("Please provide a chain or a validator to pause!")
			sb.WriteString(fmt.Sprintf("\nSee the list at /%s or /%s", constants.CommandChains, constants.CommandValidators))
		} else {
			sb.WriteString("Please provide a validator to pause!")
			sb.WriteString(fmt.Sprintf("\nSee the list at /%s", constants.CommandValidators))
		}
		return e.sendResponse(updateCtx, sb.String())
	}

	spl := strings.Split(args, " ")
	if len(spl) > 2 {
		sb.WriteString("Invalid arguments!")
		return e.sendResponse(updateCtx, sb.String())
	}

	target := spl[0]
	var duration *time.Duration
	var ultimatePause bool
	if len(spl) > 1 {
		part := spl[1]
		switch part {
		case "0", "0s":
			duration = nil // unpause
		default:
			dur, err := time.ParseDuration(part)
			if err != nil {
				sb.WriteString("Invalid duration format!")
				return e.sendResponse(updateCtx, sb.String())
			}
			if dur < 0 {
				sb.WriteString("Duration must be positive!")
				return e.sendResponse(updateCtx, sb.String())
			}
			if dur > 7*time.Hour {
				sb.WriteString("Duration must be less than 7 hours!")
				return e.sendResponse(updateCtx, sb.String())
			}
			duration = &dur
		}
	} else {
		dur := 30 * 365 * 24 * time.Hour
		duration = &dur
		ultimatePause = true
	}

	if updateCtx.isRootUser && !strings.Contains(target, "valoper") {
		found, err := e.processCommandPauseTryChainForRoot(updateCtx, target, duration, ultimatePause)
		if found || err != nil {
			return err
		}
	}

	found, err := e.processCommandPauseTryValidator(updateCtx, target, duration, ultimatePause)
	if found || err != nil {
		return err
	}

	if updateCtx.isRootUser {
		sb.WriteString("No chain or validator found with the provided identifier!")
		sb.WriteString(fmt.Sprintf("\nSee the list at /%s or /%s or use /%s", constants.CommandChains, constants.CommandValidators, constants.CommandSearch))
	} else {
		sb.WriteString("No validator found with the provided identifier!")
		sb.WriteString(fmt.Sprintf("\nSee the list at /%s or use /%s", constants.CommandValidators, constants.CommandSearch))
	}

	return e.sendResponse(updateCtx, sb.String())
}

func (e *employee) processCommandPauseTryChainForRoot(updateCtx *telegramUpdateCtx, chain string, duration *time.Duration, ultimatePause bool) (found bool, err error) {
	if !updateCtx.isRootUser {
		panic("this method should only be called by root user")
	}

	if !chainreg.HasChainRL(chain) {
		return false, nil
	}

	if duration == nil {
		chainreg.UnpauseChainWL(chain)
		e.enqueueToAllRootUsers(
			updateCtx,
			fmt.Sprintf("%s (%s) has unpaused chain [%s]", updateCtx.identity, updateCtx.username, chain),
			false,
		)
		return true, e.sendResponse(updateCtx, fmt.Sprintf("Chain [%s] has unpaused", chain))
	}

	expiry := chainreg.PauseChainWL(chain, *duration)
	e.enqueueToAllRootUsers(
		updateCtx,
		fmt.Sprintf("%s (%s) has PAUSED chain [%s] %s", updateCtx.identity, updateCtx.username, chain, func() string {
			if ultimatePause {
				return "without release date"
			} else {
				return fmt.Sprintf("for %s, until %s", duration.String(), expiry.Format(time.DateTime))
			}
		}()),
		true,
	)
	if ultimatePause {
		return true, e.sendResponse(updateCtx, fmt.Sprintf("Chain [%s] has been PAUSED without release date", chain))
	} else {
		return true, e.sendResponse(updateCtx, fmt.Sprintf("Chain [%s] has been PAUSED for %s, until %s", chain, duration.String(), expiry.Format(time.DateTime)))
	}
}

func (e *employee) processCommandPauseTryValidator(updateCtx *telegramUpdateCtx, valoper string, duration *time.Duration, ultimatePause bool) (found bool, err error) {
	var chainName string
	var granted bool

	granted = updateCtx.isRootUser

	allChains := chainreg.GetCopyAllChainConfigsRL()
	for _, chain := range allChains {
		var foundVal bool
		for _, val := range chain.GetValidators() {
			if val.ValidatorOperatorAddress != valoper {
				continue
			}

			foundVal = true
			chainName = chain.GetChainName()

			if granted {
				break
			}

			for _, watcherIdentity := range val.WatchersIdentity {
				if watcherIdentity == updateCtx.identity {
					granted = true
					break
				}
			}
		}
		if foundVal {
			break
		}
	}

	if chainName == "" || !granted {
		return false, nil
	}

	if duration == nil {
		chainreg.UnpauseValidatorWL(valoper)
		e.enqueueToAllRootUsers(
			updateCtx,
			fmt.Sprintf("%s (%s) has unpaused validator [%s] on [%s]", updateCtx.identity, updateCtx.username, valoper, chainName),
			false,
		)
		return true, e.sendResponse(updateCtx, fmt.Sprintf("Validator [%s] on [%s] has unpaused", valoper, chainName))
	}

	expiry := chainreg.PauseValidatorWL(valoper, *duration)
	e.enqueueToAllRootUsers(
		updateCtx,
		fmt.Sprintf("%s (%s) has PAUSED validator [%s] on %s %s", updateCtx.identity, updateCtx.username, valoper, chainName, func() string {
			if ultimatePause {
				return "without release date"
			} else {
				return fmt.Sprintf("for %s, until %s", duration.String(), expiry.Format(time.DateTime))
			}
		}()),
		true,
	)

	if ultimatePause {
		return true, e.sendResponse(updateCtx, fmt.Sprintf("Validator [%s] on %s has been PAUSED without release date", valoper, chainName))
	} else {
		return true, e.sendResponse(updateCtx, fmt.Sprintf("Validator [%s] on %s has been PAUSED for %s, until %s", valoper, chainName, duration.String(), expiry.Format(time.DateTime)))
	}
}
