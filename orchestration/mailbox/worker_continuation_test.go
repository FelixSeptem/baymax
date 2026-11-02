package mailbox

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestContinuationWorkerPausePreservesOriginalInFlightDelivery(t *testing.T) {
	ctx := context.Background()
	store, err := NewFileStore(filepath.Join(t.TempDir(), "mailbox.json"), Policy{})
	if err != nil {
		t.Fatal(err)
	}
	mb, err := New(store)
	if err != nil {
		t.Fatal(err)
	}
	binding := continuationTestBinding("pause-preserves")
	if _, err := mb.AdmitHostPlan(ctx, binding); err != nil {
		t.Fatal(err)
	}
	worker, err := NewContinuationWorker(mb, WorkerConfig{Enabled: true, InflightTimeout: time.Hour, HeartbeatInterval: time.Minute}, func(context.Context, Record) (HandlerResult, error) {
		return HandlerResult{Pause: &HandlerPause{
			Reason:                 MailboxPauseApprovalRequired,
			PendingActionReference: "effect:approve",
			CompletedEffectRefs:    []string{"effect:validated"},
			ConsumedBudget:         7,
			OriginalDeadline:       time.Now().UTC().Add(time.Hour),
			ExpiresAt:              time.Now().UTC().Add(time.Hour),
			CheckpointReference:    "handler-checkpoint:approval",
		}}, nil
	}, func(context.Context, MailboxContinuation, Record) (HandlerResult, error) {
		return HandlerResult{}, nil
	}, func(context.Context, MailboxContinuation) error { return nil }, "continuation-worker")
	if err != nil {
		t.Fatal(err)
	}
	processed, err := worker.RunOnce(ctx)
	if err != nil || !processed {
		t.Fatalf("RunOnce() processed=%v err=%v", processed, err)
	}
	page, err := mb.Query(ctx, QueryRequest{MessageID: binding.MessageID})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("Query() items=%d err=%v", len(page.Items), err)
	}
	record := page.Items[0]
	if record.State != StateInFlight || record.DeliveryAttempt != 1 || record.DeliveryID == "" || record.LeaseID == "" {
		t.Fatalf("pause changed delivery lifecycle: %#v", record)
	}
	snapshot, err := mb.Snapshot(ctx)
	if err != nil || len(snapshot.Continuations) != 1 {
		t.Fatalf("Snapshot() continuations=%#v err=%v", snapshot.Continuations, err)
	}
	for _, continuation := range snapshot.Continuations {
		if continuation.Continuation.PauseReason != MailboxPauseApprovalRequired || continuation.Continuation.PendingActionReference != "effect:approve" || continuation.Continuation.HandlerCheckpointReference != "handler-checkpoint:approval" {
			t.Fatalf("sealed handler continuation = %#v", continuation)
		}
	}
}

