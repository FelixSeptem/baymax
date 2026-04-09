package assembler

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestAssemblerContextPressureRuntimeFallbackPolicyFailFastOverridesBestEffort(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	stageThree := stageThreeConfigPointerForTest(t, &cfg)
	stageTwo := stageTwoConfigPointerForTest(t, &cfg)
	stageThree.Enabled = true
	stageThree.MaxContextTokens = 120
	stageThree.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	stageThree.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	stageThree.Compaction.Mode = "semantic"
	stageTwo.StagePolicy.Stage1 = "best_effort"

	runtimeCtx := runtimeconfig.DefaultConfig().Runtime.Context
	runtimeCtx.JIT.Compaction.FallbackPolicy = runtimeconfig.RuntimeContextJITCompactionFallbackPolicyFailFast

	client := modelClientFunc{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{}, errors.New("semantic unavailable")
		},
	}
	a := New(
		func() runtimeconfig.ContextAssemblerConfig { return cfg },
		WithRuntimeContextConfigProvider(func() runtimeconfig.RuntimeContextConfig {
			return runtimeCtx
		}),
	)
	msgs := []types.Message{
		{Role: "system", Content: "base"},
		{Role: "user", Content: strings.Repeat("long semantic content ", 24)},
	}
	_, _, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-semantic-runtime-fail-fast",
		SessionID:     "s-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         strings.Repeat("need compact ", 18),
		Messages:      msgs,
		ModelClient:   client,
	}, types.ModelRequest{
		RunID:    "run-semantic-runtime-fail-fast",
		Input:    strings.Repeat("need compact ", 18),
		Messages: msgs,
	})
	if err == nil {
		t.Fatal("runtime fallback policy fail_fast should abort semantic compaction")
	}
}

func TestAssemblerContextPressureRuntimeQualityThresholdOverridesCA3(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	stageThree := stageThreeConfigPointerForTest(t, &cfg)
	stageTwo := stageTwoConfigPointerForTest(t, &cfg)
	stageThree.Enabled = true
	stageThree.MaxContextTokens = 120
	stageThree.PercentThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	stageThree.AbsoluteThresholds = runtimeconfig.ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	stageThree.Compaction.Mode = "semantic"
	stageThree.Compaction.Quality.Threshold = 0.10
	stageThree.Compaction.Evidence.Keywords = []string{"mustkeep"}
	stageTwo.StagePolicy.Stage1 = "best_effort"

	runtimeCtx := runtimeconfig.DefaultConfig().Runtime.Context
	runtimeCtx.JIT.Compaction.QualityThreshold = 0.95
	runtimeCtx.JIT.Compaction.FallbackPolicy = runtimeconfig.RuntimeContextJITCompactionFallbackPolicyBestEffort

	client := modelClientFunc{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "summary dropped keyword"}, nil
		},
	}
	a := New(
		func() runtimeconfig.ContextAssemblerConfig { return cfg },
		WithRuntimeContextConfigProvider(func() runtimeconfig.RuntimeContextConfig {
			return runtimeCtx
		}),
	)
	msgs := []types.Message{
		{Role: "system", Content: "base"},
		{Role: "user", Content: strings.Repeat("mustkeep long semantic content ", 24)},
	}
	_, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-semantic-runtime-threshold",
		SessionID:     "s-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         strings.Repeat("need compact ", 18),
		Messages:      msgs,
		ModelClient:   client,
	}, types.ModelRequest{
		RunID:    "run-semantic-runtime-threshold",
		Input:    strings.Repeat("need compact ", 18),
		Messages: msgs,
	})
	if err != nil {
		t.Fatalf("Assemble should fallback under best_effort, got: %v", err)
	}
	if !result.Stage.CompactionFallback {
		t.Fatal("compaction fallback should be true")
	}
	if result.Stage.CompactionFallbackReason != "quality_below_threshold" {
		t.Fatalf("fallback reason=%q, want quality_below_threshold", result.Stage.CompactionFallbackReason)
	}
	if result.Stage.CompactionOutcomeClass != "degraded" {
		t.Fatalf("compaction outcome class=%q, want degraded", result.Stage.CompactionOutcomeClass)
	}
}

