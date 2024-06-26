package telegram_push_message_svc

//goland:noinspection SpellCheckingInspection
import (
	"fmt"
	libapp "github.com/EscanBE/go-lib/app"
	"github.com/bcdevtools/validator-health-check/config"
	"github.com/bcdevtools/validator-health-check/constants"
	"github.com/bcdevtools/validator-health-check/registry/telegram_bot_registry"
	"github.com/bcdevtools/validator-health-check/registry/user_registry"
	tptypes "github.com/bcdevtools/validator-health-check/services/telegram_push_message_svc/types"
	"github.com/bcdevtools/validator-health-check/utils"
	"github.com/pkg/errors"
	"sort"
	"strings"
	"sync"
	"time"
)

var telePusherSvc *telegramPusher

type telegramPusher struct {
	sync.RWMutex
	appCtx              config.AppContext
	queuesReceiverBased map[int64]ReceiverBasedQueue
	priorityQueue       []ReceiverBasedQueue
	nonPriorityQueue    []ReceiverBasedQueue
}

func StartTelegramPusherService(appCtx config.AppContext) {
	if telePusherSvc != nil {
		panic("to prevent API limit issue, only one instance of Telegram Pusher is allowed")
	}

	telePusherSvc = &telegramPusher{
		appCtx:              appCtx,
		queuesReceiverBased: make(map[int64]ReceiverBasedQueue),
	}

	go telePusherSvc.start()
}

func EnqueueMessageWL(message tptypes.QueueMessage) {
	telePusherSvc.enqueueMessageWL(message)
}

func (tp *telegramPusher) enqueueMessageWL(message tptypes.QueueMessage) {
	tp.Lock()
	defer tp.Unlock()

	existingQueue, exists := tp.queuesReceiverBased[message.ReceiverID]
	if !exists {
		existingQueue = newReceiverBasedQueue(message.ReceiverID, message.Priority)
		tp.queuesReceiverBased[message.ReceiverID] = existingQueue

		if message.Priority {
			tp.priorityQueue = append(tp.priorityQueue, existingQueue)
		} else {
			tp.nonPriorityQueue = append(tp.nonPriorityQueue, existingQueue)
		}
	}

	if message.EnqueueTimeUTC == (time.Time{}) {
		message.EnqueueTimeUTC = time.Now().UTC()
	}
	existingQueue.EnqueueMessageWL(message)
}

func (tp *telegramPusher) start() {
	logger := tp.appCtx.Logger
	defer libapp.TryRecoverAndExecuteExitFunctionIfRecovered(logger)

	var pushedPreviousTurn bool
	for {
		time.Sleep(300 * time.Millisecond)

		if pushedPreviousTurn {
			// sleep for a while to prevent API limit
			time.Sleep(10 * time.Second)
		}
		pushedPreviousTurn = false

		allQueues := tp.getAllQueuesRL()
		if len(allQueues) == 0 {
			continue
		}

		var firstNonEmptyQueue ReceiverBasedQueue
		for _, queue := range allQueues {
			if !queue.AnyPendingMessageRL() {
				continue
			}

			_, _, size, lastEnqueueTime := queue.GetQueueInfoRL()
			if size < 1 {
				continue
			}

			if time.Since(lastEnqueueTime) < constants.MINIMUM_BETWEEN_TELEGRAM_PUSH_SAME_USER {
				continue
			}

			firstNonEmptyQueue = queue
		}

		if firstNonEmptyQueue == nil {
			// all cool-down
			continue
		}

		receiverId := firstNonEmptyQueue.GetReceiverId()
		dequeuedMessages := firstNonEmptyQueue.DequeueMessagesWL(constants.BATCH_SIZE_TELEGRAM_PUSH_PER_USER)
		if len(dequeuedMessages) < 1 {
			logger.Error("unexpected no message", "receiver-id", receiverId)
			continue
		}

		var messages []tptypes.QueueMessage

		for _, message := range dequeuedMessages {
			if shouldSilentByChatIdRWL(receiverId, message.Message) {
				logger.Info("silenced message", "receiver-id", receiverId, "message", message.Message)
				continue
			}
			messages = append(messages, message)
		}

		if len(messages) < 1 {
			continue
		}

		sort.Slice(messages, func(i, j int) bool {
			left := messages[i]
			right := messages[j]

			if left.Fatal != right.Fatal {
				if left.Fatal {
					return true
				} else {
					return false
				}
			}

			return left.EnqueueTimeUTC.Before(right.EnqueueTimeUTC)
		})
		messagesContent := make([]string, len(messages))
		for i, message := range messages {
			messagesContent[i] = message.Message
		}
		combinedMessage := strings.Join(messagesContent, constants.BATCH_MESSAGES_LINE_DIVIDER)

		err := func(receiverId int64, messageContent string, messages []tptypes.QueueMessage) error {
			var sent bool
			defer func() {
				if !sent {
					// re-enqueue
					for _, message := range messages {
						tp.enqueueMessageWL(message)
					}
				}
			}()

			userRecord, found := user_registry.GetUserRecordByTelegramUserIdRL(receiverId)
			if !found {
				return fmt.Errorf("user record not found for receiver id %d", receiverId)
			}

			if userRecord.TelegramConfig.IsEmptyOrIncompleteConfig() {
				return fmt.Errorf("telegram config is incomplete for user identity %s", userRecord.Identity)
			}

			bot, err := telegram_bot_registry.GetTelegramBotByTokenWL(userRecord.TelegramConfig.Token, logger)
			if err != nil {
				return errors.Wrapf(err, "failed to get telegram bot for user identity %s", userRecord.Identity)
			}

			_, err = utils.Retry[string](func() (string, error) {
				_, err := bot.GetInnerTelegramBot().SendMessage(messageContent, receiverId)
				return "", err
			})
			if err != nil {
				return errors.Wrapf(err, "failed to send message to user identity %s", userRecord.Identity)
			}

			sent = true
			return nil
		}(receiverId, combinedMessage, messages)

		if err != nil {
			logger.Error("failed to push telegram message", "receiver", receiverId, "message-size", len(combinedMessage), "messages-count", len(messages), "error", err)
		}
	}
}

func (tp *telegramPusher) getAllQueuesRL() []ReceiverBasedQueue {
	tp.RLock()
	defer tp.RUnlock()

	return append(tp.priorityQueue, tp.nonPriorityQueue...)
}
