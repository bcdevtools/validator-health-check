package telegram_call_center_svc

import (
	"fmt"
	libapp "github.com/EscanBE/go-lib/app"
	"github.com/bcdevtools/validator-health-check/config"
	tbotreg "github.com/bcdevtools/validator-health-check/registry/telegram_bot_registry"
	usereg "github.com/bcdevtools/validator-health-check/registry/user_registry"
	tcctypes "github.com/bcdevtools/validator-health-check/services/telegram_call_center_svc/types"
	tpsvc "github.com/bcdevtools/validator-health-check/services/telegram_push_message_svc"
	tptypes "github.com/bcdevtools/validator-health-check/services/telegram_push_message_svc/types"
	"github.com/pkg/errors"
	"time"
)

type employee struct {
	appCtx      config.AppContext
	telegramBot tbotreg.TelegramBot
	rateLimiter tcctypes.RateLimiter
}

func newEmployee(appCtx config.AppContext, newBot tbotreg.TelegramBot, rateLimiter tcctypes.RateLimiter) *employee {
	return &employee{
		appCtx:      appCtx,
		telegramBot: newBot,
		rateLimiter: rateLimiter,
	}
}

func (e *employee) start() {
	logger := e.appCtx.Logger
	defer libapp.TryRecoverAndExecuteExitFunctionIfRecovered(logger)

	employeeID := e.telegramBot.BotID()

	logger.Info("starting new call center employee", "id", employeeID)

	for update := range e.telegramBot.GetUpdatesChannel() {
		if update.Message == nil { // ignore any non-Message updates
			continue
		}

		if !update.Message.IsCommand() { // ignore any non-command Messages
			continue
		}

		logger.Info("new telegram command", "employee", employeeID, "from", update.Message.From.ID, "msg", update.Message.Text)
		updateCtx := newTelegramUpdateCtx(update)
		err := e.processUpdate(updateCtx)
		if err != nil {
			logger.Error("error occurs during employee processing update", "error", err.Error(), "employee", employeeID, "from", update.Message.From.ID)
		}
	}
}

func (e *employee) processUpdate(updateCtx *telegramUpdateCtx) error {
	userRecord, found := usereg.GetUserRecordByTelegramUserIdRL(updateCtx.userId())
	if !found || userRecord.TelegramConfig.IsEmptyOrIncompleteConfig() {
		e.sendResponse(updateCtx, fmt.Sprintf("Hey %d, you are not allowed to use this bot", updateCtx.userId()))
		return fmt.Errorf("forbidden access, user-id: %d", updateCtx.userId())
	}
	updateCtx.identity = userRecord.Identity
	updateCtx.username = userRecord.TelegramConfig.Username
	updateCtx.isRootUser = userRecord.Root

	if !e.rateLimiter.Request(fmt.Sprintf("%d", updateCtx.userId()), 3*time.Second) {
		return e.sendResponse(updateCtx, "Rate limit exceeded, please try again later")
	}

	switch updateCtx.command() {
	case commandMe:
		return e.processCommandMe(updateCtx)
	case commandChains:
		return e.processCommandChains(updateCtx)
	case commandValidators:
		return e.processCommandValidators(updateCtx)
	case commandPause:
		return e.processCommandPause(updateCtx)
	case commandHelp:
		return e.processCommandHelp(updateCtx)
	default:
		return e.processCommandHelp(updateCtx)
	}
}

func (e *employee) sendResponse(updateCtx *telegramUpdateCtx, msg string) error {
	_, err := e.telegramBot.GetInnerTelegramBot().SendMessage(msg, updateCtx.chatId())
	return errors.Wrap(err, "failed to send response")
}

func (e *employee) enqueueToAllRootUsers(_ *telegramUpdateCtx, msg string, fatal bool) {
	for _, userRecord := range usereg.GetRootUsersIdentityRL() {
		userRecord, found := usereg.GetUserRecordByIdentityRL(userRecord)
		if !found || userRecord.TelegramConfig.IsEmptyOrIncompleteConfig() {
			continue
		}

		tpsvc.EnqueueMessageWL(tptypes.QueueMessage{
			ReceiverID: userRecord.TelegramConfig.UserId,
			Priority:   true,
			Fatal:      fatal,
			Message:    msg,
		})
	}
}
