package planbridge

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/FelixSeptem/baymax/core/history"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/mailbox"
)

type misleadingHistoryReader struct{}

func (misleadingHistoryReader) Capabilities(context.Context, types.SourceRef) (types.HistoryCapabilities, error) {
	return types.HistoryCapabilities{Source: types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner", ID: "r"}, Supported: true, Durable: true, RetentionWindow: "retained"}, nil
}
func (misleadingHistoryReader) ReadAfter(context.Context, types.SourceRef, *types.OpaqueCursor, int) (types.NativeEventPage, error) {
	return types.NativeEventPage{}, errors.New("must not read")
}

func TestNativeHistoryManifestRejectsNonMailboxSourceEvidence(t *testing.T) {
	_, err := CapabilityManifestWithHistory(context.Background(), misleadingHistoryReader{}, types.SourceRef{Kind: types.HistorySourceMailbox, Namespace: "mailbox", ID: "claimed"})
	if !errors.Is(err, types.ErrHistoryUnsupported) {
		t.Fatalf("accepted non-mailbox source evidence: %v", err)
	}
}

func TestNativeHistoryManifestRequiresConcreteRetainedSource(t *testing.T) {
	ctx := context.Background()
	for _, durable := range []bool{false, true} {
		var store mailbox.Store = mailbox.NewMemoryStore(mailbox.Policy{})
		if durable {
			var err error
			store, err = mailbox.NewFileStore(filepath.Join(t.TempDir(), "mailbox.json"), mailbox.Policy{})
			if err != nil {
				t.Fatal(err)
			}
		}
		mb, err := mailbox.New(store)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = mb.Publish(ctx, mailbox.Envelope{MessageID: "manifest", IdempotencyKey: "manifest", Kind: mailbox.KindCommand}); err != nil {
			t.Fatal(err)
		}
		source, err := mb.HistorySource(ctx)
		if err != nil {
			t.Fatal(err)
		}
		manifest, err := CapabilityManifestWithHistory(ctx, mb, source)
		if err != nil {
			t.Fatal(err)
		}
		r, err := NewCapabilityManifestReader(manifest)
		if err != nil {
			t.Fatal(err)
		}
		entry, err := r.Capability(ExecutionFormMailboxAsync)
		if err != nil {
			t.Fatal(err)
		}
		if entry.AfterCursorReplay != CapabilitySupported || (entry.DurableHistory == CapabilitySupported) != durable || entry.EvidenceReference != "fixture:native-event-history.v1" {
			t.Fatalf("manifest: %+v", entry)
		}
		if entry.SourceHistory == nil || entry.SourceHistory.Source != source || entry.SourceHistory.RetentionFloor == nil {
			t.Fatalf("missing source declaration: %+v", entry)
		}
		entry.SourceHistory.Source.ID = "caller-mutated"
		entry.SourceHistory.RetentionFloor.Value = "caller-mutated"
		again, err := r.Capability(ExecutionFormMailboxAsync)
		if err != nil || again.SourceHistory.Source != source || again.SourceHistory.RetentionFloor.Value == "caller-mutated" {
			t.Fatal("manifest aliases source declaration", err)
		}
		for _, form := range []ExecutionForm{ExecutionFormDirect, ExecutionFormSubagent, ExecutionFormTeam, ExecutionFormWorkflow} {
			entry, err = r.Capability(form)
			if err != nil || entry.DurableHistory != CapabilityUnsupported || entry.AfterCursorReplay != CapabilityUnsupported {
				t.Fatalf("upgraded unsupported source: %+v %v", entry, err)
			}
		}
	}
}

func TestNativeHistoryManifestRejectsIncompleteSourceDeclaration(t *testing.T) {
	manifest := DefaultCapabilityManifest()
	entry := &manifest.Forms[len(manifest.Forms)-1]
	entry.SourceHistory = &types.HistoryCapabilities{Source: types.SourceRef{Kind: types.HistorySourceMailbox, Namespace: "mailbox", ID: "m"}, Supported: true, Durable: true}
	entry.DurableHistory, entry.AfterCursorReplay = CapabilitySupported, CapabilitySupported
	if _, err := NewCapabilityManifestReader(manifest); err == nil {
		t.Fatal("accepted history support with no source sequence, cursor scope or retention declaration")
	}
}

func TestCapabilityManifestHistoryProjectionRecoveryRequiresDurableOwnerLedger(t *testing.T) {
	ctx := context.Background()
	source := types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner.commit.v1", ID: "run-manifest"}
	memory, err := history.New(source)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := CapabilityManifestWithHistory(ctx, memory, source)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := NewCapabilityManifestReader(manifest)
	if err != nil {
		t.Fatal(err)
	}
	direct, err := reader.Capability(ExecutionFormDirect)
	if err != nil || direct.SourceHistory == nil || direct.SourceHistory.ProjectionRecovery || direct.SourceHistory.ProjectionRecoveryReason == "" {
		t.Fatalf("memory history capability = %#v err=%v", direct, err)
	}
	root := t.TempDir()
	ledger, err := history.OpenProjectionLedger(filepath.Join(root, "ledger.json"), source)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := history.OpenFile(filepath.Join(root, "history.json"), source, history.WithProjectionLedger(ledger))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err = CapabilityManifestWithHistory(ctx, journal, source)
	if err != nil {
		t.Fatal(err)
	}
	reader, err = NewCapabilityManifestReader(manifest)
	if err != nil {
		t.Fatal(err)
	}
	direct, err = reader.Capability(ExecutionFormDirect)
	if err != nil || direct.SourceHistory == nil || !direct.SourceHistory.ProjectionRecovery || direct.SourceHistory.ProjectionRecoveryEvidence == "" {
		t.Fatalf("file history capability = %#v err=%v", direct, err)
	}
}
