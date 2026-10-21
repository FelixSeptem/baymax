package runner

import (
	"context"
	"testing"
	"time"

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
