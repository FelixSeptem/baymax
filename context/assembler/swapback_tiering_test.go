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
	cfg := stageThreeConfigSnapshotForTest(t)
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

	appended, relevance, _, _, err := a.swapBackIfNeeded(context.Background(), req, modelReq, cfg, runtimeCtx, state)
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
	appended, relevance, _, _, err = a.swapBackIfNeeded(context.Background(), req, modelReq, cfg, runtimeCtx, state)
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

func TestSwapBackIfNeededRelevanceThenRecencyDeterministicOrder(t *testing.T) {
	a := New(func() runtimeconfig.ContextAssemblerConfig {
		return runtimeconfig.DefaultConfig().ContextAssembler
	})
	cfg := stageThreeConfigSnapshotForTest(t)
	cfg.Spill.Enabled = true
	cfg.Spill.Backend = "file"
	cfg.Spill.SwapBackLimit = 2
	cfg.Spill.Path = filepath.Join(t.TempDir(), "spill.jsonl")

	now := time.Now().UTC()
	writeSpillRecordsForTest(t, cfg.Spill.Path, []spillRecord{
		{
			RunID:        "run-swapback-rank",
			OriginRef:    "ref-high-old",
			Content:      "invoice payment evidence old",
			EvidenceTags: []string{"invoice", "payment"},
			SpilledAt:    now.Add(-4 * time.Minute),
		},
		{
			RunID:        "run-swapback-rank",
			OriginRef:    "ref-high-new",
			Content:      "invoice payment evidence new",
			EvidenceTags: []string{"invoice", "payment"},
			SpilledAt:    now.Add(-1 * time.Minute),
		},
		{
			RunID:        "run-swapback-rank",
			OriginRef:    "ref-mid-newest",
			Content:      "invoice only clue newest",
			EvidenceTags: []string{"invoice"},
			SpilledAt:    now.Add(-15 * time.Second),
		},
	})
	state := a.pressureStateFor("run-swapback-rank")
	req := types.ContextAssembleRequest{
		RunID: "run-swapback-rank",
		Input: "invoice payment update",
	}
	modelReq := &types.ModelRequest{RunID: req.RunID, Input: req.Input}
	runtimeCtx := runtimeconfig.DefaultConfig().Runtime.Context
	runtimeCtx.JIT.SwapBack.Enabled = true
	runtimeCtx.JIT.SwapBack.MinRelevanceScore = 0
	runtimeCtx.JIT.SwapBack.CandidateWindow = 3
	runtimeCtx.JIT.SwapBack.RankingStrategy = runtimeconfig.RuntimeContextJITSwapBackRankingStrategyRelevanceThenRecency

	appended, _, _, _, err := a.swapBackIfNeeded(context.Background(), req, modelReq, cfg, runtimeCtx, state)
	if err != nil {
		t.Fatalf("swapBackIfNeeded failed: %v", err)
	}
	if appended != 2 {
		t.Fatalf("appended=%d, want 2", appended)
	}
	joined := joinMessageContents(modelReq.Messages)
	first := strings.Index(joined, "invoice payment evidence new")
	second := strings.Index(joined, "invoice payment evidence old")
	if first < 0 || second < 0 || first > second {
		t.Fatalf("expected deterministic relevance+recency order new->old, got messages=%#v", modelReq.Messages)
	}
	if strings.Contains(joined, "invoice only clue newest") {
		t.Fatalf("mid relevance candidate should be excluded when swap_back_limit=2, got messages=%#v", modelReq.Messages)
	}
}

