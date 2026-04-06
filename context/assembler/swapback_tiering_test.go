package assembler

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestSwapBackIfNeededUsesRelevanceThreshold(t *testing.T) {
	a := New(func() runtimeconfig.ContextAssemblerConfig {
		return runtimeconfig.DefaultConfig().ContextAssembler
	})
	cfg := runtimeconfig.DefaultConfig().ContextAssembler.CA3
	cfg.Spill.Enabled = true
	cfg.Spill.Backend = "file"
	cfg.Spill.SwapBackLimit = 8
	cfg.Spill.Path = filepath.Join(t.TempDir(), "spill.jsonl")

	now := time.Now().UTC()
	writeSpillRecordsForTest(t, cfg.Spill.Path, []spillRecord{
		{
			RunID:        "run-swapback-relevance",
			OriginRef:    "ref-1",
			Content:      "invoice payment status",
			EvidenceTags: []string{"invoice", "payment"},
			SpilledAt:    now,
		},
	})
	state := a.pressureStateFor("run-swapback-relevance")
	req := types.ContextAssembleRequest{
		RunID: "run-swapback-relevance",
		Input: "weather summary",
	}
	modelReq := &types.ModelRequest{RunID: req.RunID, Input: req.Input}
	runtimeCtx := runtimeconfig.DefaultConfig().Runtime.Context
	runtimeCtx.JIT.SwapBack.Enabled = true
	runtimeCtx.JIT.SwapBack.MinRelevanceScore = 0.6

	appended, relevance, err := a.swapBackIfNeeded(context.Background(), req, modelReq, cfg, runtimeCtx, state)
	if err != nil {
		t.Fatalf("swapBackIfNeeded failed: %v", err)
	}
	if appended != 0 {
		t.Fatalf("low-relevance query should not swap back context, appended=%d", appended)
	}
	if relevance != 0 {
		t.Fatalf("low-relevance score should be 0, got %f", relevance)
	}

	req.Input = "invoice payment update"
	modelReq = &types.ModelRequest{RunID: req.RunID, Input: req.Input}
	appended, relevance, err = a.swapBackIfNeeded(context.Background(), req, modelReq, cfg, runtimeCtx, state)
	if err != nil {
		t.Fatalf("swapBackIfNeeded failed: %v", err)
	}
	if appended != 1 {
		t.Fatalf("relevant query should swap back exactly one record, appended=%d", appended)
	}
	if relevance < 0.6 {
		t.Fatalf("relevance score=%f, want >= 0.6", relevance)
	}
	if countMessagePrefix(modelReq.Messages, "swap_back_context:") != 1 {
		t.Fatalf("swap_back_context message missing after relevance hit: %#v", modelReq.Messages)
	}
}

func TestApplyLifecycleTieringTransitionsAndPrune(t *testing.T) {
	now := time.Now().UTC()
	state := &pressureRunState{
		SpilledByRun: map[string]spillRecord{
			"hot-ref": {
				OriginRef: "hot-ref",
				Content:   "hot-content",
				SpilledAt: now.Add(-500 * time.Millisecond),
			},
			"warm-ref": {
				OriginRef: "warm-ref",
				Content:   strings.Repeat("warm-content ", 40),
				SpilledAt: now.Add(-1500 * time.Millisecond),
			},
			"cold-ref": {
				OriginRef: "cold-ref",
				Content:   "cold-content",
				SpilledAt: now.Add(-2500 * time.Millisecond),
			},
			"expired-ref": {
				OriginRef: "expired-ref",
				Content:   "expired-content",
				SpilledAt: now.Add(-4500 * time.Millisecond),
			},
		},
		SpillTierByRef: map[string]string{
			"hot-ref":     "hot",
			"warm-ref":    "hot",
			"cold-ref":    "warm",
			"expired-ref": "cold",
		},
	}
	stats, action := applyLifecycleTiering(state, now, runtimeconfig.RuntimeContextJITLifecycleTieringConfig{
		Enabled:   true,
		HotTTLMS:  1000,
		WarmTTLMS: 2000,
		ColdTTLMS: 3000,
	})
	if stats["hot"] != 1 || stats["warm"] != 1 || stats["cold"] != 1 || stats["pruned"] != 1 {
		t.Fatalf("unexpected tier stats: %#v", stats)
	}
	if stats["migrate_hot_to_warm"] != 1 || stats["migrate_warm_to_cold"] != 1 {
		t.Fatalf("expected migration counters, got %#v", stats)
	}
	if action != "prune" {
		t.Fatalf("lifecycle action=%q, want prune due expired tier cleanup", action)
	}
	if _, ok := state.SpilledByRun["expired-ref"]; ok {
		t.Fatalf("expired tier should be pruned from in-memory spill state: %#v", state.SpilledByRun)
	}
	if len([]rune(state.SpilledByRun["warm-ref"].Content)) > 256 {
		t.Fatalf("warm tier content should be compressed, got len=%d", len([]rune(state.SpilledByRun["warm-ref"].Content)))
	}
}