func TestSelectPruneCandidateRespectsOldestToolResultEligibility(t *testing.T) {
	cfg := stageThreeConfigSnapshotForTest(t)
	state := &pressureRunState{AccessFrequency: map[string]int{}}
	messages := []types.Message{
		{Role: "system", Content: "base"},
		{Role: "tool", Content: "tool_result: oldest"},
		{Role: "user", Content: "filler-alpha"},
		{Role: "user", Content: "filler-beta"},
	}

	idx := selectPruneCandidate(
		messages,
		cfg,
		state,
		cfg.Compaction.Evidence,
		runtimeconfig.RuntimeContextJITCompactionRuleEligibility{
			AllowOldestToolResult: false,
			MinRetainedEvidence:   0,
		},
		0,
	)
	if idx != 2 {
		t.Fatalf("oldest tool result must be protected when disabled, got idx=%d want=2", idx)
	}

	idx = selectPruneCandidate(
		messages,
		cfg,
		state,
		cfg.Compaction.Evidence,
		runtimeconfig.RuntimeContextJITCompactionRuleEligibility{
			AllowOldestToolResult: true,
			MinRetainedEvidence:   0,
		},
		0,
	)
	if idx != 1 {
		t.Fatalf("oldest tool result should be eligible when enabled, got idx=%d want=1", idx)
	}
}

func TestSelectPruneCandidateRespectsMinimumRetainedEvidence(t *testing.T) {
	cfg := stageThreeConfigSnapshotForTest(t)
	state := &pressureRunState{AccessFrequency: map[string]int{}}
	evidence := cfg.Compaction.Evidence
	evidence.Keywords = []string{"mustkeep"}

	messages := []types.Message{
		{Role: "system", Content: "base"},
		{Role: "user", Content: "mustkeep evidence one"},
		{Role: "user", Content: "mustkeep evidence two"},
	}

	idx := selectPruneCandidate(
		messages,
		cfg,
		state,
		evidence,
		runtimeconfig.RuntimeContextJITCompactionRuleEligibility{
			AllowOldestToolResult: true,
			MinRetainedEvidence:   2,
		},
		2,
	)
	if idx != -1 {
		t.Fatalf("no evidence should be pruned at min retained boundary, got idx=%d", idx)
	}

	idx = selectPruneCandidate(
		messages,
		cfg,
		state,
		evidence,
		runtimeconfig.RuntimeContextJITCompactionRuleEligibility{
			AllowOldestToolResult: true,
			MinRetainedEvidence:   1,
		},
		2,
	)
	if idx != 1 {
		t.Fatalf("evidence may be pruned when above min retained boundary, got idx=%d want=1", idx)
	}
}

