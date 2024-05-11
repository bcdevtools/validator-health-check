package cmd

import (
	"fmt"
	libapp "github.com/EscanBE/go-lib/app"
	libcons "github.com/EscanBE/go-lib/constants"
	logtypes "github.com/EscanBE/go-lib/logging/types"
	libbot "github.com/EscanBE/go-lib/telegram/bot"
	libutils "github.com/EscanBE/go-lib/utils"
	"github.com/bcdevtools/validator-health-check/config"
	"github.com/bcdevtools/validator-health-check/constants"
	"github.com/bcdevtools/validator-health-check/work"
	workertypes "github.com/bcdevtools/validator-health-check/work/types"
	"github.com/pkg/errors"
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

		conf, err := config.LoadConfig(homeDir)
		libutils.ExitIfErr(err, "unable to load configuration")

		// Output some options to console
		conf.PrintOptions()

		// Perform validation
		err = conf.Validate()
		libutils.ExitIfErr(err, "failed to validate configuration")

		// Initialize bot
		var bot *libbot.TelegramBot
		if len(conf.SecretConfig.TelegramToken) > 0 {
			bot, err = libbot.NewBot(conf.SecretConfig.TelegramToken)
			if err != nil {
				panic(errors.Wrap(err, "Failed to initialize Telegram bot"))
			}
			bot.EnableDebug(conf.Logging.Level == logtypes.LOG_LEVEL_DEBUG)
		}

		// Init execution context
		ctx := config.NewAppContext(conf, bot)
		logger := ctx.Logger

		logger.Debug("Application starts")

		_, _ = ctx.SendTelegramLogMessage(fmt.Sprintf("[%s] Application Start", constants.APP_NAME))

		// Increase the waitGroup by one and decrease within trapExitSignal
		waitGroup.Add(1)

		// Register the function which should be executed upon exit.
		// After register, when you want to clean-up things before exit,
		// call libapp.ExecuteExitFunction(ctx) the same was as trapExitSignal method did
		libapp.RegisterExitFunction(func(params ...any) {
			// finalize
			defer waitGroup.Done()

			if ctx.Bot != nil {
				ctx.Bot.StopReceivingUpdates()
			}

			// Implements close connection, resources,... here to prevent resource leak

		})

		// Listen for and trap any OS signal to gracefully shutdown and exit
		trapExitSignal(ctx)

		// Create workers
		// Worker defines a job consumer that is responsible for getting assigned tasks and process business logic
		// to assign task to workers, use a channel

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