func TestAssemblerContextPressureSwapBackAndTieringCombination(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	cfg.CA2.Enabled = false
	cfg.CA3.Enabled = true
	cfg.CA3.Spill.Enabled = true
	cfg.CA3.Spill.Backend = "file"
	cfg.CA3.Spill.SwapBackLimit = 8
	cfg.CA3.Spill.Path = filepath.Join(t.TempDir(), "spill.jsonl")
	cfg.CA3.MaxContextTokens = 4096

	now := time.Now().UTC()
	writeSpillRecordsForTest(t, cfg.CA3.Spill.Path, []spillRecord{
		{
			RunID:        "run-tier-combo",
			OriginRef:    "cold-relevant",
			Content:      "invoice payment memo",
			EvidenceTags: []string{"invoice", "payment"},
			SpilledAt:    now.Add(-2500 * time.Millisecond),
		},
		{
			RunID:        "run-tier-combo",
			OriginRef:    "expired-relevant",
			Content:      "invoice payment expired",
			EvidenceTags: []string{"invoice", "payment"},
			SpilledAt:    now.Add(-6000 * time.Millisecond),
		},
	})

	runtimeCtx := runtimeconfig.DefaultConfig().Runtime.Context
	runtimeCtx.JIT.SwapBack.Enabled = true
	runtimeCtx.JIT.SwapBack.MinRelevanceScore = 0.5
	runtimeCtx.JIT.LifecycleTiering.Enabled = true
	runtimeCtx.JIT.LifecycleTiering.HotTTLMS = 1000
	runtimeCtx.JIT.LifecycleTiering.WarmTTLMS = 2000
	runtimeCtx.JIT.LifecycleTiering.ColdTTLMS = 3000

	a := New(
		func() runtimeconfig.ContextAssemblerConfig { return cfg },
		WithRuntimeContextConfigProvider(func() runtimeconfig.RuntimeContextConfig {
			return runtimeCtx
		}),
	)
	a.now = func() time.Time { return now }
	state := a.pressureStateFor("run-tier-combo")
	state.SpilledByRun["expired-in-memory"] = spillRecord{
		RunID:        "run-tier-combo",
		OriginRef:    "expired-in-memory",
		Content:      "invoice payment expired-memory",
		EvidenceTags: []string{"invoice", "payment"},
		SpilledAt:    now.Add(-7000 * time.Millisecond),
	}
	state.SpillTierByRef["expired-in-memory"] = "cold"

	outReq, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-tier-combo",
		SessionID:     "session-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         "invoice payment status",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-tier-combo",
		Input:    "invoice payment status",
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if countMessagePrefix(outReq.Messages, "swap_back_context:") != 1 {
		t.Fatalf("only cold relevant context should swap back, got messages=%#v", outReq.Messages)
	}
	if !strings.Contains(joinMessageContents(outReq.Messages), "invoice payment memo") {
		t.Fatalf("missing cold-tier swap back content: %#v", outReq.Messages)
	}
	if strings.Contains(joinMessageContents(outReq.Messages), "invoice payment expired") {
		t.Fatalf("expired tier content should not swap back: %#v", outReq.Messages)
	}
	if result.Stage.ContextSwapbackRelevanceScore <= 0 {
		t.Fatalf("swapback relevance score should be recorded, got %#v", result.Stage)
	}
	if result.Stage.ContextLifecycleTierStats["cold"] <= 0 || result.Stage.ContextLifecycleTierStats["pruned"] <= 0 {
		t.Fatalf("tier stats should include cold and pruned counts, got %#v", result.Stage.ContextLifecycleTierStats)
	}
}

func writeSpillRecordsForTest(t *testing.T, path string, records []spillRecord) {
	t.Helper()
	lines := make([]string, 0, len(records))
	for i := range records {
		raw, err := json.Marshal(records[i])
		if err != nil {
			t.Fatalf("marshal spill record[%d]: %v", i, err)
		}
		lines = append(lines, string(raw))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		t.Fatalf("write spill fixture: %v", err)
	}
}
