package types

import "time"

type QueueMessage struct {
	ReceiverID     int64
	Priority       bool
	Fatal          bool
	Message        string
	EnqueueTimeUTC time.Time
}
