package telegram_push_message_svc

import (
	"fmt"
	"github.com/bcdevtools/validator-health-check/constants"
	"github.com/bcdevtools/validator-health-check/services/telegram_push_message_svc/types"
	"sync"
	"time"
)

type ReceiverBasedQueue interface {
	EnqueueMessageWL(types.QueueMessage)
	AnyPendingMessageRL() bool
	GetQueueInfoRL() (receiver int64, isReceiverPriority bool, size int, lastEnqueueUTC time.Time)
	DequeueMessagesWL(size int) []types.QueueMessage
	GetReceiverId() int64
}

var _ ReceiverBasedQueue = &receiverBasedQueue{}

type receiverBasedQueue struct {
	sync.RWMutex
	receiverId       int64
	isPriority       bool
	lastEnqueueUTC   time.Time
	enqueuedMessages []types.QueueMessage
}

func newReceiverBasedQueue(receiverId int64, isPriority bool) ReceiverBasedQueue {
	return &receiverBasedQueue{
		receiverId:       receiverId,
		isPriority:       isPriority,
		enqueuedMessages: make([]types.QueueMessage, 0),
	}
}

func (r *receiverBasedQueue) EnqueueMessageWL(message types.QueueMessage) {
	if message.ReceiverID != r.receiverId {
		panic(fmt.Errorf("receiver id mismatch, expected %d, got %d", r.receiverId, message.ReceiverID))
	}

	r.Lock()
	defer r.Unlock()

	defer func() {
		r.lastEnqueueUTC = time.Now().UTC()
	}()

	copied := message
	r.enqueuedMessages = append(r.enqueuedMessages, copied)
}

func (r *receiverBasedQueue) AnyPendingMessageRL() bool {
	r.RLock()
	defer r.RUnlock()

	return len(r.enqueuedMessages) > 0
}

func (r *receiverBasedQueue) GetQueueInfoRL() (receiver int64, isReceiverPriority bool, size int, lastEnqueueUTC time.Time) {
	r.RLock()
	defer r.RUnlock()

	return r.receiverId, r.isPriority, len(r.enqueuedMessages), r.lastEnqueueUTC
}

func (r *receiverBasedQueue) DequeueMessagesWL(size int) (result []types.QueueMessage) {
	r.Lock()
	defer r.Unlock()

	const maximumTelegramMessageLength = 4096

	if size >= len(r.enqueuedMessages) {
		// take all
		result = r.enqueuedMessages[:]
		r.enqueuedMessages = make([]types.QueueMessage, 0)
	} else {
		result = r.enqueuedMessages[:size]
		r.enqueuedMessages = r.enqueuedMessages[size:]
	}

	defer func() {
		r.lastEnqueueUTC = time.Now().UTC() // prevent multi push that can reach telegram API limit
	}()

	var resultWithCheckLimitMessageSize []types.QueueMessage
	var cumulativeMessageLength int
	for i, message := range result {
		newCumulativeMessageLength := cumulativeMessageLength + len(message.Message)
		if i > 0 {
			newCumulativeMessageLength += len(constants.BATCH_MESSAGES_LINE_DIVIDER)
		}

		if newCumulativeMessageLength >= maximumTelegramMessageLength {
			break // do not enqueue
		}

		cumulativeMessageLength = newCumulativeMessageLength
		resultWithCheckLimitMessageSize = append(resultWithCheckLimitMessageSize, message)
	}

	if len(resultWithCheckLimitMessageSize) < len(result) {
		// threshold reached

		// put the rest back to queue
		r.enqueuedMessages = append(result[len(resultWithCheckLimitMessageSize):], r.enqueuedMessages...)
	}

	return
}

func (r *receiverBasedQueue) GetReceiverId() int64 {
	return r.receiverId
}
