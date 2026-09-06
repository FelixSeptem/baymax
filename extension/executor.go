package extension

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	ReasonTimeout         = "extension.timeout"
	ReasonPanic           = "extension.panic"
	ReasonInvalidResult   = "extension.invalid_result"
	ReasonExecutionFailed = "extension.execution_failed"
)

type FailurePolicy string

const (
	FailurePolicySkip    FailurePolicy = "skip"
	FailurePolicyDeny    FailurePolicy = "deny"
	FailurePolicyDegrade FailurePolicy = "degrade"
)

type ExecutionOutcome string

const (
	ExecutionSucceeded ExecutionOutcome = "succeeded"
	ExecutionSkipped   ExecutionOutcome = "skipped"
	ExecutionDenied    ExecutionOutcome = "denied"
	ExecutionDegraded  ExecutionOutcome = "degraded"
)

type ExecutionOptions struct {
	Timeout       time.Duration
	FailurePolicy FailurePolicy
}

type ExecutionResult struct {
	Outcome ExecutionOutcome
	Reason  string
	Value   any
}

type Action func(context.Context) (any, error)

func Execute(parent context.Context, options ExecutionOptions, action Action) (result ExecutionResult) {
	if parent == nil {
		parent = context.Background()
	}
	if action == nil {
		return applyFailure(options.FailurePolicy, ReasonInvalidResult)
	}
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	type outcome struct {
		value any
		err   error
		panic any
	}
	done := make(chan outcome, 1)
	go func() {
		item := outcome{}
		defer func() { item.panic = recover(); done <- item }()
		item.value, item.err = action(ctx)
	}()
	select {
	case <-ctx.Done():
		return applyFailure(options.FailurePolicy, ReasonTimeout)
	case item := <-done:
		if item.panic != nil {
			return applyFailure(options.FailurePolicy, ReasonPanic)
		}
		if item.err != nil {
			if strings.Contains(strings.ToLower(item.err.Error()), "context deadline exceeded") || ctx.Err() == context.DeadlineExceeded {
				return applyFailure(options.FailurePolicy, ReasonTimeout)
			}
			return applyFailure(options.FailurePolicy, ReasonExecutionFailed)
		}
		if item.value == nil {
			return applyFailure(options.FailurePolicy, ReasonInvalidResult)
		}
		return ExecutionResult{Outcome: ExecutionSucceeded, Value: item.value}
	}
}

func applyFailure(policy FailurePolicy, reason string) ExecutionResult {
	switch policy {
	case FailurePolicyDegrade:
		return ExecutionResult{Outcome: ExecutionDegraded, Reason: reason}
	case FailurePolicySkip:
		return ExecutionResult{Outcome: ExecutionSkipped, Reason: reason}
	default:
		return ExecutionResult{Outcome: ExecutionDenied, Reason: reason}
	}
}

func (r ExecutionResult) Error() error {
	if r.Reason == "" || r.Outcome == ExecutionSucceeded {
		return nil
	}
	return fmt.Errorf("%s: %s", r.Outcome, r.Reason)
}
