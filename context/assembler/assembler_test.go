package assembler

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/FelixSeptem/baymax/context/journal"
	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestAssemblerStablePrefixHashWithinSessionVersion(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	req := types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: "ca1",
		Messages:      []types.Message{{Role: "system", Content: "stable"}},
	}
	modelReq := types.ModelRequest{RunID: req.RunID, Messages: req.Messages}

	_, r1, err := a.Assemble(context.Background(), req, modelReq)
	if err != nil {
		t.Fatalf("first assemble failed: %v", err)
	}
	_, r2, err := a.Assemble(context.Background(), req, modelReq)
	if err != nil {
		t.Fatalf("second assemble failed: %v", err)
	}
	if r1.Prefix.PrefixHash == "" || r1.Prefix.PrefixHash != r2.Prefix.PrefixHash {
		t.Fatalf("prefix hash mismatch: %q vs %q", r1.Prefix.PrefixHash, r2.Prefix.PrefixHash)
	}
}

func TestAssemblerFailFastOnPrefixDrift(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	base := types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: "ca1",
		Messages:      []types.Message{{Role: "system", Content: "stable"}},
	}

	if _, _, err := a.Assemble(context.Background(), base, types.ModelRequest{RunID: base.RunID, Messages: base.Messages}); err != nil {
		t.Fatalf("first assemble failed: %v", err)
	}
	drift := base
	drift.Messages = []types.Message{{Role: "system", Content: "changed"}}
	_, result, err := a.Assemble(context.Background(), drift, types.ModelRequest{RunID: drift.RunID, Messages: drift.Messages})
	if err == nil {
		t.Fatal("expected fail-fast guard error")
	}
	if result.GuardFailure != "hash.prefix.drift" {
		t.Fatalf("guard failure = %q, want hash.prefix.drift", result.GuardFailure)
	}
}

func TestAssemblerRejectsDBBackendPlaceholder(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.Storage.Backend = "db"
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	_, _, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: "ca1",
	}, types.ModelRequest{RunID: "run-1"})
	if !errors.Is(err, journal.ErrBackendNotReady) {
		t.Fatalf("err = %v, want ErrBackendNotReady", err)
	}
}
