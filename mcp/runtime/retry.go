package runtime

import (
	"context"
	"errors"
	"time"
)

type RetryControlError struct {
	Err       error
	Retryable bool
}

func (e *RetryControlError) Error() string {
	if e == nil || e.Err == nil {
		return "retry control error"
	}
	return e.Err.Error()
}

func (e *RetryControlError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NonRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryControlError{Err: err, Retryable: false}
}

func ShouldRetry(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var control *RetryControlError
	if errors.As(err, &control) {
		return control.Retryable
	}
	return true
}

func BackoffAt(base time.Duration, attempt int) time.Duration {
	if attempt <= 0 {
		return base
	}
	d := base
	for i := 0; i < attempt; i++ {
		d *= 2
		if d > 2*time.Second {
			return 2 * time.Second
		}
	}
	return d
}