func TestContinuationWorkerResumeFreshOwnerInvokesResumerOnceAndAcknowledges(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "mailbox.json")
	store, err := NewFileStore(path, Policy{})
	if err != nil {
		t.Fatal(err)
	}
	mb, err := New(store)
	if err != nil {
		t.Fatal(err)
	}
	binding := continuationTestBinding("resume-once")
	if _, err := mb.AdmitHostPlan(ctx, binding); err != nil {
		t.Fatal(err)
	}
	pausing, err := NewContinuationWorker(mb, WorkerConfig{Enabled: true}, func(context.Context, Record) (HandlerResult, error) {
		return HandlerResult{Pause: &HandlerPause{
			Reason:              MailboxPauseInputRequired,
			CheckpointReference: "handler-checkpoint:input",
			OriginalDeadline:    time.Now().UTC().Add(time.Hour),
			ExpiresAt:           time.Now().UTC().Add(time.Hour),
		}}, nil
	}, func(context.Context, MailboxContinuation, Record) (HandlerResult, error) {
		return HandlerResult{}, errors.New("original process resumer must not run")
	}, func(context.Context, MailboxContinuation) error { return nil }, "continuation-worker")
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := pausing.RunOnce(ctx); err != nil || !processed {
		t.Fatalf("pause RunOnce() processed=%v err=%v", processed, err)
	}
	snapshot, err := mb.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var continuation MailboxContinuation
	for _, saved := range snapshot.Continuations {
		continuation = saved.Continuation
	}
	if continuation.Ref.CheckpointID == "" {
		t.Fatal("missing persisted continuation")
	}
	reopenedStore, err := NewFileStore(path, Policy{})
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := New(reopenedStore)
	if err != nil {
		t.Fatal(err)
	}
	resumeCalls := 0
	resuming, err := NewContinuationWorker(reopened, WorkerConfig{Enabled: true}, func(context.Context, Record) (HandlerResult, error) {
		return HandlerResult{}, errors.New("fresh handler must not run during resume")
	}, func(_ context.Context, restored MailboxContinuation, record Record) (HandlerResult, error) {
		resumeCalls++
		if restored.Ref != continuation.Ref || record.DeliveryID != continuation.DeliveryID || record.LeaseID != continuation.LeaseID {
			t.Fatalf("resumer source identity changed: continuation=%#v record=%#v", restored, record)
		}
		return HandlerResult{}, nil
	}, func(context.Context, MailboxContinuation) error { return nil }, "continuation-worker")
	if err != nil {
		t.Fatal(err)
	}
	req := MailboxResumeRequest{Ref: continuation.Ref, Binding: binding, IdempotencyKey: "resume-once", Generation: continuation.Generation, PauseReason: continuation.PauseReason, Deployment: types.DeploymentRequest{OwnerNamespace: "mailbox", WriterFence: "fence:mailbox"}}
	first, err := resuming.ResumeHandlerContinuation(ctx, req)
	if err != nil {
		t.Fatalf("ResumeHandlerContinuation(first): %v", err)
	}
	second, err := resuming.ResumeHandlerContinuation(ctx, req)
	if err != nil {
		t.Fatalf("ResumeHandlerContinuation(duplicate): %v", err)
	}
	if first != second || resumeCalls != 1 || first.TerminalOutcome != "acked" {
		t.Fatalf("resume result first=%#v second=%#v calls=%d", first, second, resumeCalls)
	}
	page, err := reopened.Query(ctx, QueryRequest{MessageID: binding.MessageID})
	if err != nil || len(page.Items) != 1 || page.Items[0].State != StateAcked || page.Items[0].DeliveryAttempt != 1 {
		t.Fatalf("resume lifecycle = %#v err=%v", page.Items, err)
	}
}

func TestContinuationWorkerPublishesHandlerSafePointCapabilities(t *testing.T) {
	store, err := NewFileStore(filepath.Join(t.TempDir(), "mailbox.json"), Policy{})
	if err != nil {
		t.Fatal(err)
	}
	mb, err := New(store)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewContinuationWorker(mb, WorkerConfig{Enabled: true}, func(context.Context, Record) (HandlerResult, error) {
		return HandlerResult{}, nil
	}, func(context.Context, MailboxContinuation, Record) (HandlerResult, error) {
		return HandlerResult{}, nil
	}, func(context.Context, MailboxContinuation) error { return nil }, "continuation-worker"); err != nil {
		t.Fatal(err)
	}
	caps, err := mb.ContinuationCapabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !caps.HandlerExecutionSafePoint {
		t.Fatalf("handler safe point capability = false: %#v", caps)
	}
	for _, reason := range []string{"input_required", "approval_required", "budget_exhausted"} {
		evidence, ok := caps.PauseReason(reason)
		if !ok || !evidence.Supported {
			t.Fatalf("pause capability %q = %#v, declared=%v", reason, evidence, ok)
		}
	}
}

