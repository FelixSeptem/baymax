package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	fmt.Printf("mailbox-history durable=%t\nmailbox-history restart-events=%d\nmailbox-history terminal=%s\n", caps.Durable, len(page.Events), page.Events[1].History.TerminalOutcome)
	fmt.Printf("mailbox-history malformed=%t\nmailbox-history foreign=%t\nmailbox-history expired=%t\nnative-history unsupported=%s\n", errors.Is(malformed, types.ErrCursorMalformed), errors.Is(foreign, types.ErrCursorForeign), expired, strings.Join(unsupported, ","))
	return nil
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
