package collab

import (
	"context"
	"errors"
	"testing"
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
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("ValidateConfig(default) failed: %v", err)
	}

	cfg.Retry.Enabled = true
	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("ValidateConfig should reject primitive-layer retry enabled")
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