func TestContinuationWorkerRejectsMalformedHandlerPauseWithoutLifecycleMutation(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name  string
		pause HandlerPause
		want  error
	}{
		{name: "missing-checkpoint", pause: HandlerPause{Reason: MailboxPauseInputRequired, ExpiresAt: time.Now().UTC().Add(time.Hour)}, want: ErrMailboxContinuationMalformed},
		{name: "oversized-checkpoint", pause: HandlerPause{Reason: MailboxPauseInputRequired, CheckpointReference: strings.Repeat("x", 1025), ExpiresAt: time.Now().UTC().Add(time.Hour)}, want: ErrMailboxContinuationMalformed},
		{name: "unknown-reason", pause: HandlerPause{Reason: MailboxPauseReason("unknown"), CheckpointReference: "handler-checkpoint:unknown", ExpiresAt: time.Now().UTC().Add(time.Hour)}, want: ErrMailboxContinuationMalformed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, err := NewFileStore(filepath.Join(t.TempDir(), "mailbox.json"), Policy{})
			if err != nil {
				t.Fatal(err)
			}
			mb, err := New(store)
			if err != nil {
				t.Fatal(err)
			}
			binding := continuationTestBinding("malformed-" + tc.name)
			if _, err := mb.AdmitHostPlan(ctx, binding); err != nil {
				t.Fatal(err)
			}
			worker, err := NewContinuationWorker(mb, WorkerConfig{Enabled: true}, func(context.Context, Record) (HandlerResult, error) {
				return HandlerResult{Pause: &tc.pause}, nil
			}, continuationNoopResumer, func(context.Context, MailboxContinuation) error { return nil }, "continuation-worker")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := worker.RunOnce(ctx); !errors.Is(err, tc.want) {
				t.Fatalf("malformed pause err=%v, want %v", err, tc.want)
			}
			page, err := mb.Query(ctx, QueryRequest{MessageID: binding.MessageID})
			if err != nil || len(page.Items) != 1 || page.Items[0].State != StateInFlight {
				t.Fatalf("malformed pause changed lifecycle items=%#v err=%v", page.Items, err)
			}
			snapshot, err := mb.Snapshot(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(snapshot.Continuations) != 0 {
				t.Fatalf("malformed pause persisted continuation: %#v", snapshot.Continuations)
			}
		})
	}
}

func TestContinuationWorkerLeaseLossDoesNotInvokeResumer(t *testing.T) {
	ctx := context.Background()
	store, err := NewFileStore(filepath.Join(t.TempDir(), "mailbox.json"), Policy{})
	if err != nil {
		t.Fatal(err)
	}
	mb, err := New(store)
	if err != nil {
		t.Fatal(err)
	}
	binding := continuationTestBinding("lease-loss")
	if _, err := mb.AdmitHostPlan(ctx, binding); err != nil {
		t.Fatal(err)
	}
	pausing, err := NewContinuationWorker(mb, WorkerConfig{Enabled: true, InflightTimeout: time.Hour, HeartbeatInterval: time.Minute}, func(context.Context, Record) (HandlerResult, error) {
		return HandlerResult{Pause: &HandlerPause{Reason: MailboxPauseInputRequired, CheckpointReference: "handler-checkpoint:lease", OriginalDeadline: time.Now().UTC().Add(time.Hour), ExpiresAt: time.Now().UTC().Add(time.Hour)}}, nil
	}, func(context.Context, MailboxContinuation, Record) (HandlerResult, error) {
		return HandlerResult{}, nil
	}, func(context.Context, MailboxContinuation) error { return nil }, "continuation-worker")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pausing.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	snapshot, err := mb.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var continuation MailboxContinuation
	for _, saved := range snapshot.Continuations {
		continuation = saved.Continuation
	}
	if _, err := mb.Requeue(ctx, binding.MessageID, "continuation-worker", LifecycleReasonLeaseExpired); err != nil {
		t.Fatal(err)
	}
	resumerCalls := 0
	resuming, err := NewContinuationWorker(mb, WorkerConfig{Enabled: true}, func(context.Context, Record) (HandlerResult, error) {
		return HandlerResult{}, nil
	}, func(context.Context, MailboxContinuation, Record) (HandlerResult, error) {
		resumerCalls++
		return HandlerResult{}, nil
	}, func(context.Context, MailboxContinuation) error { return nil }, "continuation-worker")
	if err != nil {
		t.Fatal(err)
	}
	_, err = resuming.ResumeHandlerContinuation(ctx, MailboxResumeRequest{Ref: continuation.Ref, Binding: binding, IdempotencyKey: "lease-loss", Generation: continuation.Generation, PauseReason: continuation.PauseReason, Deployment: types.DeploymentRequest{OwnerNamespace: "mailbox", WriterFence: "fence:mailbox"}})
	if !errors.Is(err, ErrMailboxContinuationLeaseLost) || resumerCalls != 0 {
		t.Fatalf("lease loss err=%v resumerCalls=%d", err, resumerCalls)
	}
}

