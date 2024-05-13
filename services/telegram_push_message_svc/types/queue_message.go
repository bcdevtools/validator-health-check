package types

type QueueMessage struct {
	ReceiverID int64
	Priority   bool // TODO remove if unused
	Message    string
}
