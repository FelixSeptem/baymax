package history

import (
	"context"
	"errors"
	"os"
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

func TestJournalProjectionRecoveryReopensSourceFactAndAcknowledgesIdempotently(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	source := types.SourceRef{Kind: types.HistorySourceWorkflow, Namespace: "workflow.commit.v1", ID: "workflow-reopen"}
	ledgerPath := filepath.Join(root, "ledger", "projection.json")
	ledger, err := OpenProjectionLedger(ledgerPath, source)
	if err != nil {
		t.Fatal(err)
	}
	journalPath := filepath.Join(root, "history.json")
	journal, err := OpenFile(journalPath, source, WithProjectionLedger(ledger))
	if err != nil {
		t.Fatal(err)
	}
	identity, err := journal.Append(ctx, types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "workflow.completed"})
	if !errors.Is(err, types.ErrProjectionRecoveryRequired) || identity.EventID == "" {
		t.Fatalf("append outcome = %#v err=%v", identity, err)
	}
	if err := os.MkdirAll(filepath.Dir(ledgerPath), 0o700); err != nil {
		t.Fatal(err)
	}
	reopenedLedger, err := OpenProjectionLedger(ledgerPath, source)
	if err != nil {
		t.Fatal(err)
	}
	reopenedJournal, err := OpenFile(journalPath, source, WithProjectionLedger(reopenedLedger))
	if err != nil {
		t.Fatal(err)
	}
	page, err := reopenedJournal.ReadAfter(ctx, source, nil, 1)
	if err != nil || len(page.Events) != 1 {
		t.Fatalf("reopened source facts = %#v err=%v", page, err)
	}
	recovered, err := reopenedJournal.ReconcileProjection(ctx, page.Events[0])
	if err != nil || recovered.Duplicate || recovered.Record.Event.History == nil || recovered.Record.Event.History.EventID != identity.EventID {
		t.Fatalf("recovery = %#v err=%v", recovered, err)
	}
	duplicate, err := reopenedJournal.ReconcileProjection(ctx, page.Events[0])
	if err != nil || !duplicate.Duplicate || duplicate.Record.Digest != recovered.Record.Digest {
		t.Fatalf("duplicate recovery = %#v err=%v", duplicate, err)
	}
	ack, err := reopenedLedger.AcknowledgeProjection(ctx, types.ProjectionAcknowledgement{
		Source: source, EventID: recovered.Record.Event.History.EventID, Digest: recovered.Record.Digest,
		Reference: "projection/recovered", NextCursor: recovered.Record.Cursor,
	})
	if err != nil || ack.Duplicate {
		t.Fatalf("acknowledgement = %#v err=%v", ack, err)
	}
	ackDuplicate, err := reopenedLedger.AcknowledgeProjection(ctx, types.ProjectionAcknowledgement{
		Source: source, EventID: recovered.Record.Event.History.EventID, Digest: recovered.Record.Digest,
		Reference: "projection/recovered", NextCursor: recovered.Record.Cursor,
	})
	if err != nil || !ackDuplicate.Duplicate {
		t.Fatalf("duplicate acknowledgement = %#v err=%v", ackDuplicate, err)
	}
	finalPage, err := reopenedLedger.ReadCommittedAfter(ctx, nil, 1)
	if err != nil || len(finalPage.Records) != 1 || finalPage.Records[0].Event.History.EventID != identity.EventID {
		t.Fatalf("recovered ledger = %#v err=%v", finalPage, err)
	}
}

func TestJournalProjectionRecoveryRejectsConflictBeforeLedgerMutation(t *testing.T) {
	ctx := context.Background()
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.commit.v1", ID: "runner-conflict"}
	ledger, err := NewProjectionLedger(source)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := New(source, WithProjectionLedger(ledger))
	if err != nil {
		t.Fatal(err)
	}
	_, err = journal.Append(ctx, types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "run.finished"})
	if err != nil {
		t.Fatal(err)
	}
	page, err := journal.ReadAfter(ctx, source, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := journal.ReconcileProjection(ctx, page.Events[0])
	if err != nil || !duplicate.Duplicate {
		t.Fatalf("identical source fact should be duplicate = %#v err=%v", duplicate, err)
	}
	conflict := page.Events[0]
	conflict.NativeType = "run.changed"
	if _, err = journal.ReconcileProjection(ctx, conflict); !errors.Is(err, types.ErrHistoryConflict) {
		t.Fatalf("conflicting source fact error = %v", err)
	}
	committed, err := ledger.ReadCommittedAfter(ctx, nil, 1)
	if err != nil || len(committed.Records) != 1 {
		t.Fatalf("ledger after conflict = %#v err=%v", committed, err)
	}
}

func TestJournalProjectionRecoveryRequiresExplicitLedgerBinding(t *testing.T) {
	ctx := context.Background()
	source := types.SourceRef{Kind: types.HistorySourceChild, Namespace: "child.commit.v1", ID: "child-unbound"}
	j, err := New(source)
	if err != nil {
		t.Fatal(err)
	}
	event := types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "child.completed"}
	if _, err = j.Append(ctx, event); err != nil {
		t.Fatal(err)
	}
	page, err := j.ReadAfter(ctx, source, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = j.ReconcileProjection(ctx, page.Events[0]); !errors.Is(err, types.ErrHistoryUnsupported) {
		t.Fatalf("unbound recovery error = %v", err)
	}
}

func TestJournalProjectionRecoveryRejectsForeignMalformedAndGappedFacts(t *testing.T) {
	ctx := context.Background()
	source := types.SourceRef{Kind: types.HistorySourceTeam, Namespace: "team.commit.v1", ID: "team-validation"}
	ledger, err := NewProjectionLedger(source)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := New(source, WithProjectionLedger(ledger))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = journal.Append(ctx, types.NativeExecutionEvent{Version: types.NativeExecutionEventVersionV1, NativeType: "team.completed"}); err != nil {
		t.Fatal(err)
	}
	page, err := journal.ReadAfter(ctx, source, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	foreign := page.Events[0]
	foreign.History = &types.NativeHistoryIdentity{Source: types.SourceRef{Kind: source.Kind, Namespace: source.Namespace, ID: "other-team"}, Sequence: 1, EventID: foreign.History.EventID}
	if _, err = journal.ReconcileProjection(ctx, foreign); !errors.Is(err, types.ErrCursorForeign) {
		t.Fatalf("foreign recovery error = %v", err)
	}
	malformed := page.Events[0]
	malformed.Event.Payload = map[string]any{"secret": "must not be accepted"}
	if _, err = journal.ReconcileProjection(ctx, malformed); !errors.Is(err, types.ErrHistoryConflict) {
		t.Fatalf("malformed source fact error = %v", err)
	}
	gapped := page.Events[0]
	gapped.History = &types.NativeHistoryIdentity{Source: source, Sequence: 2, EventID: eventID(source, 2)}
	gapped.Event.EventID = gapped.History.EventID
	if _, err = journal.ReconcileProjection(ctx, gapped); !errors.Is(err, types.ErrHistoryGap) {
		t.Fatalf("gapped journal fact error = %v", err)
	}
	committed, err := ledger.ReadCommittedAfter(ctx, nil, 1)
	if err != nil || len(committed.Records) != 1 {
		t.Fatalf("ledger mutated by rejected recovery = %#v err=%v", committed, err)
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
