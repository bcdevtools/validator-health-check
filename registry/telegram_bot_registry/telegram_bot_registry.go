package telegram_bot_registry

//goland:noinspection SpellCheckingInspection
import (
	"fmt"
	"github.com/EscanBE/go-lib/logging"
	libbot "github.com/EscanBE/go-lib/telegram/bot"
	"sync"
)

var mutex sync.RWMutex
var globalTelegramBotByToken map[string]TelegramBot
var shuttingDown bool

func GetTelegramBotByTokenWL(token string, logger logging.Logger) (TelegramBot, error) {
	if token == "" {
		panic("empty telegram bot token")
	}

	bot, found, err := getTelegramBotByTokenRL(token, true)
	if err != nil {
		return nil, err
	}
	if found {
		return bot, nil
	}

	mutex.Lock()
	defer mutex.Unlock()

	// double check
	bot, found, err = getTelegramBotByTokenRL(token, false)
	if err != nil {
		return nil, err
	}
	if found {
		return bot, nil
	}

	_bot, err := libbot.NewBot(token)
	if err != nil {
		return nil, err
	}

	_bot = _bot.WithLogger(logger)
	bot = newTelegramBot(_bot)

	globalTelegramBotByToken[token] = bot
	return bot, nil
}

func GetAllTelegramBotsRL() TelegramBots {
	mutex.RLock()
	defer mutex.RUnlock()

	result := make([]TelegramBot, len(globalTelegramBotByToken))
	idx := 0
	for _, bot := range globalTelegramBotByToken {
		result[idx] = bot
		idx++
	}
	return result
}

func FlagShuttingDownWL() {
	mutex.Lock()
	defer mutex.Unlock()

	shuttingDown = true
}

func getTelegramBotByTokenRL(token string, acquireRLock bool) (bot TelegramBot, found bool, err error) {
	if acquireRLock {
		mutex.RLock()
		defer mutex.RUnlock()
	}

	if shuttingDown {
		err = fmt.Errorf("shutting down")
		return
	}

	bot, found = globalTelegramBotByToken[token]
	return
}

func init() {
	globalTelegramBotByToken = make(map[string]TelegramBot)
}