func TestContinuationWorkerResumeRevalidationFailurePreservesInFlightSource(t *testing.T) {
	ctx := context.Background()
	mb, continuation, binding := pausedContinuationWorker(t, Policy{}, "revalidation-reject")
	resumerCalls := 0
	resuming, err := NewContinuationWorker(mb, WorkerConfig{Enabled: true}, continuationNoopHandler,
		func(context.Context, MailboxContinuation, Record) (HandlerResult, error) {
			resumerCalls++
			return HandlerResult{}, nil
		}, func(context.Context, MailboxContinuation) error { return errors.New("authorization revoked") }, "continuation-worker")
	if err != nil {
		t.Fatal(err)
	}
	_, err = resuming.ResumeHandlerContinuation(ctx, continuationResumeRequest(continuation, binding, "revalidation-reject", types.DeploymentRequest{OwnerNamespace: "mailbox", WriterFence: "fence:mailbox"}))
	if err == nil || resumerCalls != 0 {
		t.Fatalf("revalidation err=%v resumerCalls=%d", err, resumerCalls)
	}
	page, err := mb.Query(ctx, QueryRequest{MessageID: binding.MessageID})
	if err != nil || len(page.Items) != 1 || page.Items[0].State != StateInFlight {
		t.Fatalf("revalidation changed source lifecycle items=%#v err=%v", page.Items, err)
	}
	snapshot, err := mb.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := snapshot.Continuations[continuation.Ref.CheckpointID].Claim; got != nil {
		t.Fatalf("revalidation created a resume claim: %#v", got)
	}
}

func TestContinuationWorkerResumeRejectsForeignDeploymentBeforeResumer(t *testing.T) {
	ctx := context.Background()
	mb, continuation, binding := pausedContinuationWorker(t, Policy{}, "foreign-deployment")
	resumerCalls := 0
	resuming, err := NewContinuationWorker(mb, WorkerConfig{Enabled: true}, continuationNoopHandler,
		func(context.Context, MailboxContinuation, Record) (HandlerResult, error) {
			resumerCalls++
			return HandlerResult{}, nil
		}, func(context.Context, MailboxContinuation) error { return nil }, "continuation-worker")
	if err != nil {
		t.Fatal(err)
	}
	_, err = resuming.ResumeHandlerContinuation(ctx, continuationResumeRequest(continuation, binding, "foreign-deployment", types.DeploymentRequest{OwnerNamespace: "foreign", WriterFence: "fence:foreign"}))
	if !errors.Is(err, ErrMailboxContinuationUnavailable) || resumerCalls != 0 {
		t.Fatalf("foreign deployment err=%v resumerCalls=%d", err, resumerCalls)
	}
	page, err := mb.Query(ctx, QueryRequest{MessageID: binding.MessageID})
	if err != nil || len(page.Items) != 1 || page.Items[0].State != StateInFlight {
		t.Fatalf("foreign deployment changed source lifecycle items=%#v err=%v", page.Items, err)
	}
}

