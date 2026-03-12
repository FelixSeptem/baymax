package reliability

import (
	"context"
	"errors"
	"time"

	mcpretry "github.com/FelixSeptem/baymax/mcp/retry"
)

type RetryConfig struct {
	Attempts int
	Timeout  time.Duration
	Backoff  time.Duration
}

type RetryHooks[T any] struct {
	Invoke      func(ctx context.Context, attempt int) (T, error)
	ShouldRetry func(err error) bool
	OnRetry     func(ctx context.Context, attempt int, err error) error
}

func Execute[T any](ctx context.Context, cfg RetryConfig, hooks RetryHooks[T]) (T, int, error) {
	var zero T
	if hooks.Invoke == nil {
		return zero, 0, errors.New("retry invoke hook is nil")
	}
	if cfg.Attempts <= 0 {
		cfg.Attempts = 1
	}
	if hooks.ShouldRetry == nil {
		hooks.ShouldRetry = mcpretry.ShouldRetry
	}

	var lastErr error
	for attempt := 0; attempt < cfg.Attempts; attempt++ {
		stepCtx := ctx
		cancel := func() {}
		if cfg.Timeout > 0 {
			stepCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		}
		res, err := hooks.Invoke(stepCtx, attempt)
		cancel()
		if err == nil {
			return res, attempt, nil
		}

		lastErr = err
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(stepCtx.Err(), context.DeadlineExceeded) {
			return zero, attempt, context.DeadlineExceeded
		}
		if attempt >= cfg.Attempts-1 || !hooks.ShouldRetry(err) {
			return zero, attempt, err
		}
		if hooks.OnRetry != nil {
			if hookErr := hooks.OnRetry(ctx, attempt, err); hookErr != nil {
				lastErr = hookErr
			}
		}
		select {
		case <-ctx.Done():
			return zero, attempt, ctx.Err()
		case <-time.After(mcpretry.BackoffAt(cfg.Backoff, attempt)):
		}
	}
	return zero, cfg.Attempts - 1, lastErr
}
