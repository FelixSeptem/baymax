package collab

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultConfigAndValidation(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Enabled {
		t.Fatal("default collab config should be disabled")
	}
	if cfg.DefaultAggregation != AggregationAllSettled {
		t.Fatalf("default aggregation = %q, want all_settled", cfg.DefaultAggregation)
	}
	if cfg.FailurePolicy != FailurePolicyFailFast {
		t.Fatalf("default failure policy = %q, want fail_fast", cfg.FailurePolicy)
	}
	if cfg.Retry.Enabled {
		t.Fatal("default retry.enabled should be false")
	}
	if cfg.Retry.MaxAttempts != 3 ||
		cfg.Retry.BackoffInitial != 100*time.Millisecond ||
		cfg.Retry.BackoffMax != 2*time.Second ||
		cfg.Retry.Multiplier != 2 ||
		cfg.Retry.JitterRatio != 0.2 ||
		cfg.Retry.RetryOn != RetryOnTransportOnly {
		t.Fatalf("default retry policy mismatch: %#v", cfg.Retry)
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("ValidateConfig(default) failed: %v", err)
	}
}

func TestValidateRetryConfigRejectsInvalidBounds(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Retry.MaxAttempts = 0
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected validation error for max_attempts")
	}

	cfg = DefaultConfig()
	cfg.Retry.BackoffInitial = 0
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected validation error for backoff_initial")
	}

	cfg = DefaultConfig()
	cfg.Retry.BackoffMax = 90 * time.Millisecond
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected validation error for backoff_max")
	}

	cfg = DefaultConfig()
	cfg.Retry.Multiplier = 1
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected validation error for multiplier")
	}

	cfg = DefaultConfig()
	cfg.Retry.JitterRatio = 1.5
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected validation error for jitter_ratio")
	}

	cfg = DefaultConfig()
	cfg.Retry.RetryOn = "all"
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("expected validation error for retry_on")
	}
}

func TestRetryDelayRespectsBoundsAndDeterminism(t *testing.T) {
	policy := RetryConfig{
		Enabled:        true,
		MaxAttempts:    3,
		BackoffInitial: 100 * time.Millisecond,
		BackoffMax:     200 * time.Millisecond,
		Multiplier:     2,
		JitterRatio:    0,
		RetryOn:        RetryOnTransportOnly,
	}

	d1 := RetryDelay(policy, 1, "task-a")
	d2 := RetryDelay(policy, 2, "task-a")
	d3 := RetryDelay(policy, 3, "task-a")
	if d1 != 100*time.Millisecond {
		t.Fatalf("attempt1 delay=%v, want 100ms", d1)
	}
	if d2 != 200*time.Millisecond {
		t.Fatalf("attempt2 delay=%v, want 200ms", d2)
	}
	if d3 != 200*time.Millisecond {
		t.Fatalf("attempt3 delay=%v, want 200ms (capped)", d3)
	}

	policy.JitterRatio = 0.2
	j1 := RetryDelay(policy, 2, "task-a")
	j2 := RetryDelay(policy, 2, "task-a")
	if j1 != j2 {
		t.Fatalf("jitter must be deterministic, got %v and %v", j1, j2)
	}
	if j1 <= 0 || j1 > 200*time.Millisecond {
		t.Fatalf("jittered delay out of bounds: %v", j1)
	}
}

