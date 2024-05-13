package constants

import "time"

//goland:noinspection GoSnakeCaseUsage
const (
	MINIMUM_WORKER_HEALTH_CHECK = 5
)

//goland:noinspection GoSnakeCaseUsage
const (
	MINIMUM_BETWEEN_TELEGRAM_PUSH_SAME_USER = 1 * time.Minute

	BATCH_SIZE_TELEGRAM_PUSH_PER_USER = 20

	BATCH_MESSAGES_LINE_DIVIDER = "\n---\n"

	INFORM_TELEGRAM_IF_BLOCK_OLDER_THAN = 3 * time.Minute
)
