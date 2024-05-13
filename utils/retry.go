package utils

import (
	"time"
)

type RetryOption struct {
	MinRetryCount        int
	MaximumRetryDuration time.Duration
}

func DefaultRetryOption() RetryOption {
	return RetryOption{
		MinRetryCount:        3,
		MaximumRetryDuration: 5 * time.Second,
	}
}

func (m RetryOption) MinCount(minRetryCount int) RetryOption {
	m.MinRetryCount = minRetryCount
	return m
}

func (m RetryOption) MaxDuration(maximumRetryDuration time.Duration) RetryOption {
	m.MaximumRetryDuration = maximumRetryDuration
	return m
}

var defaultRetryOption = DefaultRetryOption()

func Retry[T any](
	f func() (T, error),
	retryOption ...RetryOption,
) (res T, err error) {
	startTime := time.Now().UTC()
	tryCount := -1

	if len(retryOption) == 0 {
		retryOption = []RetryOption{defaultRetryOption}
	}

	minRetryCount := retryOption[0].MinRetryCount
	maximumRetryDuration := retryOption[0].MaximumRetryDuration

	var firstErr error
	for {
		tryCount++

		if tryCount > 0 {
			time.Sleep(100 * time.Millisecond)
		}

		res, err = f()
		if err == nil {
			return
		}

		if firstErr == nil {
			firstErr = err
		}

		if tryCount < minRetryCount {
			continue
		}

		if time.Since(startTime) < maximumRetryDuration {
			continue
		}

		break
	}

	err = firstErr
	return
}
