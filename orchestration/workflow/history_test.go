package workflow

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/FelixSeptem/baymax/core/history"
	"github.com/FelixSeptem/baymax/core/types"
)

func TestNewNativeEventHistoryRequiresWorkflowSource(t *testing.T) {
	if _, err := NewNativeEventHistory(types.SourceRef{Kind: types.HistorySourceChild, Namespace: "child", ID: "wrong"}); err == nil {
		t.Fatal("accepted non-workflow source")
	}
	if _, err := NewNativeEventHistory(types.SourceRef{Kind: types.HistorySourceWorkflow, Namespace: "workflow.history.v1", ID: "workflow"}); err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowEngineExposesNativeHistoryAppendFailure(t *testing.T) {
	source := types.SourceRef{Kind: types.HistorySourceWorkflow, Namespace: "workflow.history.v1", ID: "workflow-health"}
	root := t.TempDir()
	journal, err := OpenNativeEventHistory(filepath.Join(root, "history.json"), source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	e := New(WithNativeEventHistory(journal))
	e.emitNativeHistory(context.Background(), workflowTimelineContext{workflowID: source.ID, runID: "run"}, StepStatusRunning, map[string]any{"status": "running"})
	if !errors.Is(e.NativeEventHistoryError(), types.ErrSourceUnavailable) {
		t.Fatalf("history append error = %v", e.NativeEventHistoryError())
	}
}

func TestWorkflowTimelineCanAppendExplicitSourceFacts(t *testing.T) {
	source := types.SourceRef{Kind: types.HistorySourceWorkflow, Namespace: "workflow.history.v1", ID: "workflow-history"}
	journal, err := NewNativeEventHistory(source)
	if err != nil {
		t.Fatal(err)
	}
	e := New(WithNativeEventHistory(journal))
	seq := int64(0)
	e.emitTimeline(context.Background(), workflowTimelineContext{runID: "run-1", workflowID: source.ID}, workflowStepTimelineContext{}, StepStatusSucceeded, "workflow.completed", &seq)
	page, err := journal.ReadAfter(context.Background(), source, nil, 10)
	if err != nil || len(page.Events) != 1 || !page.Events[0].IsTerminal() {
		t.Fatalf("workflow source fact = %+v err=%v", page, err)
	}
}

func TestWorkflowNativeHistoryBindsSourceProjectionLedger(t *testing.T) {
	source := types.SourceRef{Kind: types.HistorySourceWorkflow, Namespace: "workflow.commit.v1", ID: "workflow-owner"}
	root := t.TempDir()
	ledger, err := history.OpenProjectionLedger(filepath.Join(root, "ledger.json"), source)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := OpenNativeEventHistory(filepath.Join(root, "history.json"), source, history.WithProjectionLedger(ledger))
	if err != nil {
		t.Fatal(err)
	}
	e := New(WithNativeEventHistory(journal))
	e.emitNativeHistory(context.Background(), workflowTimelineContext{workflowID: source.ID, runID: "run-owner"}, StepStatusSucceeded, map[string]any{"status": "succeeded"})
	if err := e.NativeEventHistoryError(); err != nil {
		t.Fatalf("owner history error = %v", err)
	}
	page, err := ledger.ReadCommittedAfter(context.Background(), nil, 1)
	if err != nil || len(page.Records) != 1 {
		t.Fatalf("workflow projection records = %#v err=%v", page, err)
	}
}
