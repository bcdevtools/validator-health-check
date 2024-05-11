package telegram_bot_registry

//goland:noinspection SpellCheckingInspection
import (
	libbot "github.com/EscanBE/go-lib/telegram/bot"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"sort"
	"sync"
)

type TelegramBots []TelegramBot

type TelegramBot interface {
	GetInnerTelegramBot() *libbot.TelegramBot
	GetUpdatesChannel() tgbotapi.UpdatesChannel
	StopReceivingUpdates()
	AddChatIdWL(chatId int64)
	GetAllChainIdsRL() []int64
	PriorityWL()
	IsPriorityRL() bool
}

var _ TelegramBot = &telegramBot{}

type telegramBot struct {
	sync.RWMutex
	bot      *libbot.TelegramBot
	chatIds  map[int64]bool
	priority bool
}

func newTelegramBot(bot *libbot.TelegramBot) TelegramBot {
	return &telegramBot{
		bot:     bot,
		chatIds: make(map[int64]bool),
	}
}

func (t *telegramBot) GetInnerTelegramBot() *libbot.TelegramBot {
	return t.bot
}

func (t *telegramBot) GetUpdatesChannel() tgbotapi.UpdatesChannel {
	return t.bot.GetUpdatesChannel()
}

func (t *telegramBot) StopReceivingUpdates() {
	t.bot.StopReceivingUpdates()
}

func (t *telegramBot) AddChatIdWL(chatId int64) {
	t.Lock()
	defer t.Unlock()

	t.chatIds[chatId] = true
}

func (t *telegramBot) GetAllChainIdsRL() []int64 {
	t.RLock()
	defer t.RUnlock()

	chatIds := make([]int64, len(t.chatIds))
	idx := 0
	for chatId := range t.chatIds {
		chatIds[idx] = chatId
		idx++
	}
	return chatIds
}

func (t *telegramBot) PriorityWL() {
	t.Lock()
	defer t.Unlock()

	t.priority = true
}

func (t *telegramBot) IsPriorityRL() bool {
	t.RLock()
	defer t.RUnlock()

	return t.priority
}

func (bs TelegramBots) SortByPriority() TelegramBots {
	sort.Slice(bs, func(i, j int) bool {
		return bs[i].IsPriorityRL() && !bs[j].IsPriorityRL()
	})
	return bs
}
