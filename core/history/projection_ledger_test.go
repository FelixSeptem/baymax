package history

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestProjectionLedgerCommitsReopensAndAcknowledgesIdempotently(t *testing.T) {
	ctx := context.Background()
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.commit.v1", ID: "run-1"}
	path := filepath.Join(t.TempDir(), "projection-ledger.json")
	ledger, err := OpenProjectionLedger(path, source)
	if err != nil {
		t.Fatal(err)
	}
	record, err := ledger.Commit(ctx, types.SourceCommitInput{
		Event: types.NativeExecutionEvent{
			Version:    types.NativeExecutionEventVersionV1,
			NativeType: "runner.completed",
			Event:      types.EventEnvelope{Payload: map[string]any{"secret": "not-retained"}},
		},
		TerminalEvidence: "succeeded",
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.Event.History == nil || record.Event.History.Source != source || record.Event.History.Sequence != 1 || record.Event.Event.Payload != nil {
		t.Fatalf("committed record = %#v", record)
	}
	first, err := ledger.AcknowledgeProjection(ctx, types.ProjectionAcknowledgement{
		Source:     source,
		EventID:    record.Event.History.EventID,
		Digest:     record.Digest,
		Reference:  "projection/1",
		NextCursor: record.Cursor,
	})
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := ledger.AcknowledgeProjection(ctx, types.ProjectionAcknowledgement{
		Source:     source,
		EventID:    record.Event.History.EventID,
		Digest:     record.Digest,
		Reference:  "projection/1",
		NextCursor: record.Cursor,
	})
	if err != nil || !duplicate.Duplicate || duplicate.Source != first.Source || duplicate.EventID != first.EventID || duplicate.Digest != first.Digest || duplicate.Reference != first.Reference || duplicate.NextCursor != first.NextCursor {
		t.Fatalf("duplicate acknowledgement = %#v err=%v", duplicate, err)
	}
	reopened, err := OpenProjectionLedger(path, source)
	if err != nil {
		t.Fatal(err)
	}
	page, err := reopened.ReadCommittedAfter(ctx, nil, 1)
	if err != nil || len(page.Records) != 1 || page.Records[0].Digest != record.Digest || page.Records[0].Event.Event.Payload != nil {
		t.Fatalf("reopened records = %#v err=%v", page, err)
	}
}

func TestProjectionLedgerRejectsConflictingAcknowledgement(t *testing.T) {
	ctx := context.Background()
	source := types.SourceRef{Kind: types.HistorySourceWorkflow, Namespace: "workflow.commit.v1", ID: "workflow-1"}
	ledger, err := NewProjectionLedger(source)
	if err != nil {
		t.Fatal(err)
	}
	record, err := ledger.Commit(ctx, types.SourceCommitInput{Event: types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "workflow.completed"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ledger.AcknowledgeProjection(ctx, types.ProjectionAcknowledgement{Source: source, EventID: record.Event.History.EventID, Digest: record.Digest, Reference: "projection/1", NextCursor: record.Cursor})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ledger.AcknowledgeProjection(ctx, types.ProjectionAcknowledgement{Source: source, EventID: record.Event.History.EventID, Digest: "changed", Reference: "projection/1", NextCursor: record.Cursor})
	if !errors.Is(err, types.ErrHistoryConflict) {
		t.Fatalf("conflicting acknowledgement error = %v", err)
	}
}

func TestProjectionLedgerExpiresTrimmedCommitAndRejectsInvalidAcknowledgement(t *testing.T) {
	ctx := context.Background()
	source := types.SourceRef{Kind: types.HistorySourceTeam, Namespace: "team.commit.v1", ID: "team-1"}
	ledger, err := NewProjectionLedger(source, WithProjectionRetention(1))
	if err != nil {
		t.Fatal(err)
	}
	_, err = ledger.Commit(ctx, types.SourceCommitInput{Event: types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "team.member.completed"}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := ledger.Commit(ctx, types.SourceCommitInput{Event: types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "team.completed"}, TerminalEvidence: "succeeded"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ledger.ReadCommittedAfter(ctx, ptrCursor(encodeCursor(source, 0)), 1); !errors.Is(err, types.ErrCursorExpired) {
		t.Fatalf("expired commit cursor error = %v", err)
	}
	if _, err = ledger.AcknowledgeProjection(ctx, types.ProjectionAcknowledgement{Source: source, EventID: second.Event.History.EventID, Digest: second.Digest, Reference: "", NextCursor: second.Cursor}); !errors.Is(err, types.ErrHistoryInvalid) {
		t.Fatalf("invalid acknowledgement error = %v", err)
	}
	foreign := source
	foreign.ID = "team-2"
	if _, err = ledger.AcknowledgeProjection(ctx, types.ProjectionAcknowledgement{Source: foreign, EventID: second.Event.History.EventID, Digest: second.Digest, Reference: "projection/2", NextCursor: second.Cursor}); !errors.Is(err, types.ErrCursorForeign) {
		t.Fatalf("foreign acknowledgement error = %v", err)
	}
}

func TestJournalAppendWithProjectionLedgerCreatesMatchingCommitEvidence(t *testing.T) {
	ctx := context.Background()
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.commit.v1", ID: "run-journal"}
	ledger, err := NewProjectionLedger(source)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := New(source, WithProjectionLedger(ledger))
	if err != nil {
		t.Fatal(err)
	}
	identity, err := journal.Append(ctx, types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "runner.completed"})
	if err != nil {
		t.Fatal(err)
	}
	page, err := ledger.ReadCommittedAfter(ctx, nil, 1)
	if err != nil || len(page.Records) != 1 || page.Records[0].Event.History == nil || page.Records[0].Event.History.EventID != identity.EventID || page.Records[0].Event.History.Sequence != identity.Sequence {
		t.Fatalf("commit evidence = %#v err=%v", page, err)
	}
}

func TestJournalAppendMarksProjectionRecoveryRequiredWhenLedgerPersistenceFails(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	source := types.SourceRef{Kind: types.HistorySourceWorkflow, Namespace: "workflow.commit.v1", ID: "workflow-journal"}
	ledger, err := OpenProjectionLedger(filepath.Join(root, "missing", "ledger.json"), source)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := OpenFile(filepath.Join(root, "history.json"), source, WithProjectionLedger(ledger))
	if err != nil {
		t.Fatal(err)
	}
	identity, err := journal.Append(ctx, types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "workflow.completed"})
	if !errors.Is(err, types.ErrProjectionRecoveryRequired) || identity.EventID == "" {
		t.Fatalf("append outcome = %#v err=%v", identity, err)
	}
	page, readErr := journal.ReadAfter(ctx, source, nil, 1)
	if readErr != nil || len(page.Events) != 1 || page.Events[0].History.EventID != identity.EventID {
		t.Fatalf("journal source fact = %#v err=%v", page, readErr)
	}
	if _, readErr = ledger.ReadCommittedAfter(ctx, nil, 1); readErr != nil {
		t.Fatalf("ledger read should expose an empty recovery boundary without execution: %v", readErr)
	}
}

func TestJournalCapabilitiesDeclareProjectionRecoveryOnlyForDurableLedger(t *testing.T) {
	ctx := context.Background()
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.commit.v1", ID: "run-capabilities"}
	memoryLedger, err := NewProjectionLedger(source)
	if err != nil {
		t.Fatal(err)
	}
	memoryJournal, err := New(source, WithProjectionLedger(memoryLedger))
	if err != nil {
		t.Fatal(err)
	}
	memory, err := memoryJournal.Capabilities(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	if memory.ProjectionRecovery || memory.ProjectionRecoveryReason == "" {
		t.Fatalf("memory projection recovery = %#v", memory)
	}
	root := t.TempDir()
	fileLedger, err := OpenProjectionLedger(filepath.Join(root, "ledger.json"), source)
	if err != nil {
		t.Fatal(err)
	}
	fileJournal, err := OpenFile(filepath.Join(root, "history.json"), source, WithProjectionLedger(fileLedger))
	if err != nil {
		t.Fatal(err)
	}
	file, err := fileJournal.Capabilities(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	if !file.ProjectionRecovery || file.ProjectionRecoveryEvidence == "" {
		t.Fatalf("file projection recovery = %#v", file)
	}
}