func TestExecuteAllSettledAndFirstSuccess(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true

	allSettled, err := Execute(context.Background(), cfg, Request{
		Primitive: PrimitiveAggregation,
		Strategy:  AggregationAllSettled,
		Policy:    FailurePolicyBestEffort,
		Aggregation: []Branch{
			{
				ID:       "b1",
				Required: true,
				Execute: func(context.Context) (Outcome, error) {
					return Outcome{Status: StatusSucceeded}, nil
				},
			},
			{
				ID:       "b2",
				Required: false,
				Execute: func(context.Context) (Outcome, error) {
					return Outcome{Status: StatusFailed, Error: "boom"}, nil
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute(all_settled) failed: %v", err)
	}
	if allSettled.Outcome.Status != StatusFailed {
		t.Fatalf("all_settled outcome = %q, want failed", allSettled.Outcome.Status)
	}
	if len(allSettled.Branches) != 2 {
		t.Fatalf("all_settled branch count = %d, want 2", len(allSettled.Branches))
	}

	firstSuccess, err := Execute(context.Background(), cfg, Request{
		Primitive: PrimitiveAggregation,
		Strategy:  AggregationFirstSuccess,
		Policy:    FailurePolicyBestEffort,
		Aggregation: []Branch{
			{
				ID:       "b1",
				Required: true,
				Execute: func(context.Context) (Outcome, error) {
					return Outcome{Status: StatusFailed, Error: "e1"}, nil
				},
			},
			{
				ID:       "b2",
				Required: true,
				Execute: func(context.Context) (Outcome, error) {
					return Outcome{Status: StatusSucceeded}, nil
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute(first_success) failed: %v", err)
	}
	if firstSuccess.Outcome.Status != StatusSucceeded {
		t.Fatalf("first_success outcome = %q, want succeeded", firstSuccess.Outcome.Status)
	}
	if len(firstSuccess.Branches) != 2 {
		t.Fatalf("first_success branch count = %d, want 2", len(firstSuccess.Branches))
	}
}

func TestExecuteFailFastAndNormalizeStatus(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true

	out, err := Execute(context.Background(), cfg, Request{
		Primitive: PrimitiveDelegation,
		Aggregation: []Branch{
			{
				ID:       "required",
				Required: true,
				Execute: func(context.Context) (Outcome, error) {
					return Outcome{Status: StatusPending}, errors.New("network")
				},
			},
			{
				ID:       "tail",
				Required: true,
				Execute: func(context.Context) (Outcome, error) {
					return Outcome{Status: StatusSucceeded}, nil
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute(fail_fast) failed: %v", err)
	}
	if out.Outcome.Status != StatusFailed {
		t.Fatalf("fail_fast outcome = %q, want failed", out.Outcome.Status)
	}
	if len(out.Branches) != 1 {
		t.Fatalf("fail_fast should short-circuit branches, got %d", len(out.Branches))
	}
	if out.Branches[0].Outcome.Status != StatusFailed {
		t.Fatalf("branch status should normalize to failed, got %q", out.Branches[0].Outcome.Status)
	}
	if NormalizeStatus(Status("submitted")) != StatusPending {
		t.Fatal("submitted should normalize to pending")
	}
}

func TestExecuteRetriesOnlyRetryableOutcomeWhenEnabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Retry.Enabled = true
	cfg.Retry.MaxAttempts = 3
	cfg.Retry.BackoffInitial = time.Millisecond
	cfg.Retry.BackoffMax = 2 * time.Millisecond
	cfg.Retry.JitterRatio = 0

	attempts := 0
	res, err := Execute(context.Background(), cfg, Request{
		Primitive: PrimitiveDelegation,
		Aggregation: []Branch{{
			ID:       "r",
			Required: true,
			Execute: func(context.Context) (Outcome, error) {
				attempts++
				if attempts < 3 {
					return Outcome{Status: StatusFailed, Retryable: true, Error: "connection reset"}, nil
				}
				return Outcome{Status: StatusSucceeded, Payload: map[string]any{"ok": true}}, nil
			},
		}},
	})
	if err != nil {
		t.Fatalf("Execute retryable failed: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts=%d, want 3", attempts)
	}
	if res.Outcome.Status != StatusSucceeded {
		t.Fatalf("outcome status=%q, want succeeded", res.Outcome.Status)
	}
	if got := res.Branches[0].Outcome.Payload["collab_retry_attempts"]; got != 2 {
		t.Fatalf("collab_retry_attempts=%v, want 2", got)
	}

	attempts = 0
	_, err = Execute(context.Background(), cfg, Request{
		Primitive: PrimitiveDelegation,
		Aggregation: []Branch{{
			ID:       "nr",
			Required: true,
			Execute: func(context.Context) (Outcome, error) {
				attempts++
				return Outcome{Status: StatusFailed, Retryable: false, Error: "validation failed"}, nil
			},
		}},
	})
	if err != nil {
		t.Fatalf("Execute non-retryable failed: %v", err)
	}
	if attempts != 1 {
		t.Fatalf("non-retryable attempts=%d, want 1", attempts)
	}
}
