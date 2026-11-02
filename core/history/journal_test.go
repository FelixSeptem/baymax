package history

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/sourcecoordination"
	"github.com/FelixSeptem/baymax/core/types"
)

func TestJournalReplaysWithSourceLocalCursorAndRetention(t *testing.T) {
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.history.v1", ID: "run-1"}
	j, err := New(source, WithRetention(2))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := j.Append(context.Background(), types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "runner.step", Event: types.EventEnvelope{Payload: map[string]any{"step": i}}}); err != nil {
			t.Fatal(err)
		}
	}
	caps, err := j.Capabilities(context.Background(), source)
	if err != nil || !caps.Supported || !caps.SequenceDefined || caps.RetentionFloor == nil {
		t.Fatalf("capabilities: %+v %v", caps, err)
	}
	page, err := j.ReadAfter(context.Background(), source, caps.RetentionFloor, 1)
	if err != nil || len(page.Events) != 1 || page.Events[0].History.Sequence != 2 {
		t.Fatalf("bounded replay: %+v %v", page, err)
	}
	if _, err := j.ReadAfter(context.Background(), source, &types.OpaqueCursor{Value: "foreign"}, 1); !errors.Is(err, types.ErrCursorMalformed) {
		t.Fatalf("malformed cursor: %v", err)
	}
	if _, err := j.ReadAfter(context.Background(), types.SourceRef{Kind: types.HistorySourceRunner, Namespace: source.Namespace, ID: "other"}, page.NextCursor, 1); !errors.Is(err, types.ErrCursorForeign) {
		t.Fatalf("foreign source: %v", err)
	}
}

func TestCoordinatedJournalRetainsOneSourceOrderAndReadDoesNotAcquireLease(t *testing.T) {
	ctx := context.Background()
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.history.v1", ID: "coordinated"}
	coordDir := filepath.Join(t.TempDir(), "coord")
	coordA, err := sourcecoordination.OpenFile(coordDir, sourcecoordination.Options{OwnerNamespace: source.ID})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = coordA.Close() }()
	coordB, err := sourcecoordination.OpenFile(coordDir, sourcecoordination.Options{OwnerNamespace: source.ID})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = coordB.Close() }()
	a, err := New(source, WithCoordinator(coordA, "writer-a", time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.Append(ctx, types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "runner.one"}); err != nil {
		t.Fatal(err)
	}
	b, err := New(source, WithCoordinator(coordB, "writer-b", time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = b.Append(ctx, types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "runner.two"}); err != nil {
		t.Fatalf("new owner append: %v", err)
	}
	if _, err = a.Append(ctx, types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "runner.stale"}); err == nil {
		t.Fatal("stale writer appended")
	}
	page, err := b.ReadAfter(ctx, source, nil, 10)
	if err != nil || len(page.Events) != 2 || page.Events[0].History.Sequence != 1 || page.Events[1].History.Sequence != 2 {
		t.Fatalf("coordinated page: %+v %v", page, err)
	}
	page, err = a.ReadAfter(ctx, source, nil, 10)
	if err != nil || len(page.Events) != 2 || page.Events[0].History.Sequence != 1 || page.Events[1].History.Sequence != 2 {
		t.Fatalf("stale reader did not retain committed facts: %+v %v", page, err)
	}
}

func TestCoordinatedJournalCapabilitiesFailClosedWhenCoordinatorIsUnavailable(t *testing.T) {
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.history.v1", ID: "coordinator-outage"}
	coordinator, err := sourcecoordination.OpenFile(filepath.Join(t.TempDir(), "coord"), sourcecoordination.Options{OwnerNamespace: source.ID})
	if err != nil {
		t.Fatal(err)
	}
	journal, err := New(source, WithCoordinator(coordinator, "writer", time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err = coordinator.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = journal.Capabilities(context.Background(), source); !errors.Is(err, types.ErrSourceUnavailable) {
		t.Fatalf("unavailable coordinator advertised history capability: %v", err)
	}
}

func TestJournalInitialReadIncludesRetainedFloorEvent(t *testing.T) {
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.history.v1", ID: "initial-read"}
	j, err := New(source, WithRetention(2))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := j.Append(context.Background(), types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "runner.step"}); err != nil {
			t.Fatal(err)
		}
	}
	page, err := j.ReadAfter(context.Background(), source, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Events) != 2 || page.Events[0].History.Sequence != 2 || page.Events[1].History.Sequence != 3 {
		t.Fatalf("initial retained replay = %+v", page.Events)
	}
}

