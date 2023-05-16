package google

import (
	"context"
	"errors"
	"time"

	"github.com/cenkalti/backoff/v4"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/googleapi"
)

const maxRetries = 10

func retry[T any](ctx context.Context, call func() (T, error)) (T, error) {
	b := backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxRetries), ctx)
	return backoff.RetryNotifyWithData(func() (T, error) {
		result, err := call()

		if isRetryable(err) {
			return result, err
		}
		return result, backoff.Permanent(err)
	}, b, func(err error, d time.Duration) {
		log.WithError(err).WithField("delay", d).Debugf("will retry error")
	})
}

func isRetryable(err error) bool {
	// be pesimistic for now and just retry rateLimit errors
	var gerr *googleapi.Error
	if errors.As(err, &gerr) {
		if firstErrorReason(gerr) == "rateLimitExceeded" {
			return true
		}
	}
	return false
}

func firstErrorReason(err *googleapi.Error) string {
	if len(err.Errors) == 0 {
		return ""
	}
	return err.Errors[0].Reason
}
