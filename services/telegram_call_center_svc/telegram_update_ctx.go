package telegram_call_center_svc

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type telegramUpdateCtx struct {
	update     tgbotapi.Update
	identity   string
	username   string
	isRootUser bool
}

func newTelegramUpdateCtx(update tgbotapi.Update) *telegramUpdateCtx {
	return &telegramUpdateCtx{
		update: update,
	}
}

func (c *telegramUpdateCtx) userId() int64 {
	return c.update.Message.From.ID
}

func (c *telegramUpdateCtx) chatId() int64 {
	return c.update.Message.Chat.ID
}

func (c *telegramUpdateCtx) command() string {
	return c.update.Message.Command()
}

func (c *telegramUpdateCtx) commandArgs() string {
	return c.update.Message.CommandArguments()
}
