package invoke

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
)

type fakeClient struct {
	submitFn     func(context.Context, a2a.TaskRequest) (a2a.TaskRecord, error)
	waitFn       func(context.Context, string, time.Duration, func(context.Context, a2a.TaskRecord) error) (a2a.TaskRecord, error)
	lastInterval time.Duration
}

func (f *fakeClient) Submit(ctx context.Context, req a2a.TaskRequest) (a2a.TaskRecord, error) {
	if f.submitFn != nil {
		return f.submitFn(ctx, req)
	}
	return a2a.TaskRecord{TaskID: req.TaskID}, nil
}

func (f *fakeClient) WaitResult(
	ctx context.Context,
	taskID string,
	pollInterval time.Duration,
	callback func(context.Context, a2a.TaskRecord) error,
) (a2a.TaskRecord, error) {
	f.lastInterval = pollInterval
	if f.waitFn != nil {
		return f.waitFn(ctx, taskID, pollInterval, callback)
	}
	return a2a.TaskRecord{
		TaskID: taskID,
		Status: a2a.StatusSucceeded,
		Result: map[string]any{"ok": true},
	}, nil
}

func TestInvokeSyncSuccessUsesDefaultPollInterval(t *testing.T) {
	client := &fakeClient{}
	out, err := invokeSync(context.Background(), client, Request{
		TaskID: "task-1",
		Method: "workflow.dispatch",
	})
	if err != nil {
		t.Fatalf("InvokeSync failed: %v", err)
	}
	if out.TerminalStatus != a2a.StatusSucceeded {
		t.Fatalf("terminal status = %q, want succeeded", out.TerminalStatus)
	}
	if out.Error != nil {
		t.Fatalf("error should be nil, got %#v", out.Error)
	}
	if client.lastInterval != DefaultPollInterval {
		t.Fatalf("default poll interval = %s, want %s", client.lastInterval, DefaultPollInterval)
	}
}

func TestInvokeSyncContextTimeoutHasPriority(t *testing.T) {
	client := &fakeClient{
		waitFn: func(ctx context.Context, _ string, _ time.Duration, _ func(context.Context, a2a.TaskRecord) error) (a2a.TaskRecord, error) {
			<-ctx.Done()
			return a2a.TaskRecord{}, ctx.Err()
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	out, err := invokeSync(ctx, client, Request{TaskID: "task-timeout"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if out.Error == nil {
		t.Fatal("expected normalized error for timeout")
	}
	if out.Error.Layer != string(a2a.ErrorLayerTransport) {
		t.Fatalf("timeout layer = %q, want transport", out.Error.Layer)
	}
	if !out.Error.Retryable {
		t.Fatal("timeout should be retryable")
	}
}

func TestInvokeSyncContextCanceledHasPriority(t *testing.T) {
	client := &fakeClient{
		waitFn: func(ctx context.Context, _ string, _ time.Duration, _ func(context.Context, a2a.TaskRecord) error) (a2a.TaskRecord, error) {
			<-ctx.Done()
			return a2a.TaskRecord{}, ctx.Err()
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := invokeSync(ctx, client, Request{TaskID: "task-cancel"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled, got %v", err)
	}
}

func TestInvokeSyncNormalizesTransportProtocolSemanticErrors(t *testing.T) {
	t.Run("transport", func(t *testing.T) {
		client := &fakeClient{
			submitFn: func(context.Context, a2a.TaskRequest) (a2a.TaskRecord, error) {
				return a2a.TaskRecord{}, context.DeadlineExceeded
			},
		}
		out, err := invokeSync(context.Background(), client, Request{TaskID: "task-transport"})
		if err == nil {
			t.Fatal("expected error")
		}
		if out.Error == nil || out.Error.Layer != string(a2a.ErrorLayerTransport) || !out.Error.Retryable {
			t.Fatalf("transport normalization mismatch: %#v", out.Error)
		}
	})

	t.Run("protocol", func(t *testing.T) {
		client := &fakeClient{
			waitFn: func(context.Context, string, time.Duration, func(context.Context, a2a.TaskRecord) error) (a2a.TaskRecord, error) {
				return a2a.TaskRecord{}, errors.New("unsupported method")
			},
		}
		out, err := invokeSync(context.Background(), client, Request{TaskID: "task-protocol"})
		if err == nil {
			t.Fatal("expected error")
		}
		if out.Error == nil || out.Error.Layer != string(a2a.ErrorLayerProtocol) || out.Error.Retryable {
			t.Fatalf("protocol normalization mismatch: %#v", out.Error)
		}
	})

	t.Run("semantic", func(t *testing.T) {
		client := &fakeClient{
			waitFn: func(context.Context, string, time.Duration, func(context.Context, a2a.TaskRecord) error) (a2a.TaskRecord, error) {
				return a2a.TaskRecord{
					TaskID:       "task-semantic",
					Status:       a2a.StatusFailed,
					ErrorMessage: "invalid payload",
				}, nil
			},
		}
		out, err := invokeSync(context.Background(), client, Request{TaskID: "task-semantic"})
		if err != nil {
			t.Fatalf("failed terminal should not return invocation error: %v", err)
		}
		if out.Error == nil {
			t.Fatal("expected normalized terminal error")
		}
		if out.Error.Layer != string(a2a.ErrorLayerSemantic) {
			t.Fatalf("semantic layer mismatch: %#v", out.Error)
		}
		if out.Error.Retryable {
			t.Fatalf("semantic error should not be retryable: %#v", out.Error)
		}
	})
}
