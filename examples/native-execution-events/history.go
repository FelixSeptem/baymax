package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/FelixSeptem/baymax/core/history"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/mailbox"
)

// demonstrateHistory separates source lifecycle writes from the reopened reader.
// The reader phase never calls an execution, delivery or recovery operation.
func demonstrateHistory() (resultErr error) {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "baymax-native-history-example-")
	if err != nil {
		return err
	}
	defer func() {
		// Only this example's newly created temporary directory is removed.
		resultErr = errors.Join(resultErr, os.RemoveAll(dir))
	}()
	path := filepath.Join(dir, "mailbox.json")
	source, confirmed, err := writeMailboxHistory(ctx, path)
	if err != nil {
		return err
	}
	store, err := mailbox.NewFileStore(path, mailbox.Policy{})
	if err != nil {
		return err
	}
	reader, err := mailbox.New(store)
	if err != nil {
		return err
	}
	caps, err := reader.Capabilities(ctx, source)
	if err != nil {
		return err
	}
	page, err := reader.ReadAfter(ctx, source, confirmed, 10)
	if err != nil {
		return err
	}
	if len(page.Events) != 2 || !page.Complete || page.Events[1].History.TerminalOutcome != "acked" {
		return fmt.Errorf("unexpected retained mailbox terminal evidence")
	}
	_, malformed := reader.ReadAfter(ctx, source, &types.OpaqueCursor{Value: "not-a-cursor"}, 10)
	foreignSource := source
	foreignSource.ID = "another-source"
	_, foreign := reader.ReadAfter(ctx, foreignSource, confirmed, 10)
	if !errors.Is(malformed, types.ErrCursorMalformed) || !errors.Is(foreign, types.ErrCursorForeign) {
		return fmt.Errorf("cursor errors did not fail closed")
	}
	expired, err := demonstrateRetentionExpiry(ctx)
	if err != nil {
		return err
	}
	var unsupported []string
	audit := types.UnsupportedNativeEventHistoryReader{}
	for _, kind := range []types.HistorySourceKind{types.HistorySourceRunner, types.HistorySourceChild, types.HistorySourceTeam, types.HistorySourceWorkflow} {
		ref := types.SourceRef{Kind: kind, Namespace: "example", ID: "unsupported-source"}
		capability, err := audit.Capabilities(ctx, ref)
		if err != nil || capability.Supported {
			return fmt.Errorf("unexpected history capability for %s", kind)
		}
		if _, err := audit.ReadAfter(ctx, ref, nil, 1); !errors.Is(err, types.ErrHistoryUnsupported) {
			return fmt.Errorf("missing typed unsupported for %s", kind)
		}
		unsupported = append(unsupported, string(kind))
	}
	reconciled, duplicate, conflict, projectionUnsupported, err := demonstrateProjectionRecovery(ctx, dir)
	if err != nil {
		return err
	}
	fmt.Printf("mailbox-history durable=%t\nmailbox-history restart-events=%d\nmailbox-history terminal=%s\n", caps.Durable, len(page.Events), page.Events[1].History.TerminalOutcome)
	fmt.Printf("mailbox-history malformed=%t\nmailbox-history foreign=%t\nmailbox-history expired=%t\nnative-history unsupported=%s\n", errors.Is(malformed, types.ErrCursorMalformed), errors.Is(foreign, types.ErrCursorForeign), expired, strings.Join(unsupported, ","))
	fmt.Printf("source-projection reconciled=%t\nsource-projection duplicate=%t\nsource-projection conflict=%t\nsource-projection unsupported=%t\n", reconciled, duplicate, conflict, projectionUnsupported)
	return nil
}

