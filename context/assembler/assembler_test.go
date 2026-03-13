package assembler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	if result.Stage.Stage2ReasonCode == "" || result.Stage.Stage2ErrorLayer == "" {
		t.Fatalf("expected stage2 layered error fields, got %#v", result.Stage)
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
	if err == nil {
		t.Fatal("expected fail_fast error, got nil")
	}
	if !strings.Contains(err.Error(), "endpoint is required") {
		t.Fatalf("err = %v, want endpoint required", err)
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

func TestAssemblerCA2Stage2ContextRedacted(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.Stage2.Provider = "file"
	stage2File := filepath.Join(t.TempDir(), "stage2.jsonl")
	if err := os.WriteFile(stage2File, []byte(`{"session_id":"session-1","content":"{\"access_token\":\"secret-token\"}"}`), 0o600); err != nil {
		t.Fatalf("write stage2 file: %v", err)
	}
	cfg.CA2.Stage2.FilePath = stage2File
	cfg.CA2.Routing.MinInputChars = 1
	a := New(
		func() runtimeconfig.ContextAssemblerConfig { return cfg },
		WithRedactionConfigProvider(func() runtimeconfig.SecurityRedactionConfig {
			return runtimeconfig.SecurityRedactionConfig{
				Enabled:  true,
				Strategy: runtimeconfig.SecurityRedactionKeyword,
				Keywords: []string{"token"},
			}
		}),
	)
	outReq, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: "ca1",
		Input:         "lookup",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-1",
		Input:    "lookup",
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Stage.Status != types.AssembleStageStatusStage2Used {
		t.Fatalf("stage status = %q, want stage2_used", result.Stage.Status)
	}
	foundMasked := false
	for _, msg := range outReq.Messages {
		if strings.Contains(msg.Content, `"access_token":"***"`) {
			foundMasked = true
			break
		}
	}
	if !foundMasked {
		t.Fatalf("expected redacted stage2 content, got %#v", outReq.Messages)
	}
}

func TestAssemblerCA2Stage2DiagnosticsFields(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.Stage2.Provider = "file"
	stage2File := filepath.Join(t.TempDir(), "stage2.jsonl")
	content := strings.Join([]string{
		`{"session_id":"session-1","content":"ctx-a"}`,
		`{"session_id":"session-1","content":"ctx-b"}`,
	}, "\n")
	if err := os.WriteFile(stage2File, []byte(content), 0o600); err != nil {
		t.Fatalf("write stage2 file: %v", err)
	}
	cfg.CA2.Stage2.FilePath = stage2File
	cfg.CA2.Routing.MinInputChars = 1

	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	_, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: "ca1",
		Input:         "lookup",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-1",
		Input:    "lookup",
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Stage.Stage2HitCount != 2 {
		t.Fatalf("stage2_hit_count = %d, want 2", result.Stage.Stage2HitCount)
	}
	if result.Stage.Stage2Source != "file" {
		t.Fatalf("stage2_source = %q, want file", result.Stage.Stage2Source)
	}
	if result.Stage.Stage2Reason != "ok" {
		t.Fatalf("stage2_reason = %q, want ok", result.Stage.Stage2Reason)
	}
	if result.Stage.Stage2ReasonCode != "ok" {
		t.Fatalf("stage2_reason_code = %q, want ok", result.Stage.Stage2ReasonCode)
	}
	if result.Stage.Stage2ErrorLayer != "" {
		t.Fatalf("stage2_error_layer = %q, want empty", result.Stage.Stage2ErrorLayer)
	}
	if result.Stage.Stage2Profile != "file" {
		t.Fatalf("stage2_profile = %q, want file", result.Stage.Stage2Profile)
	}
}

func TestAssemblerCA3EmergencyRejectsLowPriorityStage2(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.Stage2.Provider = "file"
	stage2File := filepath.Join(t.TempDir(), "stage2.jsonl")
	if err := os.WriteFile(stage2File, []byte(`{"session_id":"session-1","content":"ctx"}`), 0o600); err != nil {
		t.Fatalf("write stage2 file: %v", err)
	}
	cfg.CA2.Stage2.FilePath = stage2File
	cfg.CA2.Routing.MinInputChars = 1
	cfg.CA3.Enabled = true
	cfg.CA3.MaxContextTokens = 100
	cfg.CA3.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	cfg.CA3.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	cfg.CA3.Emergency.RejectLowPriority = true
	cfg.CA3.Emergency.HighPriorityTokens = []string{"urgent"}

	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	_, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: "ca1",
		Input:         strings.Repeat("x", 500),
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-1",
		Input:    strings.Repeat("x", 500),
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Stage.Stage2SkipReason != "ca3.emergency.low_priority_rejected" {
		t.Fatalf("stage2 skip reason = %q", result.Stage.Stage2SkipReason)
	}
	if result.Stage.PressureZone == "" {
		t.Fatalf("pressure zone should be populated: %#v", result.Stage)
	}
}

func TestAssemblerCA3ProtectedMessagesNotPruned(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA3.Enabled = true
	cfg.CA3.MaxContextTokens = 40
	cfg.CA3.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	cfg.CA3.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 4, Comfort: 8, Warning: 12, Danger: 16, Emergency: 20,
	}
	cfg.CA3.Prune.TargetPercent = 30
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	msgs := []types.Message{
		{Role: "system", Content: "base"},
		{Role: "user", Content: "critical: keep this message no matter what"},
		{Role: "user", Content: strings.Repeat("filler ", 80)},
	}
	outReq, _, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-2",
		SessionID:     "session-1",
		PrefixVersion: "ca1",
		Input:         strings.Repeat("need trim", 10),
		Messages:      msgs,
	}, types.ModelRequest{
		RunID:    "run-2",
		Input:    strings.Repeat("need trim", 10),
		Messages: msgs,
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	foundCritical := false
	for _, msg := range outReq.Messages {
		if strings.Contains(msg.Content, "critical: keep this message") {
			foundCritical = true
		}
	}
	if !foundCritical {
		t.Fatalf("critical message should not be pruned: %#v", outReq.Messages)
	}
}

func TestAssemblerCA3SpillIdempotentAcrossRetry(t *testing.T) {
	dir := t.TempDir()
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(dir, "journal.jsonl")
	cfg.CA3.Enabled = true
	cfg.CA3.MaxContextTokens = 80
	cfg.CA3.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	cfg.CA3.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 8, Comfort: 16, Warning: 24, Danger: 32, Emergency: 40,
	}
	cfg.CA3.Spill.Path = filepath.Join(dir, "spill.jsonl")
	cfg.CA3.Spill.Backend = "file"
	a := New(func() runtimeconfig.ContextAssemblerConfig { return cfg })
	req := types.ContextAssembleRequest{
		RunID:         "run-3",
		SessionID:     "session-1",
		PrefixVersion: "ca1",
		Input:         strings.Repeat("large ", 100),
		Messages: []types.Message{
			{Role: "system", Content: "base"},
			{Role: "user", Content: strings.Repeat("payload ", 120)},
		},
	}
	modelReq := types.ModelRequest{RunID: req.RunID, Input: req.Input, Messages: req.Messages}
	if _, _, err := a.Assemble(context.Background(), req, modelReq); err != nil {
		t.Fatalf("first assemble failed: %v", err)
	}
	if _, _, err := a.Assemble(context.Background(), req, modelReq); err != nil {
		t.Fatalf("second assemble failed: %v", err)
	}
	raw, err := os.ReadFile(cfg.CA3.Spill.Path)
	if err != nil {
		t.Fatalf("read spill file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	seen := map[string]struct{}{}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			t.Fatalf("duplicate spill line found, expected idempotent spill writes: %s", line)
		}
		seen[line] = struct{}{}
	}
}
