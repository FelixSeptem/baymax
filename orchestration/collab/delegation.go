package collab

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/orchestration/invoke"
)

type DelegationRequest struct {
	TaskID       string
	WorkflowID   string
	TeamID       string
	StepID       string
	AttemptID    string
	AgentID      string
	PeerID       string
	Method       string
	Payload      map[string]any
	PollInterval int64
}

type DelegationAsyncAck struct {
	TaskID     string
	WorkflowID string
	TeamID     string
	StepID     string
	PeerID     string
}

func DelegateSync(ctx context.Context, client invoke.Client, req invoke.Request) (Outcome, error) {
	return DelegateSyncWithRetry(ctx, client, req, RetryConfig{Enabled: false}, nil)
}

func DelegateSyncWithRetry(
	ctx context.Context,
	client invoke.Client,
	req invoke.Request,
	retry RetryConfig,
	observer RetryObserver,
) (Outcome, error) {
	policy, err := resolveRetryConfig(DefaultConfig().Retry, retry)
	if err != nil {
		return Outcome{Status: StatusFailed, Error: err.Error()}, err
	}
	attempt := 1
	retryCount := 0
	for {
		outcome, callErr := delegateSyncOnce(ctx, client, req)
		if !shouldRetryOutcome(policy, outcome) {
			if retryCount > 0 && outcome.Status == StatusSucceeded {
				emitRetryEvent(observer, RetryEvent{
					Type:        RetryEventSuccess,
					Attempt:     attempt,
					MaxAttempts: policy.MaxAttempts,
					RetryOn:     policy.RetryOn,
				})
			}
			return withRetryPayload(outcome, retryCount, false), callErr
		}
		if attempt >= policy.MaxAttempts {
			emitRetryEvent(observer, RetryEvent{
				Type:         RetryEventExhausted,
				Attempt:      attempt,
				MaxAttempts:  policy.MaxAttempts,
				RetryOn:      policy.RetryOn,
				Retryable:    true,
				ErrorMessage: strings.TrimSpace(outcome.Error),
			})
			return withRetryPayload(outcome, retryCount, true), callErr
		}
		delay := RetryDelay(
			policy,
			attempt,
			"delegation_sync",
			strings.TrimSpace(req.WorkflowID),
			strings.TrimSpace(req.TeamID),
			strings.TrimSpace(req.StepID),
			strings.TrimSpace(req.TaskID),
			strings.TrimSpace(req.PeerID),
		)
		emitRetryEvent(observer, RetryEvent{
			Type:         RetryEventAttempt,
			Attempt:      attempt,
			MaxAttempts:  policy.MaxAttempts,
			Delay:        delay,
			RetryOn:      policy.RetryOn,
			Retryable:    true,
			ErrorMessage: strings.TrimSpace(outcome.Error),
		})
		if waitErr := waitWithContext(ctx, delay); waitErr != nil {
			return withRetryPayload(outcome, retryCount, false), waitErr
		}
		retryCount++
		attempt++
	}
}

func delegateSyncOnce(ctx context.Context, client invoke.Client, req invoke.Request) (Outcome, error) {
	if client == nil {
		err := errors.New("a2a client is not configured")
		return Outcome{Status: StatusFailed, Error: err.Error()}, err
	}
	bridge, err := invoke.NewInMemoryMailboxBridge()
	if err != nil {
		return Outcome{Status: StatusFailed, Error: err.Error()}, err
	}
	outcome, err := bridge.InvokeSync(ctx, client, req)
	if err != nil {
		return Outcome{
			Status:    StatusFailed,
			Retryable: outcome.Error != nil && outcome.Error.Retryable,
			Error:     normalizeDelegationError(err, outcome),
		}, err
	}
	if outcome.TerminalStatus == a2a.StatusSucceeded {
		return Outcome{
			Status:  StatusSucceeded,
			Payload: cloneMap(outcome.Result),
		}, nil
	}
	normalized := normalizeDelegationError(nil, outcome)
	if normalized == "" {
		normalized = fmt.Sprintf("a2a task status %q", outcome.TerminalStatus)
	}
	return Outcome{
		Status:    NormalizeStatus(Status(outcome.TerminalStatus)),
		Retryable: outcome.Error != nil && outcome.Error.Retryable,
		Error:     normalized,
		Payload:   cloneMap(outcome.Result),
	}, nil
}

