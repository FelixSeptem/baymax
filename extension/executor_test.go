package extension

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExecuteBoundedHookTimeoutAppliesSkipPolicy(t *testing.T) {
	result := Execute(context.Background(), ExecutionOptions{Timeout: 10 * time.Millisecond, FailurePolicy: FailurePolicySkip}, func(ctx context.Context) (any, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	if result.Outcome != ExecutionSkipped || result.Reason != ReasonTimeout {
		t.Fatalf("result=%#v, want skipped timeout", result)
	}
}

func TestExecuteRecoversPanicAndAppliesDenyPolicy(t *testing.T) {
	result := Execute(context.Background(), ExecutionOptions{FailurePolicy: FailurePolicyDeny}, func(context.Context) (any, error) {
		panic("boom")
	})
	if result.Outcome != ExecutionDenied || result.Reason != ReasonPanic {
		t.Fatalf("result=%#v, want denied panic", result)
	}
}

func TestExecuteRejectsInvalidResultAndAppliesDegradePolicy(t *testing.T) {
	result := Execute(context.Background(), ExecutionOptions{FailurePolicy: FailurePolicyDegrade}, func(context.Context) (any, error) {
		return nil, nil
	})
	if result.Outcome != ExecutionDegraded || result.Reason != ReasonInvalidResult {
		t.Fatalf("result=%#v, want degraded invalid result", result)
	}
}

func TestExecutePropagatesSuccessAndErrors(t *testing.T) {
	success := Execute(context.Background(), ExecutionOptions{}, func(context.Context) (any, error) { return "ok", nil })
	if success.Outcome != ExecutionSucceeded || success.Value != "ok" {
		t.Fatalf("success=%#v", success)
	}
	failure := Execute(context.Background(), ExecutionOptions{FailurePolicy: FailurePolicyDeny}, func(context.Context) (any, error) { return nil, errors.New("failed") })
	if failure.Outcome != ExecutionDenied || failure.Reason != ReasonExecutionFailed {
		t.Fatalf("failure=%#v", failure)
	}
}
