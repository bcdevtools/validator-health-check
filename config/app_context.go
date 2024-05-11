package config

import (
	"github.com/EscanBE/go-lib/logging"
	"github.com/EscanBE/go-lib/telegram/bot"
	libutils "github.com/EscanBE/go-lib/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// AppContext hold the working context of the application entirely.
// It contains application configuration, logger,...
type AppContext struct {
	Config AppConfig
	Bot    *bot.TelegramBot
	Logger logging.Logger
}

// NewAppContext inits `AppContext` with needed information to be stored within and return it
func NewAppContext(conf *AppConfig, bot *bot.TelegramBot) *AppContext {
	logger := logging.NewDefaultLogger()

	ctx := &AppContext{
		Config: *conf,
		Bot:    bot,
		Logger: logger,
	}

	err := logger.ApplyConfig(conf.Logging)
	libutils.ExitIfErr(err, "failed to apply logging config")

	return ctx
}

func (aec AppContext) SendTelegramLogMessage(msg string) (*tgbotapi.Message, error) {
	if aec.Bot == nil {
		return nil, nil
	}
	m, err := aec.SendTelegramMessage(tgbotapi.NewMessage(aec.Config.TelegramConfig.LogChannelID, msg))
	if err != nil {
		aec.Logger.Error("Failed to send telegram log message", "type", "log", "error", err.Error())
	}
	return m, err
}

func (aec AppContext) SendTelegramErrorMessage(msg string) (*tgbotapi.Message, error) {
	if aec.Bot == nil {
		return nil, nil
	}
	m, err := aec.SendTelegramMessage(tgbotapi.NewMessage(aec.Config.TelegramConfig.ErrChannelID, msg))
	if err != nil {
		aec.Logger.Error("Failed to send telegram error message", "type", "error", "error", err.Error())
		return nil, err
	}
	return m, nil
}

func (aec AppContext) SendTelegramError(err error) (*tgbotapi.Message, error) {
	return aec.SendTelegramErrorMessage(err.Error())
}

func (aec AppContext) SendTelegramMessage(c tgbotapi.Chattable) (*tgbotapi.Message, error) {
	if aec.Bot == nil {
		return nil, nil
	}
	m, err := aec.Bot.Send(c)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