func TestSwapBackIfNeededCandidateWindowLimitsSelection(t *testing.T) {
	a := New(func() runtimeconfig.ContextAssemblerConfig {
		return runtimeconfig.DefaultConfig().ContextAssembler
	})
	cfg := stageThreeConfigSnapshotForTest(t)
	cfg.Spill.Enabled = true
	cfg.Spill.Backend = "file"
	cfg.Spill.SwapBackLimit = 3
	cfg.Spill.Path = filepath.Join(t.TempDir(), "spill.jsonl")

	now := time.Now().UTC()
	writeSpillRecordsForTest(t, cfg.Spill.Path, []spillRecord{
		{
			RunID:        "run-swapback-window",
			OriginRef:    "ref-oldest",
			Content:      "invoice payment oldest",
			EvidenceTags: []string{"invoice", "payment"},
			SpilledAt:    now.Add(-4 * time.Minute),
		},
		{
			RunID:        "run-swapback-window",
			OriginRef:    "ref-middle",
			Content:      "invoice payment middle",
			EvidenceTags: []string{"invoice", "payment"},
			SpilledAt:    now.Add(-2 * time.Minute),
		},
		{
			RunID:        "run-swapback-window",
			OriginRef:    "ref-newest",
			Content:      "invoice payment newest",
			EvidenceTags: []string{"invoice", "payment"},
			SpilledAt:    now.Add(-20 * time.Second),
		},
	})
	state := a.pressureStateFor("run-swapback-window")
	req := types.ContextAssembleRequest{
		RunID: "run-swapback-window",
		Input: "invoice payment update",
	}
	modelReq := &types.ModelRequest{RunID: req.RunID, Input: req.Input}
	runtimeCtx := runtimeconfig.DefaultConfig().Runtime.Context
	runtimeCtx.JIT.SwapBack.Enabled = true
	runtimeCtx.JIT.SwapBack.MinRelevanceScore = 0
	runtimeCtx.JIT.SwapBack.CandidateWindow = 1
	runtimeCtx.JIT.SwapBack.RankingStrategy = runtimeconfig.RuntimeContextJITSwapBackRankingStrategyRecencyOnly

	appended, _, _, _, err := a.swapBackIfNeeded(context.Background(), req, modelReq, cfg, runtimeCtx, state)
	if err != nil {
		t.Fatalf("swapBackIfNeeded failed: %v", err)
	}
	if appended != 1 {
		t.Fatalf("appended=%d, want 1 due candidate_window=1", appended)
	}
	joined := joinMessageContents(modelReq.Messages)
	if !strings.Contains(joined, "invoice payment newest") {
		t.Fatalf("expected newest candidate under recency-only+window=1, got=%#v", modelReq.Messages)
	}
	if strings.Contains(joined, "invoice payment middle") || strings.Contains(joined, "invoice payment oldest") {
		t.Fatalf("older candidates should be excluded by candidate window, got=%#v", modelReq.Messages)
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

func TestApplyLifecycleTieringConflictBoundaryPrefersCanonicalTierOrder(t *testing.T) {
	now := time.Now().UTC()
	state := &pressureRunState{
		SpilledByRun: map[string]spillRecord{
			"boundary-ref": {
				OriginRef: "boundary-ref",
				Content:   "boundary-content",
				SpilledAt: now.Add(-1000 * time.Millisecond),
			},
		},
		SpillTierByRef: map[string]string{
			"boundary-ref": "cold",
		},
	}
	stats, action := applyLifecycleTiering(state, now, runtimeconfig.RuntimeContextJITLifecycleTieringConfig{
		Enabled:   true,
		HotTTLMS:  1000,
		WarmTTLMS: 1000,
		ColdTTLMS: 1000,
	})
	if stats["hot"] != 1 {
		t.Fatalf("boundary age should choose hot tier by canonical precedence, stats=%#v", stats)
	}
	if stats["migrate_cold_to_hot"] != 1 {
		t.Fatalf("boundary migration should be deterministic cold->hot, stats=%#v", stats)
	}
	if strings.TrimSpace(action) != "" {
		t.Fatalf("hot boundary transition should not emit spill/compress/prune action, got=%q", action)
	}
	if state.SpillTierByRef["boundary-ref"] != "hot" {
		t.Fatalf("spill tier should settle to hot at boundary, got=%q", state.SpillTierByRef["boundary-ref"])
	}
}

func TestAssemblerContextPressureSwapBackAndTieringCombination(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	stageTwo := stageTwoConfigPointerForTest(t, &cfg)
	stageThree := stageThreeConfigPointerForTest(t, &cfg)
	stageTwo.Enabled = false
	stageThree.Enabled = true
	stageThree.Spill.Enabled = true
	stageThree.Spill.Backend = "file"
	stageThree.Spill.SwapBackLimit = 8
	stageThree.Spill.Path = filepath.Join(t.TempDir(), "spill.jsonl")
	stageThree.MaxContextTokens = 4096

	now := time.Now().UTC()
	writeSpillRecordsForTest(t, stageThree.Spill.Path, []spillRecord{
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
