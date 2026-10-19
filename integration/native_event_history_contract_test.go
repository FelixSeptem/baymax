package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/host/eventadapter"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/mailbox"
	"github.com/FelixSeptem/baymax/orchestration/planbridge"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	"github.com/FelixSeptem/baymax/orchestration/teams"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
)

func TestNativeHistoryMailboxRestartReplaysTerminalWithoutLifecycleMutation(t *testing.T) {
	ctx := context.Background()
	if path := os.Getenv("BAYMAX_INTEGRATION_HISTORY_PATH"); path != "" {
		var source types.SourceRef
		if err := json.Unmarshal([]byte(os.Getenv("BAYMAX_INTEGRATION_HISTORY_SOURCE")), &source); err != nil {
			t.Fatal(err)
		}
		assertNativeHistoryRestart(t, path, source, &types.OpaqueCursor{Value: os.Getenv("BAYMAX_INTEGRATION_HISTORY_CURSOR")})
		return
	}
	path := filepath.Join(t.TempDir(), "mailbox.json")
	store, err := mailbox.NewFileStore(path, mailbox.Policy{})
	if err != nil {
		t.Fatal(err)
	}
	owner, err := mailbox.New(store)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := planbridge.NewCapabilityManifestReader(planbridge.DefaultCapabilityManifest())
	if err != nil {
		t.Fatal(err)
	}
	b := bridgeBinding()
	b.Mailbox = &planbridge.MailboxBinding{Owner: owner, Manifest: manifest, Binding: mailbox.HostPlanBinding{
		ContractVersion: mailbox.HostPlanBindingContractVersion,
		MessageID:       "history-restart-message",
		IdempotencyKey:  "history-restart-logical",
		RunID:           "history-run",
		ParentRunID:     "history-parent",
		CausationID:     "history-causation",
		AuthorityDigest: "history-authority",
		WriteScope:      "history-write-scope",
		EffectNamespace: "history-effects",
		Occurrence:      "history-occurrence",
		HostCorrelation: "history-host",
		ResourceLimit:   1,
	}}
	d := bridgeDescriptor(planbridge.ShapeDirect)
	d.Delivery = planbridge.DeliveryMailbox
	d, err = planbridge.Describe(d, b, planbridge.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	p, err := planbridge.Compile(d, b, planbridge.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.Run(ctx, planbridge.Execution{}); err != nil {
		t.Fatal(err)
	}
	source, err := owner.HistorySource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// The consumer confirmed only the admission fact before it crashed.
	admission, err := owner.ReadAfter(ctx, source, nil, 1)
	if err != nil || len(admission.Events) != 1 || admission.Events[0].NativeType != "mailbox.admit" || admission.NextCursor == nil {
		t.Fatalf("admission page: %+v err=%v", admission, err)
	}
	claimed, ok, err := owner.Consume(ctx, "history-consumer")
	if err != nil || !ok {
		t.Fatalf("consume: ok=%v err=%v", ok, err)
	}
	if _, err = owner.Ack(ctx, claimed.Envelope.MessageID, "history-consumer"); err != nil {
		t.Fatal(err)
	}
	assertNativeHistoryRestart(t, path, source, admission.NextCursor)
	sourceJSON, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	// A separate process has no references to the original owner or its memory.
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestNativeHistoryMailboxRestartReplaysTerminalWithoutLifecycleMutation$", "-test.count=1")
	command.Env = append(os.Environ(), "BAYMAX_INTEGRATION_HISTORY_PATH="+path, "BAYMAX_INTEGRATION_HISTORY_SOURCE="+string(sourceJSON), "BAYMAX_INTEGRATION_HISTORY_CURSOR="+admission.NextCursor.Value)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("new process replay: %v\n%s", err, output)
	}
}

func assertNativeHistoryRestart(t *testing.T, path string, source types.SourceRef, cursor *types.OpaqueCursor) {
	t.Helper()
	ctx := context.Background()
	beforeDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	restartedStore, err := mailbox.NewFileStore(path, mailbox.Policy{})
	if err != nil {
		t.Fatal(err)
	}
	restarted, err := mailbox.New(restartedStore)
	if err != nil {
		t.Fatal(err)
	}
	if restartedSource, err := restarted.HistorySource(ctx); err != nil || restartedSource != source {
		t.Fatalf("source identity changed after restart: %v %v", restartedSource, err)
	}
	before, err := restarted.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	page, err := restarted.ReadAfter(ctx, source, cursor, 10)
	if err != nil || len(page.Events) != 2 || !page.Complete {
		t.Fatalf("terminal replay: %+v err=%v", page, err)
	}
	terminal := page.Events[len(page.Events)-1]
	if terminal.NativeType != "mailbox.ack" || terminal.History == nil || terminal.History.TerminalOutcome != "acked" {
		t.Fatalf("terminal evidence: %+v", terminal)
	}
	for _, event := range page.Events {
		if event.History == nil || event.History.Source != source || event.History.Sequence == 0 || event.Event.Sequence != 0 {
			t.Fatalf("source-local identity/order drift: %+v", event)
		}
		if event.History.Source.Kind != types.HistorySourceMailbox || event.Event.RunID != "history-run" {
			t.Fatalf("source correlation drift: %+v", event)
		}
		for key, want := range map[string]string{
			"parent_run_id":    "history-parent",
			"causation_id":     "history-causation",
			"authority_digest": "history-authority",
			"write_scope":      "history-write-scope",
			"effect_namespace": "history-effects",
			"occurrence":       "history-occurrence",
			"host_correlation": "history-host",
		} {
			if event.Event.Payload[key] != want {
				t.Fatalf("%s lost in replay: %#v", key, event.Event.Payload)
			}
		}
	}
	after, err := restarted.Snapshot(ctx)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("read-only replay changed source state: before=%+v after=%+v err=%v", before, after, err)
	}
	again, err := restarted.ReadAfter(ctx, source, cursor, 10)
	if err != nil || !reflect.DeepEqual(page, again) {
		t.Fatalf("same cursor did not reproduce source facts: %+v %v", again, err)
	}
	afterDisk, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(beforeDisk, afterDisk) {
		t.Fatalf("replay wrote source store: %v", err)
	}
}