func TestJournalRejectsInvalidSourceFactsBeforeAppend(t *testing.T) {
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.history.v1", ID: "source-facts"}
	j, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	bad := types.NativeExecutionEvent{
		Version:    types.NativeExecutionEventVersionV1,
		NativeType: "runner.step",
		History:    &types.NativeHistoryIdentity{Source: source, Sequence: 99, EventID: "callback-event-id"},
	}
	if _, err := j.Append(context.Background(), bad); err != nil {
		t.Fatalf("append should own identity, got %v", err)
	}
	page, err := j.ReadAfter(context.Background(), source, nil, 1)
	if err != nil || len(page.Events) != 1 || page.Events[0].History.Sequence != 1 {
		t.Fatalf("source-owned identity not normalized: %+v %v", page, err)
	}
}

func TestJournalRedactsEventPayloadBeforeRetention(t *testing.T) {
	source := types.SourceRef{Kind: types.HistorySourceWorkflow, Namespace: "workflow.history.v1", ID: "redaction"}
	j, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.Append(context.Background(), types.NativeExecutionEvent{
		Version:    types.NativeExecutionEventVersionV1,
		NativeType: "workflow.completed",
		Event: types.EventEnvelope{Payload: map[string]any{
			"secret":          "must-not-persist",
			"business_commit": true,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	page, err := j.ReadAfter(context.Background(), source, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Events) != 1 || page.Events[0].Event.Payload != nil {
		t.Fatalf("event payload retained in source history: %+v", page.Events)
	}
}

func TestFileJournalReopensWithoutExecutingSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	source := types.SourceRef{Kind: types.HistorySourceWorkflow, Namespace: "workflow.history.v1", ID: "workflow-1"}
	j, err := OpenFile(path, source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.Append(context.Background(), types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "workflow.step", Event: types.EventEnvelope{Payload: map[string]any{"state": "done"}}}); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenFile(path, source)
	if err != nil {
		t.Fatal(err)
	}
	page, err := reopened.ReadAfter(context.Background(), source, nil, 10)
	if err != nil || len(page.Events) != 1 || page.Events[0].History.Source != source {
		t.Fatalf("reopened history: %+v %v", page, err)
	}
	before, _ := reopened.Snapshot(context.Background())
	if _, err := reopened.ReadAfter(context.Background(), source, page.NextCursor, 10); err != nil {
		t.Fatal(err)
	}
	after, _ := reopened.Snapshot(context.Background())
	if before.Sequence != after.Sequence || len(before.Events) != len(after.Events) {
		t.Fatal("read mutated journal")
	}
}

func TestFileJournalAppendFailureDoesNotExposeUncommittedEvent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.history.v1", ID: "run-fail"}
	j, err := OpenFile(path, source)
	if err != nil {
		t.Fatal(err)
	}
	j.path = filepath.Join(t.TempDir(), "missing", "history.json")
	if _, err := j.Append(context.Background(), types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "runner.step"}); !errors.Is(err, types.ErrSourceUnavailable) {
		t.Fatalf("append error = %v", err)
	}
	page, err := j.ReadAfter(context.Background(), source, nil, 10)
	if err != nil || len(page.Events) != 0 || !page.Complete {
		t.Fatalf("uncommitted event exposed: %+v %v", page, err)
	}
}

func TestAppendHealthRetainsSourceUnavailable(t *testing.T) {
	var health AppendHealth
	if err := health.Err(); err != nil {
		t.Fatalf("initial history health = %v", err)
	}
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.history.v1", ID: "run-health"}
	health.Record(source, types.ErrSourceUnavailable)
	if !errors.Is(health.Err(), types.ErrSourceUnavailable) || !strings.Contains(health.Err().Error(), source.ID) {
		t.Fatalf("history health error = %v", health.Err())
	}
}

func TestFileJournalFailureAfterRetentionRollsBackWholeState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.history.v1", ID: "run-retention-fail"}
	j, err := OpenFile(path, source, WithRetention(2))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := j.Append(context.Background(), types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "runner.step"}); err != nil {
			t.Fatal(err)
		}
	}
	before, err := j.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	j.path = filepath.Join(t.TempDir(), "missing", "history.json")
	if _, err := j.Append(context.Background(), types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "runner.step"}); !errors.Is(err, types.ErrSourceUnavailable) {
		t.Fatalf("append error = %v", err)
	}
	after, err := j.Snapshot(context.Background())
	if err != nil || before.Sequence != after.Sequence || before.Floor != after.Floor || len(before.Events) != len(after.Events) {
		t.Fatalf("retention state changed after failed append: before=%+v after=%+v err=%v", before, after, err)
	}
}
