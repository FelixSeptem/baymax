package reliability

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecuteRetryThenSuccess(t *testing.T) {
	var attempts int32
	got, finalAttempt, err := Execute(context.Background(), RetryConfig{
		Attempts: 2,
		Timeout:  time.Second,
		Backoff:  time.Millisecond,
	}, RetryHooks[int]{
		Invoke: func(ctx context.Context, attempt int) (int, error) {
			if atomic.AddInt32(&attempts, 1) == 1 {
				return 0, errors.New("temporary")
			}
			return 42, nil
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if got != 42 {
		t.Fatalf("result = %d, want 42", got)
	}
	if finalAttempt != 1 {
		t.Fatalf("final attempt = %d, want 1", finalAttempt)
	}
}

func TestExecuteTimeout(t *testing.T) {
	_, _, err := Execute(context.Background(), RetryConfig{
		Attempts: 1,
		Timeout:  10 * time.Millisecond,
	}, RetryHooks[int]{
		Invoke: func(ctx context.Context, attempt int) (int, error) {
			<-ctx.Done()
			return 0, ctx.Err()
		},
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want deadline exceeded", err)
	}
}