func TestNativeHistoryUnavailableDoesNotChangeMailboxAdmissionOrParity(t *testing.T) {
	ctx := context.Background()
	store, err := mailbox.NewFileStore(filepath.Join(t.TempDir(), "batched.json"), mailbox.Policy{}, mailbox.WithPersistBatchSize(10))
	if err != nil {
		t.Fatal(err)
	}
	owner, err := mailbox.New(store)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := planbridge.NewCapabilityManifestReader(planbridge.DefaultCapabilityManifest())
	if err != nil {
		t.Fatal(err)
	}
	b := bridgeBinding()
	b.Mailbox = &planbridge.MailboxBinding{Owner: owner, Manifest: manifest, Binding: mailbox.HostPlanBinding{
		ContractVersion: mailbox.HostPlanBindingContractVersion,
		MessageID:       "history-unavailable-message",
		IdempotencyKey:  "history-unavailable-logical",
		RunID:           "history-unavailable-run",
		ParentRunID:     "history-unavailable-parent",
		CausationID:     "history-unavailable-causation",
		AuthorityDigest: "history-unavailable-authority",
		WriteScope:      "history-unavailable-scope",
		EffectNamespace: "history-unavailable-effects",
		Occurrence:      "history-unavailable-occurrence",
		HostCorrelation: "history-unavailable-host",
		ResourceLimit:   1,
	}}
	d := bridgeDescriptor(planbridge.ShapeDirect)
	d.Delivery = planbridge.DeliveryMailbox
	d, err = planbridge.Describe(d, b, planbridge.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	p, err := planbridge.Compile(d, b, planbridge.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	projections := 0
	execution := planbridge.Execution{Sink: eventadapter.SinkFunc(func(context.Context, types.NativeExecutionEvent) error { projections++; return nil })}
	run, err := p.Run(ctx, execution)
	if err != nil || run.Mailbox == nil {
		t.Fatalf("admission must remain available while history is unavailable: %+v err=%v", run, err)
	}
	if _, err = owner.HistorySource(ctx); !errors.Is(err, types.ErrSourceUnavailable) {
		t.Fatalf("expected typed history unavailability, got %v", err)
	}
	stream, err := p.Stream(ctx, execution)
	if err != nil || stream.Mailbox == nil || !stream.Mailbox.Duplicate {
		t.Fatalf("stream admission parity: %+v err=%v", stream, err)
	}
	for field, pair := range map[string][2]string{
		"message":          {run.Mailbox.MessageID, stream.Mailbox.MessageID},
		"run":              {run.Mailbox.RunID, stream.Mailbox.RunID},
		"parent":           {run.Mailbox.ParentRunID, stream.Mailbox.ParentRunID},
		"causation":        {run.Mailbox.CausationID, stream.Mailbox.CausationID},
		"authority":        {run.Mailbox.AuthorityDigest, stream.Mailbox.AuthorityDigest},
		"occurrence":       {run.Mailbox.Occurrence, stream.Mailbox.Occurrence},
		"effect namespace": {run.Mailbox.EffectNamespace, stream.Mailbox.EffectNamespace},
		"host correlation": {run.Mailbox.HostCorrelation, stream.Mailbox.HostCorrelation},
	} {
		if pair[0] != pair[1] {
			t.Fatalf("Run/Stream %s drift: %#v", field, pair)
		}
	}
	if _, ok, err := owner.Consume(ctx, "owner"); err != nil || !ok {
		t.Fatalf("consume: ok=%v err=%v", ok, err)
	}
	if _, err := owner.Ack(ctx, b.Mailbox.Binding.MessageID, "owner"); err != nil {
		t.Fatal(err)
	}
	terminalRun, err := p.Run(ctx, execution)
	if err != nil {
		t.Fatal(err)
	}
	terminalStream, err := p.Stream(ctx, execution)
	if err != nil || terminalRun.Mailbox == nil || terminalStream.Mailbox == nil || !reflect.DeepEqual(terminalRun.Mailbox, terminalStream.Mailbox) || terminalRun.Mailbox.TerminalOutcome != "acked" {
		t.Fatalf("unavailable-history terminal parity: run=%+v stream=%+v err=%v", terminalRun, terminalStream, err)
	}
	if _, err = owner.HistorySource(ctx); !errors.Is(err, types.ErrSourceUnavailable) {
		t.Fatalf("history reads must not flush pending transitions: %v", err)
	}
	if err = store.Flush(); err != nil {
		t.Fatal(err)
	}
	source, err := owner.HistorySource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	before, err := owner.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	page, err := owner.ReadAfter(ctx, source, nil, 10)
	if err != nil || len(page.Events) != 3 || page.Events[2].History.TerminalOutcome != "acked" {
		t.Fatalf("committed history: %+v err=%v", page, err)
	}
	after, err := owner.Snapshot(ctx)
	if err != nil || !reflect.DeepEqual(before, after) || len(after.Records) != 1 || projections != 0 {
		t.Fatalf("history changed logical admission or invoked host projection: records=%d projections=%d err=%v", len(after.Records), projections, err)
	}
}

func TestNativeHistoryUnsupportedSourcesKeepLiveExecutionAndCorrelation(t *testing.T) {
	for _, kind := range []types.HistorySourceKind{types.HistorySourceRunner, types.HistorySourceChild, types.HistorySourceTeam, types.HistorySourceWorkflow} {
		for _, stream := range []bool{false, true} {
			t.Run(string(kind)+map[bool]string{false: "/run", true: "/stream"}[stream], func(t *testing.T) {
				ctx := context.Background()
				reader := types.UnsupportedNativeEventHistoryReader{}
				source := types.SourceRef{Kind: kind, Namespace: "source-audit.v1", ID: "audit-run"}
				calls := 0
				b := bridgeBinding()
				shape := planbridge.ShapeDirect
				switch kind {
				case types.HistorySourceRunner:
					b.Direct = &planbridge.DirectBinding{Runner: nativeEngine(t, func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
						calls++
						return types.ModelResponse{FinalAnswer: "done"}, nil
					}, nil), Request: types.RunRequest{RunID: "audit-run", Input: "offline"}}
				case types.HistorySourceChild:
					shape = planbridge.ShapeSubagent
					b.Child = &planbridge.ChildBinding{TerminalOnly: true, Request: composer.ChildDispatchRequest{Target: composer.ChildTargetLocal, Task: scheduler.Task{TaskID: "audit-child", RunID: "audit-run", ParentRunID: "audit-parent", StepID: "audit-step", MaxAttempts: 1}, ParentRemainingBudget: time.Second, ChildTimeout: time.Second, LocalRunner: composer.LocalChildRunnerFunc(func(context.Context, scheduler.Task) (map[string]any, error) {
						calls++
						return map[string]any{"result": "done"}, nil
					})}}
				case types.HistorySourceTeam:
					shape = planbridge.ShapeTeam
					b.Team = &planbridge.TeamBinding{TerminalOnly: true, Plan: teams.Plan{RunID: "audit-run", TeamID: "audit-team", Strategy: teams.StrategyParallel, Tasks: []teams.Task{{TaskID: "audit-member", AgentID: "offline", Runner: teams.TaskRunnerFunc(func(context.Context, teams.Task) (teams.TaskResult, error) {
						calls++
						return teams.TaskResult{Output: "done"}, nil
					})}}}}
				case types.HistorySourceWorkflow:
					shape = planbridge.ShapeWorkflow
					b.Workflow = &planbridge.WorkflowBinding{TerminalOnly: true, Request: workflow.RunRequest{RunID: "audit-run", DSL: workflow.Definition{WorkflowID: "audit-workflow", Steps: []workflow.Step{{StepID: "audit-step"}}}}, Adapter: workflow.DispatchAdapter{Runner: func(context.Context, string, workflow.Step, int) (workflow.StepOutput, error) {
						calls++
						return workflow.StepOutput{Payload: map[string]any{"result": "done"}}, nil
					}}}
				}
				p := bridgeCompile(t, shape, b)
				if _, err := reader.ReadAfter(ctx, source, nil, 1); !errors.Is(err, types.ErrHistoryUnsupported) || calls != 0 {
					t.Fatalf("unsupported read activated %s: calls=%d err=%v", kind, calls, err)
				}
				var mu sync.Mutex
				var events []types.NativeExecutionEvent
				_, err := bridgeExecute(t, p, stream, planbridge.Execution{Sink: eventadapter.SinkFunc(func(_ context.Context, event types.NativeExecutionEvent) error {
					mu.Lock()
					defer mu.Unlock()
					events = append(events, event)
					return nil
				})})
				if err != nil || calls != 1 || len(events) == 0 {
					t.Fatalf("live source changed: calls=%d events=%d err=%v", calls, len(events), err)
				}
				parentSeen := kind != types.HistorySourceChild
				for _, event := range events {
					if event.Event.RunID != "audit-run" || event.History != nil || event.Event.Sequence != 0 || event.Capabilities.Cursor || event.Capabilities.DurableHistory {
						t.Fatalf("live correlation/ownership drift: %+v", event)
					}
					parentSeen = parentSeen || event.Event.Payload["parent_run_id"] == "audit-parent"
				}
				caps, err := reader.Capabilities(ctx, source)
				if err != nil || caps.Supported || caps.SequenceDefined || caps.SequenceScope != "none" || caps.UnsupportedReason == "" || !parentSeen {
					t.Fatalf("source audit/correlation drift: %+v parent=%v err=%v", caps, parentSeen, err)
				}
				if _, err := reader.ReadAfter(ctx, source, nil, 1); !errors.Is(err, types.ErrHistoryUnsupported) || calls != 1 {
					t.Fatalf("history read retriggered source: calls=%d err=%v", calls, err)
				}
			})
		}
	}
}

func TestNativeHistorySourceAuditAndAdapterIdentityBoundary(t *testing.T) {
	ctx := context.Background()
	reader := types.UnsupportedNativeEventHistoryReader{}
	for _, kind := range []types.HistorySourceKind{types.HistorySourceRunner, types.HistorySourceChild, types.HistorySourceTeam, types.HistorySourceWorkflow} {
		source := types.SourceRef{Kind: kind, Namespace: "callback-only.v1", ID: string(kind) + "-source"}
		caps, err := reader.Capabilities(ctx, source)
		if err != nil || caps.Supported || caps.Durable || caps.SequenceDefined || caps.CursorScope != "none" {
			t.Fatalf("source %s capability audit: %+v err=%v", kind, caps, err)
		}
		if _, err = reader.ReadAfter(ctx, source, nil, 1); !errors.Is(err, types.ErrHistoryUnsupported) {
			t.Fatalf("source %s history fallback: %v", kind, err)
		}
	}

	var projected types.NativeExecutionEvent
	observer, err := eventadapter.New(eventadapter.Options{
		Projection: types.NativeExecutionOptions{Source: types.ProtocolSourceRunner, ExpectedRunID: "callback-run"},
		Sink: eventadapter.SinkFunc(func(_ context.Context, event types.NativeExecutionEvent) error {
			projected = event
			return nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer observer.Close()
	observer.OnEvent(ctx, types.Event{Type: "run.finished", RunID: "callback-run", TraceID: "callback-trace", Time: time.Now(), Payload: map[string]any{"state": "completed"}})
	if projected.Event.EventID == "" || projected.History != nil || projected.Capabilities.Identity != "mapping_reference" || projected.Capabilities.Cursor || projected.Capabilities.DurableHistory {
		t.Fatalf("callback was presented as retained history: %+v", projected)
	}
	// A callback EventID is an observation identity only; the source audit still
	// rejects it without creating a cursor or replaying the callback.
	if _, err = reader.ReadAfter(ctx, types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "callback-only.v1", ID: "runner-source"}, &types.OpaqueCursor{Value: projected.Event.EventID}, 1); !errors.Is(err, types.ErrHistoryUnsupported) {
		t.Fatalf("callback EventID accepted as history cursor: %v", err)
	}
}
