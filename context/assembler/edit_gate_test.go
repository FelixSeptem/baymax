package assembler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestApplyContextEditGateAllowThresholdMet(t *testing.T) {
	cfg := runtimeconfig.RuntimeContextJITEditGateConfig{
		Enabled:            true,
		ClearAtLeastTokens: 1,
		MinGainRatio:       0.2,
	}
	result := applyContextEditGate(
		[]string{"alpha", "alpha", "beta"},
		cfg,
	)
	if result.Decision != contextEditGateDecisionAllow {
		t.Fatalf("decision = %q, want %q", result.Decision, contextEditGateDecisionAllow)
	}
	if result.EstimatedSavedTokens <= 0 {
		t.Fatalf("estimated_saved_tokens = %d, want > 0", result.EstimatedSavedTokens)
	}
	if len(result.Chunks) != 2 {
		t.Fatalf("chunks len after gate = %d, want 2", len(result.Chunks))
	}
}

func TestApplyContextEditGateNegativeDecisions(t *testing.T) {
	t.Run("threshold_too_high", func(t *testing.T) {
		cfg := runtimeconfig.RuntimeContextJITEditGateConfig{
			Enabled:            true,
			ClearAtLeastTokens: 999,
			MinGainRatio:       0.1,
		}
		result := applyContextEditGate([]string{"a", "a"}, cfg)
		if result.Decision != contextEditGateDecisionDenySavedTokens {
			t.Fatalf("decision = %q, want %q", result.Decision, contextEditGateDecisionDenySavedTokens)
		}
		if len(result.Chunks) != 2 {
			t.Fatalf("gate deny should preserve chunks, got len=%d", len(result.Chunks))
		}
	})

	t.Run("gain_ratio_insufficient", func(t *testing.T) {
		cfg := runtimeconfig.RuntimeContextJITEditGateConfig{
			Enabled:            true,
			ClearAtLeastTokens: 1,
			MinGainRatio:       0.8,
		}
		result := applyContextEditGate(
			[]string{
				strings.Repeat("long-", 80),
				strings.Repeat("long-", 80),
				strings.Repeat("other-", 80),
			},
			cfg,
		)
		if result.Decision != contextEditGateDecisionDenyGainRatio {
			t.Fatalf("decision = %q, want %q", result.Decision, contextEditGateDecisionDenyGainRatio)
		}
		if len(result.Chunks) != 3 {
			t.Fatalf("gate deny should preserve chunks, got len=%d", len(result.Chunks))
		}
	})

	t.Run("config_conflict", func(t *testing.T) {
		cfg := runtimeconfig.RuntimeContextJITEditGateConfig{
			Enabled:            true,
			ClearAtLeastTokens: 0,
			MinGainRatio:       0,
		}
		result := applyContextEditGate([]string{"a", "a"}, cfg)
		if result.Decision != contextEditGateDecisionDenyConfig {
			t.Fatalf("decision = %q, want %q", result.Decision, contextEditGateDecisionDenyConfig)
		}
		if len(result.Chunks) != 2 {
			t.Fatalf("gate deny should preserve chunks, got len=%d", len(result.Chunks))
		}
	})
}

func TestAssemblerCA2EditGateDenyKeepsSemantics(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = true
	cfg.CA2.Stage2.Provider = runtimeconfig.ContextStage2ProviderFile
	stage2File := filepath.Join(t.TempDir(), "stage2.jsonl")
	content := strings.Join([]string{
		`{"session_id":"session-1","content":"dup-context"}`,
		`{"session_id":"session-1","content":"dup-context"}`,
		`{"session_id":"session-1","content":"keep-context"}`,
	}, "\n")
	if err := os.WriteFile(stage2File, []byte(content), 0o600); err != nil {
		t.Fatalf("write stage2 file: %v", err)
	}
	cfg.CA2.Stage2.FilePath = stage2File
	cfg.CA2.Routing.MinInputChars = 1

	contextCfg := runtimeconfig.DefaultConfig().Runtime.Context
	contextCfg.JIT.EditGate.Enabled = true
	contextCfg.JIT.EditGate.ClearAtLeastTokens = 999
	contextCfg.JIT.EditGate.MinGainRatio = 0.1
	contextCfg.JIT.ReferenceFirst.Enabled = false

	a := New(
		func() runtimeconfig.ContextAssemblerConfig { return cfg },
		WithRuntimeContextConfigProvider(func() runtimeconfig.RuntimeContextConfig {
			return contextCfg
		}),
	)
	outReq, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-edit-gate-deny",
		SessionID:     "session-1",
		PrefixVersion: "ca1",
		Input:         "lookup context",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-edit-gate-deny",
		Input:    "lookup context",
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Stage.ContextEditGateDecision != contextEditGateDecisionDenySavedTokens {
		t.Fatalf("edit gate decision = %q, want %q", result.Stage.ContextEditGateDecision, contextEditGateDecisionDenySavedTokens)
	}
	if result.Stage.ContextEditEstimatedSavedTokens <= 0 {
		t.Fatalf("estimated saved tokens = %d, want > 0", result.Stage.ContextEditEstimatedSavedTokens)
	}

	stage2Context := ""
	for _, msg := range outReq.Messages {
		if strings.HasPrefix(msg.Content, "stage2_context:\n") {
			stage2Context = msg.Content
			break
		}
	}
	if stage2Context == "" {
		t.Fatalf("missing stage2_context message: %#v", outReq.Messages)
	}
	if countSubstring(stage2Context, "dup-context") != 2 {
		t.Fatalf("edit gate denied path must keep duplicate semantics, got: %q", stage2Context)
	}
}

func countSubstring(s string, sub string) int {
	if sub == "" {
		return 0
	}
	count := 0
	remain := s
	for {
		idx := strings.Index(remain, sub)
		if idx < 0 {
			return count
		}
		count++
		remain = remain[idx+len(sub):]
	}
}