func DelegateAsync(ctx context.Context, client invoke.AsyncClient, req invoke.AsyncRequest, sink a2a.ReportSink) (DelegationAsyncAck, error) {
	return DelegateAsyncWithRetry(ctx, client, req, sink, RetryConfig{Enabled: false}, nil)
}

func DelegateAsyncWithRetry(
	ctx context.Context,
	client invoke.AsyncClient,
	req invoke.AsyncRequest,
	sink a2a.ReportSink,
	retry RetryConfig,
	observer RetryObserver,
) (DelegationAsyncAck, error) {
	policy, err := resolveRetryConfig(DefaultConfig().Retry, retry)
	if err != nil {
		return DelegationAsyncAck{}, err
	}
	attempt := 1
	for {
		ack, submitErr := delegateAsyncOnce(ctx, client, req, sink)
		if submitErr == nil {
			if attempt > 1 {
				emitRetryEvent(observer, RetryEvent{
					Type:        RetryEventSuccess,
					Attempt:     attempt,
					MaxAttempts: policy.MaxAttempts,
					RetryOn:     policy.RetryOn,
				})
			}
			return ack, nil
		}
		if !shouldRetryAsyncSubmitError(policy, submitErr) {
			return DelegationAsyncAck{}, submitErr
		}
		if attempt >= policy.MaxAttempts {
			emitRetryEvent(observer, RetryEvent{
				Type:         RetryEventExhausted,
				Attempt:      attempt,
				MaxAttempts:  policy.MaxAttempts,
				RetryOn:      policy.RetryOn,
				Retryable:    true,
				ErrorMessage: strings.TrimSpace(submitErr.Error()),
			})
			return DelegationAsyncAck{}, submitErr
		}
		delay := RetryDelay(
			policy,
			attempt,
			"delegation_async_submit",
			strings.TrimSpace(req.WorkflowID),
			strings.TrimSpace(req.TeamID),
			strings.TrimSpace(req.StepID),
			strings.TrimSpace(req.TaskID),
			strings.TrimSpace(req.PeerID),
		)
		emitRetryEvent(observer, RetryEvent{
			Type:         RetryEventAttempt,
			Attempt:      attempt,
			MaxAttempts:  policy.MaxAttempts,
			Delay:        delay,
			RetryOn:      policy.RetryOn,
			Retryable:    true,
			ErrorMessage: strings.TrimSpace(submitErr.Error()),
		})
		if waitErr := waitWithContext(ctx, delay); waitErr != nil {
			return DelegationAsyncAck{}, waitErr
		}
		attempt++
	}
}

func delegateAsyncOnce(ctx context.Context, client invoke.AsyncClient, req invoke.AsyncRequest, sink a2a.ReportSink) (DelegationAsyncAck, error) {
	bridge, err := invoke.NewInMemoryMailboxBridge()
	if err != nil {
		return DelegationAsyncAck{}, err
	}
	ack, err := bridge.InvokeAsync(ctx, client, req, sink)
	if err != nil {
		return DelegationAsyncAck{}, err
	}
	return DelegationAsyncAck{
		TaskID:     strings.TrimSpace(ack.TaskID),
		WorkflowID: strings.TrimSpace(ack.WorkflowID),
		TeamID:     strings.TrimSpace(ack.TeamID),
		StepID:     strings.TrimSpace(ack.StepID),
		PeerID:     strings.TrimSpace(ack.PeerID),
	}, nil
}

func shouldRetryAsyncSubmitError(policy RetryConfig, err error) bool {
	if !policy.Enabled || err == nil {
		return false
	}
	_, layer, _ := a2a.ClassifyError(err)
	if policy.RetryOn == RetryOnTransportOnly {
		return layer == a2a.ErrorLayerTransport
	}
	return false
}

func emitRetryEvent(observer RetryObserver, ev RetryEvent) {
	if observer == nil {
		return
	}
	observer(ev)
}

func normalizeDelegationError(err error, outcome invoke.Outcome) string {
	if err != nil {
		return strings.TrimSpace(err.Error())
	}
	if outcome.Error != nil && strings.TrimSpace(outcome.Error.Message) != "" {
		return strings.TrimSpace(outcome.Error.Message)
	}
	return ""
}