func demonstrateProjectionRecovery(ctx context.Context, dir string) (bool, bool, bool, bool, error) {
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "example.runner.commit.v1", ID: "projection-recovery"}
	ledgerPath := filepath.Join(dir, "projection", "ledger.json")
	historyPath := filepath.Join(dir, "projection-history.json")
	ledger, err := history.OpenProjectionLedger(ledgerPath, source)
	if err != nil {
		return false, false, false, false, err
	}
	journal, err := history.OpenFile(historyPath, source, history.WithProjectionLedger(ledger))
	if err != nil {
		return false, false, false, false, err
	}
	_, appendErr := journal.Append(ctx, types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "run.finished"})
	if !errors.Is(appendErr, types.ErrProjectionRecoveryRequired) {
		return false, false, false, false, fmt.Errorf("expected recovery-required append, got %v", appendErr)
	}
	if err := os.MkdirAll(filepath.Dir(ledgerPath), 0o700); err != nil {
		return false, false, false, false, err
	}
	reopenedLedger, err := history.OpenProjectionLedger(ledgerPath, source)
	if err != nil {
		return false, false, false, false, err
	}
	reopenedJournal, err := history.OpenFile(historyPath, source, history.WithProjectionLedger(reopenedLedger))
	if err != nil {
		return false, false, false, false, err
	}
	page, err := reopenedJournal.ReadAfter(ctx, source, nil, 1)
	if err != nil || len(page.Events) != 1 {
		return false, false, false, false, fmt.Errorf("read source recovery fact: %v", err)
	}
	recovered, err := reopenedJournal.ReconcileProjection(ctx, page.Events[0])
	if err != nil {
		return false, false, false, false, err
	}
	repeated, err := reopenedJournal.ReconcileProjection(ctx, page.Events[0])
	if err != nil {
		return false, false, false, false, err
	}
	conflictEvent := page.Events[0]
	conflictEvent.NativeType = "run.changed"
	_, conflictErr := reopenedJournal.ReconcileProjection(ctx, conflictEvent)
	if !errors.Is(conflictErr, types.ErrHistoryConflict) {
		return false, false, false, false, fmt.Errorf("expected conflict, got %v", conflictErr)
	}
	unbound, err := history.New(source)
	if err != nil {
		return false, false, false, false, err
	}
	if _, err := unbound.Append(ctx, types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "run.finished"}); err != nil {
		return false, false, false, false, err
	}
	unboundPage, err := unbound.ReadAfter(ctx, source, nil, 1)
	if err != nil {
		return false, false, false, false, err
	}
	_, unsupportedErr := unbound.ReconcileProjection(ctx, unboundPage.Events[0])
	if !errors.Is(unsupportedErr, types.ErrHistoryUnsupported) {
		return false, false, false, false, fmt.Errorf("expected unsupported, got %v", unsupportedErr)
	}
	return !recovered.Duplicate, repeated.Duplicate, true, true, nil
}

func writeMailboxHistory(ctx context.Context, path string) (types.SourceRef, *types.OpaqueCursor, error) {
	store, err := mailbox.NewFileStore(path, mailbox.Policy{})
	if err != nil {
		return types.SourceRef{}, nil, err
	}
	owner, err := mailbox.New(store)
	if err != nil {
		return types.SourceRef{}, nil, err
	}
	if _, err := owner.Publish(ctx, mailbox.Envelope{MessageID: "example-message", IdempotencyKey: "example-message", Kind: mailbox.KindCommand, RunID: "example-run"}); err != nil {
		return types.SourceRef{}, nil, err
	}
	source, err := owner.HistorySource(ctx)
	if err != nil {
		return types.SourceRef{}, nil, err
	}
	admission, err := owner.ReadAfter(ctx, source, nil, 1)
	if err != nil {
		return types.SourceRef{}, nil, err
	}
	// A host persists this cursor only after its own projection is confirmed.
	confirmed := admission.NextCursor
	record, ok, err := owner.Consume(ctx, "example-consumer")
	if err != nil || !ok {
		return types.SourceRef{}, nil, fmt.Errorf("consume example message: ok=%t err=%v", ok, err)
	}
	if _, err := owner.Ack(ctx, record.Envelope.MessageID, "example-consumer"); err != nil {
		return types.SourceRef{}, nil, err
	}
	return source, confirmed, store.Flush()
}

func demonstrateRetentionExpiry(ctx context.Context) (bool, error) {
	// Use a separate process-local source to make the fixed retention bound visible.
	owner, err := mailbox.New(mailbox.NewMemoryStore(mailbox.Policy{}))
	if err != nil {
		return false, err
	}
	source, err := owner.HistorySource(ctx)
	if err != nil {
		return false, err
	}
	start, err := owner.ReadAfter(ctx, source, nil, 1)
	if err != nil {
		return false, err
	}
	for i := 0; i <= mailbox.HistoryRetentionEntries; i++ {
		id := fmt.Sprintf("retention-%d", i)
		if _, err := owner.Publish(ctx, mailbox.Envelope{MessageID: id, IdempotencyKey: id, Kind: mailbox.KindCommand}); err != nil {
			return false, err
		}
	}
	page, err := owner.ReadAfter(ctx, source, start.NextCursor, 1)
	if !errors.Is(err, types.ErrCursorExpired) || page.RetentionFloor == nil || len(page.Events) != 0 {
		return false, fmt.Errorf("retention did not return typed expiry with floor")
	}
	return true, nil
}
