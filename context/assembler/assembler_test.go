package assembler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/context/journal"
	"github.com/FelixSeptem/baymax/context/provider"
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

func TestAssemblerCA2RoutesToStage2ByKeyword(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.Stage2.Provider = "file"
	stage2File := filepath.Join(t.TempDir(), "stage2.jsonl")
	if err := os.WriteFile(stage2File, []byte(`{"session_id":"session-1","content":"external-ctx"}`), 0o600); err != nil {
		t.Fatalf("write stage2 file: %v", err)
	}
	cfg.CA2.Stage2.FilePath = stage2File
	cfg.CA2.Routing.TriggerKeywords = []string{"lookup"}
	cfg.CA2.Routing.MinInputChars = 9999
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	req := types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: "ca1",
		Input:         "please lookup details",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}
	outReq, result, err := a.Assemble(context.Background(), req, types.ModelRequest{
		RunID:    req.RunID,
		Input:    req.Input,
		Messages: req.Messages,
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Stage.Status != types.AssembleStageStatusStage2Used {
		t.Fatalf("stage status = %q, want stage2_used", result.Stage.Status)
	}
	found := false
	for _, msg := range outReq.Messages {
		if strings.Contains(msg.Content, "external-ctx") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("stage2 context not appended: %#v", outReq.Messages)
	}
}

func TestAssemblerCA2Stage2BestEffort(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.Stage2.Provider = "rag"
	cfg.CA2.StagePolicy.Stage2 = "best_effort"
	cfg.CA2.Routing.MinInputChars = 1
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	_, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: "ca1",
		Input:         "x",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-1",
		Input:    "x",
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if err != nil {
		t.Fatalf("Assemble should continue in best_effort: %v", err)
	}
	if result.Stage.Status != types.AssembleStageStatusDegraded {
		t.Fatalf("stage status = %q, want degraded", result.Stage.Status)
	}
}

func TestAssemblerCA2Stage2FailFast(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.Stage2.Provider = "db"
	cfg.CA2.StagePolicy.Stage2 = "fail_fast"
	cfg.CA2.Routing.MinInputChars = 1
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	_, _, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: "ca1",
		Input:         "x",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-1",
		Input:    "x",
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if !errors.Is(err, provider.ErrProviderNotReady) {
		t.Fatalf("err = %v, want ErrProviderNotReady", err)
	}
}

func TestAssemblerCA2RecapAppended(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.Routing.MinInputChars = 9999
	cfg.CA2.Routing.TriggerKeywords = nil
	cfg.CA2.TailRecap.Enabled = true
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	outReq, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: "ca1",
		Input:         "short",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-1",
		Input:    "short",
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Recap.Status != types.RecapStatusAppended && result.Recap.Status != types.RecapStatusTruncated {
		t.Fatalf("recap status = %q, want appended/truncated", result.Recap.Status)
	}
	last := outReq.Messages[len(outReq.Messages)-1].Content
	if !strings.HasPrefix(last, "tail_recap:") {
		t.Fatalf("tail recap message missing: %q", last)
	}
}
