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

func TestDiscoverStage2ReferencesDeduplicatesAndRespectsMaxRefs(t *testing.T) {
	discovery, catalog := discoverStage2References(
		[]string{"alpha", "beta", "alpha", "gamma"},
		"file",
		2,
	)
	if len(discovery.References) != 2 {
		t.Fatalf("references len = %d, want 2", len(discovery.References))
	}
	if discovery.Deduplicated != 1 {
		t.Fatalf("deduplicated = %d, want 1", discovery.Deduplicated)
	}
	if discovery.MaxRefsApplied != 2 {
		t.Fatalf("max_refs_applied = %d, want 2", discovery.MaxRefsApplied)
	}
	if len(catalog) != 3 {
		t.Fatalf("catalog len = %d, want 3 unique locators", len(catalog))
	}
}

func TestResolveSelectedStage2ReferencesRejectsMissingFields(t *testing.T) {
	_, err := resolveSelectedStage2References(
		[]types.ContextReference{{
			ID:      "",
			Type:    "stage2_chunk",
			Locator: "stage2://file/abc",
		}},
		map[string]string{"stage2://file/abc": "ctx"},
		64,
		contextRefMissingPolicyFailFast,
	)
	if err == nil || !strings.Contains(err.Error(), "reference.id is required") {
		t.Fatalf("expected reference.id validation error, got %v", err)
	}
}

func TestResolveSelectedStage2ReferencesRejectsInvalidLocator(t *testing.T) {
	_, err := resolveSelectedStage2References(
		[]types.ContextReference{{
			ID:      "ref-1",
			Type:    "stage2_chunk",
			Locator: "file://abc",
		}},
		map[string]string{"file://abc": "ctx"},
		64,
		contextRefMissingPolicyFailFast,
	)
	if err == nil || !strings.Contains(err.Error(), "reference.locator must use stage2:// scheme") {
		t.Fatalf("expected invalid locator validation error, got %v", err)
	}
}

func TestResolveSelectedStage2ReferencesBudgetOverflow(t *testing.T) {
	discovery, catalog := discoverStage2References(
		[]string{
			strings.Repeat("a", 120),
			strings.Repeat("b", 120),
		},
		"file",
		8,
	)
	result, err := resolveSelectedStage2References(
		discovery.References,
		catalog,
		40,
		contextRefMissingPolicySkipAndRecord,
	)
	if err != nil {
		t.Fatalf("resolveSelectedStage2References failed: %v", err)
	}
	if !result.Truncated {
		t.Fatal("truncated = false, want true")
	}
	if len(result.Resolved) != 1 {
		t.Fatalf("resolved len = %d, want 1 within budget", len(result.Resolved))
	}
	if result.BudgetUsedTokens <= 0 || result.BudgetUsedTokens > 40 {
		t.Fatalf("budget_used_tokens = %d, want in (0,40]", result.BudgetUsedTokens)
	}
}

func TestResolveSelectedStage2ReferencesMissingPolicy(t *testing.T) {
	ref := types.ContextReference{
		ID:      "ref-missing",
		Type:    "stage2_chunk",
		Locator: "stage2://file/missing",
	}
	result, err := resolveSelectedStage2References(
		[]types.ContextReference{ref},
		map[string]string{},
		64,
		contextRefMissingPolicySkipAndRecord,
	)
	if err != nil {
		t.Fatalf("skip_and_record policy should not error, got %v", err)
	}
	if len(result.Missing) != 1 {
		t.Fatalf("missing len = %d, want 1", len(result.Missing))
	}

	_, err = resolveSelectedStage2References(
		[]types.ContextReference{ref},
		map[string]string{},
		64,
		contextRefMissingPolicyFailFast,
	)
	if err == nil || !strings.Contains(err.Error(), "selected reference locator not found") {
		t.Fatalf("expected fail_fast missing locator error, got %v", err)
	}
}

func TestAssemblerCA2ReferenceFirstInjectsRefsBeforeBody(t *testing.T) {
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

	contextCfg := runtimeconfig.DefaultConfig().Runtime.Context
	contextCfg.JIT.ReferenceFirst.Enabled = true
	contextCfg.JIT.ReferenceFirst.MaxRefs = 8
	contextCfg.JIT.ReferenceFirst.MaxResolveTokens = 2048

	a := New(
		func() runtimeconfig.ContextAssemblerConfig { return cfg },
		WithRuntimeContextConfigProvider(func() runtimeconfig.RuntimeContextConfig {
			return contextCfg
		}),
	)

	outReq, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-ref-first-1",
		SessionID:     "session-1",
		PrefixVersion: "ca1",
		Input:         "lookup",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-ref-first-1",
		Input:    "lookup",
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Stage.Status != types.AssembleStageStatusStage2Used {
		t.Fatalf("stage status = %q, want stage2_used", result.Stage.Status)
	}
	if result.Stage.ContextRefDiscoverCount == 0 {
		t.Fatal("context_ref_discover_count = 0, want > 0")
	}
	if result.Stage.ContextRefResolveCount == 0 {
		t.Fatal("context_ref_resolve_count = 0, want > 0")
	}

	refsIdx := -1
	ctxIdx := -1
	for i := range outReq.Messages {
		content := outReq.Messages[i].Content
		if strings.HasPrefix(content, "stage2_refs:") {
			refsIdx = i
		}
		if strings.HasPrefix(content, "stage2_context:") {
			ctxIdx = i
		}
	}
	if refsIdx == -1 || ctxIdx == -1 {
		t.Fatalf("missing stage2 refs/context messages: %#v", outReq.Messages)
	}
	if refsIdx > ctxIdx {
		t.Fatalf("reference metadata must be injected before full-body context, refs_idx=%d ctx_idx=%d", refsIdx, ctxIdx)
	}
}
