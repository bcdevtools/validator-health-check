package telegram_call_center_svc

//goland:noinspection SpellCheckingInspection
import (
	libapp "github.com/EscanBE/go-lib/app"
	"github.com/bcdevtools/validator-health-check/config"
	tbotreg "github.com/bcdevtools/validator-health-check/registry/telegram_bot_registry"
	tcctypes "github.com/bcdevtools/validator-health-check/services/telegram_call_center_svc/types"
	"sync"
)

type telegramCallCenter struct {
	sync.Mutex
	appCtx        config.AppContext
	rateLimiter   tcctypes.RateLimiter
	uniqueTracker map[string]bool
}

func StartTelegramCallCenterService(appCtx config.AppContext) {
	callCenter := &telegramCallCenter{
		appCtx:        appCtx,
		rateLimiter:   tcctypes.NewRateLimiter(),
		uniqueTracker: make(map[string]bool),
	}
	go callCenter.start(appCtx)
}

func (cc *telegramCallCenter) start(appCtx config.AppContext) {
	logger := appCtx.Logger
	defer libapp.TryRecoverAndExecuteExitFunctionIfRecovered(logger)

	for newBot := range tbotreg.ChannelNewBot {
		botId := newBot.BotID()
		launch := func(botId string) bool {
			cc.Lock()
			defer cc.Unlock()

			if _, found := cc.uniqueTracker[botId]; found {
				logger.Error("telegram bot already launched", "botId", botId)
				return false
			}

			cc.uniqueTracker[botId] = true
			return true
		}(botId)

		if !launch {
			continue
		}

		employee := newEmployee(appCtx, newBot, cc.rateLimiter)
		go employee.start()
	}
}
