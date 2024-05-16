package cmd

//goland:noinspection SpellCheckingInspection
import (
	"fmt"
	libapp "github.com/EscanBE/go-lib/app"
	libcons "github.com/EscanBE/go-lib/constants"
	libutils "github.com/EscanBE/go-lib/utils"
	"github.com/bcdevtools/validator-health-check/config"
	chainreg "github.com/bcdevtools/validator-health-check/registry/chain_registry"
	tbotreg "github.com/bcdevtools/validator-health-check/registry/telegram_bot_registry"
	usereg "github.com/bcdevtools/validator-health-check/registry/user_registry"
	tpsvc "github.com/bcdevtools/validator-health-check/services/telegram_push_message_svc"
	"github.com/bcdevtools/validator-health-check/services/telegram_push_message_svc/types"
	"github.com/bcdevtools/validator-health-check/utils"
	"github.com/bcdevtools/validator-health-check/work/health_check_worker"
	workertypes "github.com/bcdevtools/validator-health-check/work/health_check_worker/types"
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
		logger.Debug("launching go routine to inform startup")
		go routineInformStartup(ctx)

		logger.Info("launching go routine to hot-reload config")
		go routineHotReload(ctx)

		// Start telegram pusher service
		logger.Debug("starting telegram pusher service")
		tpsvc.StartTelegramPusherService(types.TpContext{
			AppCtx: *ctx,
		})

		// Start health-check workers
		for id := 1; id <= appCfg.WorkerConfig.HealthCheckCount; id++ {
			workerWorkingCtx := &workertypes.HcwContext{
				WorkerID: id,
				AppCtx:   *ctx,
			}

			logger.Debug("starting health-check worker", "wid", workerWorkingCtx.WorkerID)
			go health_check_worker.NewHcWorker(workerWorkingCtx).Start()
		}

		// end
		waitGroup.Wait()
	},
}

func routineHotReload(ctx *config.AppContext) {
	logger := ctx.Logger
	defer libapp.TryRecoverAndExecuteExitFunctionIfRecovered(logger)

	hotReloadInterval := ctx.AppConfig.General.HotReloadInterval
	if hotReloadInterval < 30*time.Second {
		hotReloadInterval = time.Minute
	}
	firstRun := true

	for {
		if firstRun {
			firstRun = false
		} else {
			time.Sleep(hotReloadInterval)
		}
		logger.Debug("hot-reload config")

		// Reload users config
		usersConf := func() *config.UsersConfig {
			usersConf, err := config.LoadUsersConfig(homeDir)
			if err != nil {
				logger.Error("failed to hot-reload users config, failed to load", "error", err.Error())
				return nil
			}
			userRecords := usersConf.ToUserRecords()
			if err := userRecords.Validate(); err != nil {
				logger.Error("failed to hot-reload users config, validation failed", "error", err.Error())
				return nil
			}

			// Update users config
			if err := usereg.UpdateUsersConfigWL(userRecords); err != nil {
				logger.Error("failed to hot-reload users config, failed to update registry", "error", err.Error())
				return nil
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

			return usersConf
		}()

		// Reload chains config
		if usersConf != nil {
			func(usersConf *config.UsersConfig) {
				chainsConf, err := config.LoadChainsConfig(homeDir)
				if err != nil {
					logger.Error("failed to hot-reload chains config, failed to load", "error", err.Error())
					return
				}
				if err := chainsConf.Validate(usersConf); err != nil {
					logger.Error("failed to hot-reload chains config, validation failed", "error", err.Error())
					return
				}

				// Update chains config
				if err := chainreg.UpdateChainsConfigWL(chainsConf, usersConf); err != nil {
					logger.Error("failed to hot-reload chains config, failed to update registry", "error", err.Error())
					return
				}
			}(usersConf)
		}
	}
}

func routineInformStartup(ctx *config.AppContext) {
	logger := ctx.Logger
	defer libapp.TryRecoverAndExecuteExitFunctionIfRecovered(logger)

	time.Sleep(1 * time.Second)
	for {
		time.Sleep(1 * time.Second)

		if len(tbotreg.GetAllTelegramBotsRL()) > 0 {
			time.Sleep(2 * time.Second) // wait for all telegram bots to be initialized
			break
		}

		logger.Info("waiting for telegram bots to be initialized")
	}

	safeSendTelegramMessageToAll(ctx, "startup", "validator health-check bot is started", false)
}

func safeShutdownTelegram(ctx *config.AppContext) {
	tbotreg.FlagShuttingDownWL()
	safeSendTelegramMessageToAll(ctx, "shutdown", "validator health-check bot is shutting down", true)
}

func safeSendTelegramMessageToAll(ctx *config.AppContext, action string, message string, stop bool) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.Error("panic occurred during send message to all users", "error", r)
		}
	}()

	for _, bot := range tbotreg.GetAllTelegramBotsRL().SortByPriority() {
		if stop {
			bot.StopReceivingUpdates()
		}

		chatIds := bot.GetAllChainIdsRL()
		if len(chatIds) > 0 {
			if bot.IsPriorityRL() {
				_bot := bot.GetInnerTelegramBot()
				for _, chatId := range chatIds {
					_, err := utils.Retry[string](func() (string, error) {
						_, err := _bot.SendMessage(message, chatId)
						return "", err
					})
					if err != nil {
						ctx.Logger.Error("failed to send telegram message to chat", "chat-id", chatId, "action", action, "priority", bot.IsPriorityRL(), "error", err.Error())
					}
				}
			} else {
				err := bot.GetInnerTelegramBot().SendMessageToMultipleChats(message, chatIds, nil)
				if err != nil {
					ctx.Logger.Error("failed to send telegram message to multiple chats", "chat-count", len(chatIds), "action", action, "priority", bot.IsPriorityRL(), "error", err.Error())
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