func TestContinuationWorkerResumeFailureUsesOwnerRetryAndDLQ(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name       string
		policy     Policy
		errorMode  string
		wantState  MessageState
		wantResult string
	}{
		{name: "requeue", policy: Policy{MaxAttempts: 2, DLQEnabled: true}, errorMode: WorkerHandlerErrorPolicyRequeue, wantState: StateQueued, wantResult: "retry"},
		{name: "nack-to-dlq", policy: Policy{MaxAttempts: 1, DLQEnabled: true}, errorMode: WorkerHandlerErrorPolicyNack, wantState: StateDeadLetter, wantResult: "dlq"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mb, continuation, binding := pausedContinuationWorker(t, tc.policy, "resume-failure-"+tc.name)
			resuming, err := NewContinuationWorker(mb, WorkerConfig{Enabled: true, HandlerErrorPolicy: tc.errorMode}, continuationNoopHandler,
				func(context.Context, MailboxContinuation, Record) (HandlerResult, error) {
					return HandlerResult{}, errors.New("resumer failure")
				}, func(context.Context, MailboxContinuation) error { return nil }, "continuation-worker")
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := resuming.ResumeHandlerContinuation(ctx, continuationResumeRequest(continuation, binding, "resume-failure-"+tc.name, types.DeploymentRequest{OwnerNamespace: "mailbox", WriterFence: "fence:mailbox"}))
			if err != nil {
				t.Fatal(err)
			}
			if outcome.LifecycleState != tc.wantState || outcome.LifecycleResult != tc.wantResult {
				t.Fatalf("resume failure outcome=%#v, want state=%q result=%q", outcome, tc.wantState, tc.wantResult)
			}
			duplicate, err := resuming.ResumeHandlerContinuation(ctx, continuationResumeRequest(continuation, binding, "resume-failure-"+tc.name, types.DeploymentRequest{OwnerNamespace: "mailbox", WriterFence: "fence:mailbox"}))
			if err != nil || duplicate != outcome {
				t.Fatalf("duplicate failure outcome=%#v err=%v want=%#v", duplicate, err, outcome)
			}
		})
	}
}

func pausedContinuationWorker(t *testing.T, policy Policy, suffix string) (*Mailbox, MailboxContinuation, HostPlanBinding) {
	t.Helper()
	ctx := context.Background()
	store, err := NewFileStore(filepath.Join(t.TempDir(), "mailbox.json"), policy)
	if err != nil {
		t.Fatal(err)
	}
	mb, err := New(store)
	if err != nil {
		t.Fatal(err)
	}
	binding := continuationTestBinding(suffix)
	if _, err := mb.AdmitHostPlan(ctx, binding); err != nil {
		t.Fatal(err)
	}
	worker, err := NewContinuationWorker(mb, WorkerConfig{Enabled: true, InflightTimeout: time.Hour, HeartbeatInterval: time.Minute}, func(context.Context, Record) (HandlerResult, error) {
		return HandlerResult{Pause: &HandlerPause{Reason: MailboxPauseInputRequired, CheckpointReference: "handler-checkpoint:" + suffix, OriginalDeadline: time.Now().UTC().Add(time.Hour), ExpiresAt: time.Now().UTC().Add(time.Hour)}}, nil
	}, continuationNoopResumer, func(context.Context, MailboxContinuation) error { return nil }, "continuation-worker")
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := worker.RunOnce(ctx); err != nil || !processed {
		t.Fatalf("pause RunOnce() processed=%v err=%v", processed, err)
	}
	snapshot, err := mb.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, saved := range snapshot.Continuations {
		return mb, saved.Continuation, binding
	}
	t.Fatal("missing paused continuation")
	return nil, MailboxContinuation{}, HostPlanBinding{}
}

func continuationNoopHandler(context.Context, Record) (HandlerResult, error) {
	return HandlerResult{}, nil
}

func continuationNoopResumer(context.Context, MailboxContinuation, Record) (HandlerResult, error) {
	return HandlerResult{}, nil
}

func continuationResumeRequest(continuation MailboxContinuation, binding HostPlanBinding, idempotency string, deployment types.DeploymentRequest) MailboxResumeRequest {
	return MailboxResumeRequest{Ref: continuation.Ref, Binding: binding, IdempotencyKey: idempotency, Generation: continuation.Generation, PauseReason: continuation.PauseReason, Deployment: deployment}
}

func continuationTestBinding(suffix string) HostPlanBinding {
	return HostPlanBinding{ContractVersion: HostPlanBindingContractVersion, MessageID: "continuation-" + suffix, IdempotencyKey: "logical-" + suffix, RunID: "run-" + suffix, ParentRunID: "parent-" + suffix, CausationID: "cause-" + suffix, AuthorityDigest: "authority", PlanDigest: "sha256:plan", WriteScope: "scope", EffectNamespace: "effects", Occurrence: "occurrence", HostCorrelation: "host", ResourceLimit: 1}
}
