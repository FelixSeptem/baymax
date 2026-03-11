package retry

import (
	"context"
	"errors"
	"time"
)

type ControlError struct {
	Err       error
	Retryable bool
}

func (e *ControlError) Error() string {
	if e == nil || e.Err == nil {
		return "retry control error"
	}
	return e.Err.Error()
}

func (e *ControlError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NonRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &ControlError{Err: err, Retryable: false}
}

func ShouldRetry(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var control *ControlError
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
