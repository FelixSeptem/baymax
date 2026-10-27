package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/history"
	"github.com/FelixSeptem/baymax/core/types"
)

func TestNewNativeEventHistoryRequiresRunnerSource(t *testing.T) {
	if _, err := NewNativeEventHistory(types.SourceRef{Kind: types.HistorySourceWorkflow, Namespace: "workflow", ID: "wrong"}); err == nil {
		t.Fatal("accepted non-runner source")
	}
	journal, err := NewNativeEventHistory(types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.history.v1", ID: "run"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := any(journal).(types.NativeEventHistoryReader); !ok {
		t.Fatal("runner history does not expose source reader")
	}
}

func TestRunnerExposesNativeHistoryAppendFailure(t *testing.T) {
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.history.v1", ID: "run-health"}
	root := t.TempDir()
	journal, err := OpenNativeEventHistory(filepath.Join(root, "history.json"), source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	e := New(&fakeModel{}, WithNativeEventHistory(journal))
	e.appendNativeHistory(context.Background(), types.Event{Version: types.EventSchemaVersionV1, Type: "run.started", RunID: source.ID, Time: time.Now()})
	if !errors.Is(e.NativeEventHistoryError(), types.ErrSourceUnavailable) {
		t.Fatalf("history append error = %v", e.NativeEventHistoryError())
	}
}

func TestRunnerNativeHistoryAppendIsExplicitAndSourceOwned(t *testing.T) {
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.history.v1", ID: "run-history"}
	journal, err := NewNativeEventHistory(source)
	if err != nil {
		t.Fatal(err)
	}
	e := &Engine{nativeHistory: journal}
	e.appendNativeHistory(context.Background(), types.Event{Version: types.EventSchemaVersionV1, Type: "run.started", RunID: source.ID, Time: time.Now(), Payload: map[string]any{"secret": "redact"}})
	e.appendNativeHistory(context.Background(), types.Event{Version: types.EventSchemaVersionV1, Type: "run.started", RunID: "other"})
	page, err := journal.ReadAfter(context.Background(), source, nil, 10)
	if err != nil || len(page.Events) != 1 {
		t.Fatalf("source-owned append = %+v err=%v", page, err)
	}
	if page.Events[0].Event.Payload != nil || page.Events[0].History.Source != source {
		t.Fatalf("history retained non-source fact: %+v", page.Events[0])
	}
}

func TestRunnerNativeHistoryBindsSourceProjectionLedger(t *testing.T) {
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.commit.v1", ID: "run-owner"}
	root := t.TempDir()
	ledger, err := history.OpenProjectionLedger(filepath.Join(root, "ledger.json"), source)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := OpenNativeEventHistory(filepath.Join(root, "history.json"), source, history.WithProjectionLedger(ledger))
	if err != nil {
		t.Fatal(err)
	}
	e := New(&fakeModel{}, WithNativeEventHistory(journal))
	e.appendNativeHistory(context.Background(), types.Event{Version: types.EventSchemaVersionV1, Type: "run.finished", RunID: source.ID, Time: time.Now()})
	if err := e.NativeEventHistoryError(); err != nil {
		t.Fatalf("owner history error = %v", err)
	}
	page, err := ledger.ReadCommittedAfter(context.Background(), nil, 1)
	if err != nil || len(page.Records) != 1 {
		t.Fatalf("runner projection records = %#v err=%v", page, err)
	}
	sourcePage, err := journal.ReadAfter(context.Background(), source, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := journal.ReconcileProjection(context.Background(), sourcePage.Events[0])
	if err != nil || !recovered.Duplicate || recovered.Record.Event.History.Source != source {
		t.Fatalf("runner recovery seam = %#v err=%v", recovered, err)
	}
}
