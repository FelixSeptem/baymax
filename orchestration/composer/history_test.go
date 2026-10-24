package composer

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

type historyTestRunner struct{}

func (historyTestRunner) Run(context.Context, types.RunRequest, types.EventHandler) (types.RunResult, error) {
	return types.RunResult{}, nil
}

func (historyTestRunner) Stream(context.Context, types.RunRequest, types.EventHandler) (types.RunResult, error) {
	return types.RunResult{}, nil
}

func TestNewNativeEventHistoryRequiresChildSource(t *testing.T) {
	if _, err := NewNativeEventHistory(types.SourceRef{Kind: types.HistorySourceTeam, Namespace: "team", ID: "wrong"}); err == nil {
		t.Fatal("accepted non-child source")
	}
	if _, err := NewNativeEventHistory(types.SourceRef{Kind: types.HistorySourceChild, Namespace: "child.history.v1", ID: "child"}); err != nil {
		t.Fatal(err)
	}
}

func TestComposerExposesNativeHistoryAppendFailure(t *testing.T) {
	source := types.SourceRef{Kind: types.HistorySourceChild, Namespace: "child.history.v1", ID: "child-health"}
	root := t.TempDir()
	journal, err := OpenNativeEventHistory(filepath.Join(root, "history.json"), source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	c, err := New(nil, WithRunner(historyTestRunner{}), WithNativeEventHistory(journal))
	if err != nil {
		t.Fatal(err)
	}
	c.appendNativeHistory(context.Background(), types.Event{Version: types.EventSchemaVersionV1, RunID: source.ID, Type: "run.started", Time: time.Now()})
	if !errors.Is(c.NativeEventHistoryError(), types.ErrSourceUnavailable) {
		t.Fatalf("history append error = %v", c.NativeEventHistoryError())
	}
}

func TestComposerNativeHistoryBindsSourceProjectionLedger(t *testing.T) {
	source := types.SourceRef{Kind: types.HistorySourceChild, Namespace: "child.commit.v1", ID: "child-owner"}
	root := t.TempDir()
	ledger, err := history.OpenProjectionLedger(filepath.Join(root, "ledger.json"), source)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := OpenNativeEventHistory(filepath.Join(root, "history.json"), source, history.WithProjectionLedger(ledger))
	if err != nil {
		t.Fatal(err)
	}
	c, err := New(nil, WithRunner(historyTestRunner{}), WithNativeEventHistory(journal))
	if err != nil {
		t.Fatal(err)
	}
	c.appendNativeHistory(context.Background(), types.Event{Version: types.EventSchemaVersionV1, RunID: source.ID, Type: "run.finished", Time: time.Now()})
	if err := c.NativeEventHistoryError(); err != nil {
		t.Fatalf("owner history error = %v", err)
	}
	page, err := ledger.ReadCommittedAfter(context.Background(), nil, 1)
	if err != nil || len(page.Records) != 1 {
		t.Fatalf("child projection records = %#v err=%v", page, err)
	}
}
