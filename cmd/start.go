package cmd

//goland:noinspection SpellCheckingInspection
import (
	"fmt"
	libapp "github.com/EscanBE/go-lib/app"
	libcons "github.com/EscanBE/go-lib/constants"
	libutils "github.com/EscanBE/go-lib/utils"
	"github.com/bcdevtools/validator-health-check/config"
	tbotreg "github.com/bcdevtools/validator-health-check/registry/telegram_bot_registry"
	usereg "github.com/bcdevtools/validator-health-check/registry/user_registry"
	"github.com/bcdevtools/validator-health-check/utils"
	"github.com/bcdevtools/validator-health-check/work"
	workertypes "github.com/bcdevtools/validator-health-check/work/types"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"sync"
	"time"
)

var (
	waitGroup sync.WaitGroup
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start job",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Process id: %d\n", os.Getpid())

		appCfg, err := config.LoadAppConfig(homeDir)
		libutils.ExitIfErr(err, "unable to load configuration")

		// Output some options to console
		appCfg.PrintOptions()

		// Perform validation
		err = appCfg.Validate()
		libutils.ExitIfErr(err, "failed to validate configuration")

		// Init execution context
		ctx := config.NewAppContext(appCfg)
		logger := ctx.Logger

		logger.Debug("application starts")

		// Increase the waitGroup by one and decrease within trapExitSignal
		waitGroup.Add(1)

		// Register the function which should be executed upon exit.
		// After register, when you want to clean-up things before exit,
		// call libapp.ExecuteExitFunction(ctx) the same was as trapExitSignal method did
		libapp.RegisterExitFunction(func(params ...any) {
			// finalize
			defer waitGroup.Done()

			// Implements close connection, resources,... here to prevent resource leak
			safeShutdownTelegram(ctx)
		})

		// Listen for and trap any OS signal to gracefully shutdown and exit
		trapExitSignal(ctx)

		// Launch go routines
		logger.Info("launching go routine to hot-reload users config")
		go routineReloadUsersConfig(ctx)

		// Create workers

		// Start workers
		workerWorkingCtx := &workertypes.WorkerContext{
			WorkerID: time.Now().UTC().String(),
			AppCtx:   *ctx,
			Logger:   ctx.Logger,
			RoCfg:    workertypes.WorkerReadonlyConfig{},
			RwCache:  &workertypes.WorkerWritableCache{},
		}

		logger.Debug("starting worker")
		go work.NewWorker(workerWorkingCtx).Start()

		// end
		waitGroup.Wait()
	},
}

func routineReloadUsersConfig(ctx *config.AppContext) {
	logger := ctx.Logger
	defer libapp.TryRecoverAndExecuteExitFunctionIfRecovered(logger)

	hotReloadInterval := ctx.AppConfig.General.HotReloadInterval
	if hotReloadInterval < time.Minute {
		hotReloadInterval = time.Minute
	}
	firstRun := true

	for {
		if firstRun {
			firstRun = false
		} else {
			time.Sleep(hotReloadInterval)
		}
		logger.Info("hot-reload users config")

		// Reload users config
		usersConf, err := config.LoadUsersConfig(homeDir)
		if err != nil {
			logger.Error("failed to hot-reload users config, failed to load", "error", err.Error())
			continue
		}
		userRecords := usersConf.ToUserRecords()
		if err := userRecords.Validate(); err != nil {
			logger.Error("failed to hot-reload users config, validation failed", "error", err.Error())
			continue
		}

		// Update users config
		if err := usereg.UpdateUsersConfigWL(userRecords); err != nil {
			logger.Error("failed to hot-reload users config, failed to update registry", "error", err.Error())
			continue
		}

		// Init telegram bot per user
		for _, userRecord := range userRecords {
			if userRecord.TelegramConfig == nil {
				continue
			}

			// Init telegram bot
			telegramBot, err := tbotreg.GetTelegramBotByTokenWL(userRecord.TelegramConfig.Token, logger)
			if err != nil {
				logger.Error("failed to hot-reload users config, failed to init telegram bot", "identity", userRecord.Identity, "error", err.Error())
				continue
			}

			if userRecord.Root {
				telegramBot.PriorityWL()
			}

			telegramBot.AddChatIdWL(userRecord.TelegramConfig.UserId)
		}
	}
}

func safeShutdownTelegram(ctx *config.AppContext) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.Error("panic occurred during safe shutdown telegram", "error", r)
		}
	}()

	tbotreg.FlagShuttingDownWL()

	for _, bot := range tbotreg.GetAllTelegramBotsRL().SortByPriority() {
		bot.StopReceivingUpdates()

		chatIds := bot.GetAllChainIdsRL()
		if len(chatIds) > 0 {
			if bot.IsPriorityRL() {
				_bot := bot.GetInnerTelegramBot()
				for _, chatId := range chatIds {
					err := utils.Retry(func() error {
						_, err := _bot.SendMessage("validator health-check bot is shutting down", chatId)
						return err
					})
					if err != nil {
						ctx.Logger.Error("failed to send telegram message to chat", "chat-id", chatId, "priority", bot.IsPriorityRL(), "error", err.Error())
					}
				}
			} else {
				err := bot.GetInnerTelegramBot().SendMessageToMultipleChats("validator health-check bot is shutting down", chatIds, nil)
				if err != nil {
					ctx.Logger.Error("failed to send telegram message to multiple chats", "chat-count", len(chatIds), "priority", bot.IsPriorityRL(), "error", err.Error())
				}
			}
		}
	}
}

func init() {
	rootCmd.AddCommand(startCmd)
}

// trapExitSignal traps the signal which being emitted when interrupting the application. Implement connection/resource close to prevent resource leaks
func trapExitSignal(ctx *config.AppContext) {
	var sigCh = make(chan os.Signal)

	signal.Notify(sigCh, libcons.TrapExitSignals...)

	go func() {
		sig := <-sigCh
		ctx.Logger.Info(
			"caught signal; shutting down...",
			"os.signal", sig.String(),
		)

		libapp.ExecuteExitFunction()
	}()
}