func TestFileSpillBackendLoadByRunMalformedCleanupAndRecoveryMarker(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "spill.jsonl")

	rec := spillRecord{
		RunID:     "run-load-cleanup",
		OriginRef: "ref-1",
		Content:   "valid payload",
		SpilledAt: time.Now().UTC(),
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	lines := []string{
		string(raw),
		"{malformed-json",
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	backend := newFileSpillBackendWithOptions(path, fileSpillBackendOptions{
		ReuseHandle: false,
		ColdStore: runtimeconfig.RuntimeContextJITColdStoreConfig{
			Cleanup: runtimeconfig.RuntimeContextJITColdStoreCleanupConfig{
				Enabled:   true,
				BatchSize: 8,
			},
		},
	})
	records, err := backend.LoadByRun(context.Background(), "run-load-cleanup", 8)
	if err != nil {
		t.Fatalf("LoadByRun failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("LoadByRun records=%d, want 1", len(records))
	}
	if got := backend.LastGovernanceAction(); got != "cleanup_applied" {
		t.Fatalf("LastGovernanceAction=%q, want cleanup_applied", got)
	}
	if got := backend.LastRecoveryMarker(); got != "cold_store_recovered_malformed_lines" {
		t.Fatalf("LastRecoveryMarker=%q, want cold_store_recovered_malformed_lines", got)
	}
	fileRaw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read rewritten spill file: %v", err)
	}
	if strings.Contains(string(fileRaw), "{malformed-json") {
		t.Fatalf("malformed line should be cleaned up, got file=%q", string(fileRaw))
	}
}

func TestFileSpillBackendAppendBatchAppliesRetentionMaxRecords(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "spill.jsonl")
	backend := newFileSpillBackendWithOptions(path, fileSpillBackendOptions{
		ReuseHandle: false,
		ColdStore: runtimeconfig.RuntimeContextJITColdStoreConfig{
			Retention: runtimeconfig.RuntimeContextJITColdStoreRetentionConfig{
				MaxAgeMS:   int((24 * time.Hour) / time.Millisecond),
				MaxRecords: 2,
			},
			Quota: runtimeconfig.RuntimeContextJITColdStoreQuotaConfig{
				MaxBytes: 1024 * 1024,
			},
		},
	})

	records := []spillRecord{
		{RunID: "run-retention", OriginRef: "ref-1", Content: "one", SpilledAt: time.Now().UTC().Add(-3 * time.Minute)},
		{RunID: "run-retention", OriginRef: "ref-2", Content: "two", SpilledAt: time.Now().UTC().Add(-2 * time.Minute)},
		{RunID: "run-retention", OriginRef: "ref-3", Content: "three", SpilledAt: time.Now().UTC().Add(-1 * time.Minute)},
	}
	if err := backend.AppendBatch(context.Background(), records); err != nil {
		t.Fatalf("AppendBatch failed: %v", err)
	}
	if got := backend.LastGovernanceAction(); got != "retention_applied" {
		t.Fatalf("LastGovernanceAction=%q, want retention_applied", got)
	}

	out, err := backend.LoadByRun(context.Background(), "run-retention", 10)
	if err != nil {
		t.Fatalf("LoadByRun failed: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("retention max_records should keep 2 newest records, got=%d", len(out))
	}
	if out[0].OriginRef != "ref-2" || out[1].OriginRef != "ref-3" {
		t.Fatalf("retention should keep newest records deterministically, got=%#v", out)
	}
}

func TestAssemblerSwapBackIdempotentDeduplicatesOnRepeatedAssemble(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	stageTwo := stageTwoConfigPointerForTest(t, &cfg)
	stageThree := stageThreeConfigPointerForTest(t, &cfg)
	stageTwo.Enabled = false
	stageThree.Enabled = true
	stageThree.Spill.Enabled = true
	stageThree.Spill.Backend = "file"
	stageThree.Spill.SwapBackLimit = 4
	stageThree.Spill.Path = filepath.Join(t.TempDir(), "spill.jsonl")
	stageThree.MaxContextTokens = 4096

	now := time.Now().UTC()
	writeSpillRecordsForTest(t, stageThree.Spill.Path, []spillRecord{
		{
			RunID:        "run-swapback-idempotent",
			OriginRef:    "cold-1",
			Content:      "invoice payment memo",
			EvidenceTags: []string{"invoice", "payment"},
			SpilledAt:    now.Add(-2 * time.Second),
		},
	})
	runtimeCtx := runtimeconfig.DefaultConfig().Runtime.Context
	runtimeCtx.JIT.SwapBack.Enabled = true
	runtimeCtx.JIT.SwapBack.MinRelevanceScore = 0.3
	runtimeCtx.JIT.SwapBack.RankingStrategy = runtimeconfig.RuntimeContextJITSwapBackRankingStrategyRelevanceThenRecency
	runtimeCtx.JIT.SwapBack.CandidateWindow = 4

	a := New(
		func() runtimeconfig.ContextAssemblerConfig { return cfg },
		WithRuntimeContextConfigProvider(func() runtimeconfig.RuntimeContextConfig {
			return runtimeCtx
		}),
	)
	a.now = func() time.Time { return now }

	assembleReq := types.ContextAssembleRequest{
		RunID:         "run-swapback-idempotent",
		SessionID:     "session-1",
		PrefixVersion: semanticPrefixVersion,
		Input:         "invoice payment status",
		Messages:      []types.Message{{Role: "system", Content: "base"}},
	}
	modelReq := types.ModelRequest{
		RunID:    assembleReq.RunID,
		Input:    assembleReq.Input,
		Messages: assembleReq.Messages,
	}

	firstReq, firstResult, err := a.Assemble(context.Background(), assembleReq, modelReq)
	if err != nil {
		t.Fatalf("first Assemble failed: %v", err)
	}
	if firstResult.Stage.SwapBackCount != 1 {
		t.Fatalf("first swap_back_count=%d, want 1", firstResult.Stage.SwapBackCount)
	}
	if countMessagePrefix(firstReq.Messages, "swap_back_context:") != 1 {
		t.Fatalf("first assemble should append one swap_back_context message, got %#v", firstReq.Messages)
	}

	secondReq, secondResult, err := a.Assemble(context.Background(), assembleReq, modelReq)
	if err != nil {
		t.Fatalf("second Assemble failed: %v", err)
	}
	if secondResult.Stage.SwapBackCount != 0 {
		t.Fatalf("second swap_back_count=%d, want 0 due idempotent dedup", secondResult.Stage.SwapBackCount)
	}
	if secondResult.Stage.ContextRecoveryConsistencyMarker != "deduplicated" {
		t.Fatalf("second recovery marker=%q, want deduplicated", secondResult.Stage.ContextRecoveryConsistencyMarker)
	}
	if countMessagePrefix(secondReq.Messages, "swap_back_context:") != 0 {
		t.Fatalf("second assemble should not append duplicate swap_back_context message, got %#v", secondReq.Messages)
	}
}

func TestAssemblerSwapBackDeterministicAcrossRestart(t *testing.T) {
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	cfg.JournalPath = filepath.Join(t.TempDir(), "journal.jsonl")
	stageTwo := stageTwoConfigPointerForTest(t, &cfg)
	stageThree := stageThreeConfigPointerForTest(t, &cfg)
	stageTwo.Enabled = false
	stageThree.Enabled = true
	stageThree.Spill.Enabled = true
	stageThree.Spill.Backend = "file"
	stageThree.Spill.SwapBackLimit = 2
	stageThree.Spill.Path = filepath.Join(t.TempDir(), "spill.jsonl")
	stageThree.MaxContextTokens = 4096

	now := time.Now().UTC()
	writeSpillRecordsForTest(t, stageThree.Spill.Path, []spillRecord{
		{
			RunID:        "run-swapback-restart",
			OriginRef:    "cold-older",
			Content:      "invoice payment older",
			EvidenceTags: []string{"invoice", "payment"},
			SpilledAt:    now.Add(-3 * time.Minute),
		},
		{
			RunID:        "run-swapback-restart",
			OriginRef:    "cold-newer",
			Content:      "invoice payment newer",
			EvidenceTags: []string{"invoice", "payment"},
			SpilledAt:    now.Add(-1 * time.Minute),
		},
	})
	runtimeCtx := runtimeconfig.DefaultConfig().Runtime.Context
	runtimeCtx.JIT.SwapBack.Enabled = true
	runtimeCtx.JIT.SwapBack.MinRelevanceScore = 0.3
	runtimeCtx.JIT.SwapBack.RankingStrategy = runtimeconfig.RuntimeContextJITSwapBackRankingStrategyRelevanceThenRecency
	runtimeCtx.JIT.SwapBack.CandidateWindow = 4

	runOnce := func() (types.ModelRequest, types.ContextAssembleResult, error) {
		a := New(
			func() runtimeconfig.ContextAssemblerConfig { return cfg },
			WithRuntimeContextConfigProvider(func() runtimeconfig.RuntimeContextConfig {
				return runtimeCtx
			}),
		)
		a.now = func() time.Time { return now }
		return a.Assemble(context.Background(), types.ContextAssembleRequest{
			RunID:         "run-swapback-restart",
			SessionID:     "session-1",
			PrefixVersion: semanticPrefixVersion,
			Input:         "invoice payment status",
			Messages:      []types.Message{{Role: "system", Content: "base"}},
		}, types.ModelRequest{
			RunID:    "run-swapback-restart",
			Input:    "invoice payment status",
			Messages: []types.Message{{Role: "system", Content: "base"}},
		})
	}

	firstReq, firstResult, err := runOnce()
	if err != nil {
		t.Fatalf("first restart assemble failed: %v", err)
	}
	secondReq, secondResult, err := runOnce()
	if err != nil {
		t.Fatalf("second restart assemble failed: %v", err)
	}
	if firstResult.Stage.SwapBackCount != secondResult.Stage.SwapBackCount {
		t.Fatalf("swap_back_count mismatch across restart first=%d second=%d", firstResult.Stage.SwapBackCount, secondResult.Stage.SwapBackCount)
	}
	if joinMessageContents(firstReq.Messages) != joinMessageContents(secondReq.Messages) {
		t.Fatalf("swap-back output drift across restart first=%#v second=%#v", firstReq.Messages, secondReq.Messages)
	}
}
