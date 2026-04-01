package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
)

func TestManagerHotReloadRollbackAndSuccess(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
mcp:
  active_profile: default
  profiles:
    default:
      retry: 1
reload:
  enabled: true
  debounce: 20ms
`)

	mgr, err := NewManager(ManagerOptions{
		FilePath:        file,
		EnvPrefix:       "BAYMAX",
		EnableHotReload: true,
	})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().MCP.Profiles["default"].Retry
	if before != 1 {
		t.Fatalf("initial retry = %d, want 1", before)
	}

	writeConfig(t, file, `
mcp:
  active_profile: default
  profiles:
    default:
      retry: -1
`)

	time.Sleep(200 * time.Millisecond)
	afterInvalid := mgr.EffectiveConfig().MCP.Profiles["default"].Retry
	if afterInvalid != before {
		t.Fatalf("invalid reload should rollback, retry = %d, want %d", afterInvalid, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}

	writeConfig(t, file, `
mcp:
  active_profile: default
  profiles:
    default:
      retry: 5
reload:
  enabled: true
  debounce: 20ms
`)
	waitFor(t, 2*time.Second, func() bool {
		return mgr.EffectiveConfig().MCP.Profiles["default"].Retry == 5
	})
}

func TestManagerConcurrentReadsDuringReload(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 3s
      retry: 1
      backoff: 10ms
      queue_size: 32
      backpressure: block
      read_pool_size: 4
      write_pool_size: 1
reload:
  enabled: true
  debounce: 20ms
`)

	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	var wg sync.WaitGroup
	stop := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					cfg := mgr.EffectiveConfig()
					if cfg.MCP.Profiles[cfg.MCP.ActiveProfile].CallTimeout <= 0 {
						t.Errorf("observed partial snapshot")
						return
					}
					_, err := mgr.ResolvePolicy(cfg.MCP.ActiveProfile, nil)
					if err != nil {
						t.Errorf("ResolvePolicy failed: %v", err)
						return
					}
				}
			}
		}()
	}

	writeConfig(t, file, `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 9s
      retry: 2
      backoff: 15ms
      queue_size: 64
      backpressure: reject
      read_pool_size: 6
      write_pool_size: 2
reload:
  enabled: true
  debounce: 20ms
`)
	waitFor(t, 2*time.Second, func() bool {
		return mgr.EffectiveConfig().MCP.Profiles["default"].CallTimeout == 9*time.Second
	})
	close(stop)
	wg.Wait()
}

func TestManagerEffectiveConfigSanitizedUsesSecurityKeywords(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 3s
      retry: 1
      backoff: 10ms
      queue_size: 32
      backpressure: block
      read_pool_size: 4
      write_pool_size: 1
security:
  redaction:
    enabled: true
    strategy: keyword
    keywords: [secret]
`)

	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	out := mgr.RedactPayload(map[string]any{"client_secret": "abc", "name": "ok"})
	if out["client_secret"] != "***" {
		t.Fatalf("client_secret = %#v, want ***", out["client_secret"])
	}
	if out["name"] != "ok" {
		t.Fatalf("name = %#v, want ok", out["name"])
	}
}

func TestManagerRedactPayloadKeepsSkillTokenizerSignal(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 3s
      retry: 1
      backoff: 10ms
      queue_size: 32
      backpressure: block
      read_pool_size: 4
      write_pool_size: 1
security:
  redaction:
    enabled: true
    strategy: keyword
    keywords: [token, password, secret, api_key, apikey]
`)

	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	out := mgr.RedactPayload(map[string]any{
		"tokenizer_mode":          SkillTriggerScoringTokenizerMixedCJKEN,
		"candidate_pruned_count":  1,
		"budget_mode":             SkillTriggerScoringBudgetModeAdaptive,
		"selected_semantic_count": 3,
		"score_margin_top1_top2":  0.08,
		"budget_decision_reason":  "adaptive.max_k_reached",
		"bearer_token":            "secret-token",
	})
	if out["tokenizer_mode"] != SkillTriggerScoringTokenizerMixedCJKEN {
		t.Fatalf("tokenizer_mode = %#v, want %q", out["tokenizer_mode"], SkillTriggerScoringTokenizerMixedCJKEN)
	}
	if out["candidate_pruned_count"] != 1 {
		t.Fatalf("candidate_pruned_count = %#v, want 1", out["candidate_pruned_count"])
	}
	if out["budget_mode"] != SkillTriggerScoringBudgetModeAdaptive {
		t.Fatalf("budget_mode = %#v, want %q", out["budget_mode"], SkillTriggerScoringBudgetModeAdaptive)
	}
	if out["selected_semantic_count"] != 3 {
		t.Fatalf("selected_semantic_count = %#v, want 3", out["selected_semantic_count"])
	}
	if out["score_margin_top1_top2"] != 0.08 {
		t.Fatalf("score_margin_top1_top2 = %#v, want 0.08", out["score_margin_top1_top2"])
	}
	if out["budget_decision_reason"] != "adaptive.max_k_reached" {
		t.Fatalf("budget_decision_reason = %#v, want adaptive.max_k_reached", out["budget_decision_reason"])
	}
	if out["bearer_token"] != "***" {
		t.Fatalf("bearer_token = %#v, want ***", out["bearer_token"])
	}
}

func TestManagerPrecheckStage2External(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 3s
      retry: 1
      backoff: 10ms
      queue_size: 32
      backpressure: block
      read_pool_size: 4
      write_pool_size: 1
`)

	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	ext := DefaultConfig().ContextAssembler.CA2.Stage2.External
	ext.Endpoint = "http://127.0.0.1:8080/retrieve"
	ext.Profile = ContextStage2ExternalProfileRAGFlowLike
	result := mgr.PrecheckStage2External(ContextStage2ProviderHTTP, ext)
	if err := result.FirstError(); err != nil {
		t.Fatalf("precheck FirstError() = %v, want nil", err)
	}
}

func TestManagerTimelineTrendsAPIAndReloadRollback(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 3s
      retry: 1
      backoff: 10ms
      queue_size: 32
      backpressure: block
      read_pool_size: 4
      write_pool_size: 1
diagnostics:
  timeline_trend:
    enabled: true
    last_n_runs: 2
    time_window: 1m
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	base := time.Now()
	mgr.RecordRunTimelineEvent("run-1", "model", "running", 1, base)
	mgr.RecordRunTimelineEvent("run-1", "model", "succeeded", 2, base.Add(10*time.Millisecond))
	mgr.RecordRun(runtimediag.RunRecord{Time: base.Add(20 * time.Millisecond), RunID: "run-1", Status: "success"})
	trends := mgr.TimelineTrends(runtimediag.TimelineTrendQuery{Mode: runtimediag.TimelineTrendModeLastNRuns})
	if len(trends) == 0 {
		t.Fatal("timeline trends should not be empty")
	}

	writeConfig(t, file, `
mcp:
  active_profile: default
  profiles:
    default:
      retry: 1
diagnostics:
  timeline_trend:
    enabled: true
    last_n_runs: 0
    time_window: 1m
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	if mgr.EffectiveConfig().Diagnostics.TimelineTrend.LastNRuns != 2 {
		t.Fatalf("invalid reload should rollback timeline trend config, got %#v", mgr.EffectiveConfig().Diagnostics.TimelineTrend)
	}
}

func TestManagerCA2ExternalTrendsAPIAndReloadRollback(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 3s
      retry: 1
      backoff: 10ms
      queue_size: 32
      backpressure: block
      read_pool_size: 4
      write_pool_size: 1
diagnostics:
  ca2_external_trend:
    enabled: true
    window: 1m
    thresholds:
      p95_latency_ms: 50
      error_rate: 0.1
      hit_rate: 0.5
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	base := time.Now()
	mgr.RecordRun(runtimediag.RunRecord{
		Time:             base.Add(10 * time.Millisecond),
		RunID:            "run-ca2-1",
		Stage2Provider:   "http",
		Stage2LatencyMs:  80,
		Stage2HitCount:   0,
		Stage2ReasonCode: "timeout",
		Stage2ErrorLayer: "transport",
	})
	items := mgr.CA2ExternalTrends(runtimediag.CA2ExternalTrendQuery{})
	if len(items) != 1 {
		t.Fatalf("ca2 external trends len = %d, want 1", len(items))
	}
	if items[0].Provider != "http" {
		t.Fatalf("provider = %q, want http", items[0].Provider)
	}
	if len(items[0].ThresholdHits) == 0 {
		t.Fatalf("threshold hits should not be empty: %#v", items[0])
	}

	writeConfig(t, file, `
mcp:
  active_profile: default
  profiles:
    default:
      retry: 1
diagnostics:
  ca2_external_trend:
    enabled: true
    window: 0s
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	if mgr.EffectiveConfig().Diagnostics.CA2ExternalTrend.Window != 1*time.Minute {
		t.Fatalf("invalid reload should rollback CA2 trend config, got %#v", mgr.EffectiveConfig().Diagnostics.CA2ExternalTrend)
	}
}

func TestManagerDiagnosticsCardinalityInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
diagnostics:
  cardinality:
    enabled: true
    max_map_entries: 64
    max_list_entries: 64
    max_string_bytes: 2048
    overflow_policy: truncate_and_record
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Diagnostics.Cardinality
	if before.OverflowPolicy != DiagnosticsCardinalityOverflowTruncateAndRecord {
		t.Fatalf("before diagnostics.cardinality.overflow_policy = %q, want truncate_and_record", before.OverflowPolicy)
	}

	writeConfig(t, file, `
diagnostics:
  cardinality:
    enabled: true
    max_map_entries: 64
    max_list_entries: 64
    max_string_bytes: 2048
    overflow_policy: drop_new
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	afterEnum := mgr.EffectiveConfig().Diagnostics.Cardinality
	if afterEnum.OverflowPolicy != before.OverflowPolicy {
		t.Fatalf("invalid overflow_policy reload should rollback, got %#v want %#v", afterEnum, before)
	}

	writeConfig(t, file, `
diagnostics:
  cardinality:
    enabled: true
    max_map_entries: 0
    max_list_entries: 64
    max_string_bytes: 2048
    overflow_policy: truncate_and_record
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	afterThreshold := mgr.EffectiveConfig().Diagnostics.Cardinality
	if afterThreshold.MaxMapEntries != before.MaxMapEntries {
		t.Fatalf("invalid max_map_entries reload should rollback, got %#v want %#v", afterThreshold, before)
	}

	writeConfig(t, file, `
diagnostics:
  cardinality:
    enabled: definitely
    max_map_entries: 64
    max_list_entries: 64
    max_string_bytes: 2048
    overflow_policy: truncate_and_record
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	afterBool := mgr.EffectiveConfig().Diagnostics.Cardinality
	if afterBool.Enabled != before.Enabled {
		t.Fatalf("invalid enabled boolean reload should rollback, got %#v want %#v", afterBool, before)
	}

	reloads := mgr.RecentReloads(10)
	failed := 0
	for _, rec := range reloads {
		if !rec.Success {
			failed++
		}
	}
	if failed < 3 {
		t.Fatalf("expected at least 3 failed reload records, got %#v", reloads)
	}
}

func TestManagerUnifiedRunQueryAPIAndCompatibility(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 3s
      retry: 1
      backoff: 10ms
      queue_size: 32
      backpressure: block
      read_pool_size: 4
      write_pool_size: 1
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	base := time.Now()
	mgr.RecordRun(runtimediag.RunRecord{
		Time:       base,
		RunID:      "run-a18-manager-1",
		Status:     "success",
		TeamID:     "team-a",
		WorkflowID: "wf-a",
		TaskID:     "task-a18-manager-1",
	})

	query, err := mgr.QueryRuns(runtimediag.UnifiedRunQueryRequest{
		RunID: "run-a18-manager-1",
	})
	if err != nil {
		t.Fatalf("QueryRuns failed: %v", err)
	}
	if len(query.Items) != 1 || query.Items[0].RunID != "run-a18-manager-1" {
		t.Fatalf("query result mismatch: %#v", query)
	}
	if query.PageSize != runtimediag.DefaultUnifiedQueryPageSize || query.SortField != "time" || query.SortOrder != "desc" {
		t.Fatalf("query defaults mismatch: %#v", query)
	}

	recent := mgr.RecentRuns(5)
	if len(recent) != 1 || recent[0].RunID != "run-a18-manager-1" {
		t.Fatalf("RecentRuns compatibility changed: %#v", recent)
	}
	pageSizeInvalid := 201
	if _, err := mgr.QueryRuns(runtimediag.UnifiedRunQueryRequest{PageSize: &pageSizeInvalid}); err == nil {
		t.Fatal("expected fail-fast for invalid page size")
	}
}

func TestManagerCA2AgenticInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	stage2File := filepath.ToSlash(filepath.Join(t.TempDir(), "stage2.jsonl"))
	writeConfig(t, file, fmt.Sprintf(`
context_assembler:
  enabled: true
  ca2:
    enabled: true
    routing_mode: agentic
    agentic:
      decision_timeout: 80ms
      failure_policy: %s
    stage2:
      provider: file
      file_path: %s
reload:
  enabled: true
  debounce: 20ms
`, ContextCA2AgenticFailurePolicyBestEffortRules, stage2File))
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().ContextAssembler.CA2.Agentic.FailurePolicy
	if before != ContextCA2AgenticFailurePolicyBestEffortRules {
		t.Fatalf("before failure_policy = %q, want %q", before, ContextCA2AgenticFailurePolicyBestEffortRules)
	}

	writeConfig(t, file, fmt.Sprintf(`
context_assembler:
  enabled: true
  ca2:
    enabled: true
    routing_mode: agentic
    agentic:
      decision_timeout: 80ms
      failure_policy: deny
    stage2:
      provider: file
      file_path: %s
reload:
  enabled: true
  debounce: 20ms
`, stage2File))
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().ContextAssembler.CA2.Agentic.FailurePolicy
	if after != before {
		t.Fatalf("invalid ca2 agentic reload should rollback, failure_policy = %q, want %q", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerSkillTriggerEmbeddingInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
skill:
  trigger_scoring:
    strategy: lexical_plus_embedding
    confidence_threshold: 0.25
    tie_break: highest_priority
    suppress_low_confidence: true
    keyword_weights:
      db: 1.5
    embedding:
      enabled: true
      provider: openai
      model: text-embedding-3-small
      timeout: 300ms
      similarity_metric: cosine
      lexical_weight: 0.7
      embedding_weight: 0.3
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Skill.TriggerScoring.Embedding.LexicalWeight
	if before != 0.7 {
		t.Fatalf("before lexical_weight = %v, want 0.7", before)
	}

	writeConfig(t, file, `
skill:
  trigger_scoring:
    strategy: lexical_plus_embedding
    confidence_threshold: 0.25
    tie_break: highest_priority
    suppress_low_confidence: true
    keyword_weights:
      db: 1.5
    embedding:
      enabled: true
      provider: openai
      model: text-embedding-3-small
      timeout: 300ms
      similarity_metric: cosine
      lexical_weight: -0.1
      embedding_weight: 0.3
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Skill.TriggerScoring.Embedding.LexicalWeight
	if after != before {
		t.Fatalf("invalid skill embedding reload should rollback, lexical_weight = %v, want %v", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerSkillTriggerLexicalBudgetInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
skill:
  trigger_scoring:
    strategy: lexical_weighted_keywords
    confidence_threshold: 0.25
    tie_break: highest_priority
    suppress_low_confidence: true
    max_semantic_candidates: 3
    lexical:
      tokenizer_mode: mixed_cjk_en
    budget:
      mode: adaptive
      adaptive:
        min_k: 1
        max_k: 3
        min_score_margin: 0.08
    keyword_weights:
      db: 1.5
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Skill.TriggerScoring.MaxSemanticCandidates
	if before != 3 {
		t.Fatalf("before max_semantic_candidates = %d, want 3", before)
	}

	writeConfig(t, file, `
skill:
  trigger_scoring:
    strategy: lexical_weighted_keywords
    confidence_threshold: 0.25
    tie_break: highest_priority
    suppress_low_confidence: true
    max_semantic_candidates: 0
    lexical:
      tokenizer_mode: mixed_cjk_en
    budget:
      mode: adaptive
      adaptive:
        min_k: 1
        max_k: 3
        min_score_margin: 0.08
    keyword_weights:
      db: 1.5
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Skill.TriggerScoring.MaxSemanticCandidates
	if after != before {
		t.Fatalf("invalid lexical budget reload should rollback, max_semantic_candidates = %d, want %d", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerSkillTriggerAdaptiveBudgetInvalidRangeRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
skill:
  trigger_scoring:
    strategy: lexical_weighted_keywords
    confidence_threshold: 0.25
    tie_break: highest_priority
    suppress_low_confidence: true
    max_semantic_candidates: 5
    lexical:
      tokenizer_mode: mixed_cjk_en
    budget:
      mode: adaptive
      adaptive:
        min_k: 1
        max_k: 5
        min_score_margin: 0.08
    keyword_weights:
      db: 1.5
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Skill.TriggerScoring.Budget.Adaptive.MaxK
	if before != 5 {
		t.Fatalf("before max_k = %d, want 5", before)
	}

	writeConfig(t, file, `
skill:
  trigger_scoring:
    strategy: lexical_weighted_keywords
    confidence_threshold: 0.25
    tie_break: highest_priority
    suppress_low_confidence: true
    max_semantic_candidates: 5
    lexical:
      tokenizer_mode: mixed_cjk_en
    budget:
      mode: adaptive
      adaptive:
        min_k: 2
        max_k: 1
        min_score_margin: 0.08
    keyword_weights:
      db: 1.5
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Skill.TriggerScoring.Budget.Adaptive.MaxK
	if after != before {
		t.Fatalf("invalid adaptive budget reload should rollback, max_k = %d, want %d", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestSecurityPolicyContractInvalidSecurityReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
security:
  tool_governance:
    enabled: true
    mode: enforce
    permission:
      default: allow
      deny_action: deny
      by_tool:
        local+echo: allow
    rate_limit:
      enabled: true
      scope: process
      window: 1m
      limit: 10
      exceed_action: deny
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Security.ToolGovernance.Permission.ByTool["local+echo"]
	if before != SecurityToolPolicyAllow {
		t.Fatalf("before policy = %q, want allow", before)
	}

	writeConfig(t, file, `
security:
  tool_governance:
    enabled: true
    mode: enforce
    permission:
      default: allow
      deny_action: deny
      by_tool:
        local.echo: deny
    rate_limit:
      enabled: true
      scope: process
      window: 1m
      limit: 10
      exceed_action: deny
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Security.ToolGovernance.Permission.ByTool["local+echo"]
	if after != before {
		t.Fatalf("invalid security reload should rollback, policy = %q, want %q", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestSecurityEventContractInvalidSecurityEventReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
security:
  security_event:
    enabled: true
    alert:
      trigger_policy: deny_only
      sink: callback
      callback:
        require_registered: false
    severity:
      default: high
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Security.SecurityEvent.Alert.TriggerPolicy
	if before != SecurityEventAlertPolicyDenyOnly {
		t.Fatalf("before trigger_policy = %q, want deny_only", before)
	}

	writeConfig(t, file, `
security:
  security_event:
    enabled: true
    alert:
      trigger_policy: all
      sink: callback
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Security.SecurityEvent.Alert.TriggerPolicy
	if after != before {
		t.Fatalf("invalid security_event reload should rollback, trigger_policy = %q, want %q", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestSecurityDeliveryContractInvalidSecurityEventDeliveryReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
security:
  security_event:
    enabled: true
    alert:
      trigger_policy: deny_only
      sink: callback
    delivery:
      mode: async
      queue:
        size: 32
        overflow_policy: drop_old
      timeout: 1s
      retry:
        max_attempts: 3
        backoff_initial: 40ms
        backoff_max: 120ms
      circuit_breaker:
        failure_threshold: 5
        open_window: 3s
        half_open_probes: 1
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Security.SecurityEvent.Delivery.Mode
	if before != SecurityEventDeliveryModeAsync {
		t.Fatalf("before delivery.mode = %q, want async", before)
	}

	writeConfig(t, file, `
security:
  security_event:
    enabled: true
    alert:
      trigger_policy: deny_only
      sink: callback
    delivery:
      mode: async
      queue:
        size: 32
        overflow_policy: drop_old
      timeout: 1s
      retry:
        max_attempts: 5
        backoff_initial: 40ms
        backoff_max: 120ms
      circuit_breaker:
        failure_threshold: 5
        open_window: 3s
        half_open_probes: 1
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Security.SecurityEvent.Delivery.Retry.MaxAttempts
	if after != 3 {
		t.Fatalf("invalid security_event.delivery reload should rollback, max_attempts = %d, want 3", after)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestSecuritySandboxContractInvalidSandboxReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
security:
  sandbox:
    enabled: true
    mode: observe
    policy:
      default_action: host
      profile: default
      fallback_action: allow_and_record
      fallback_action_by_tool:
        local+shell: deny
    executor:
      backend: windows_job
      session_mode: per_call
      required_capabilities: [stdout_stderr_capture]
    profiles:
      default:
        network:
          mode: network_off
        filesystem:
          readonly_root: true
        mounts: []
        resource_limits:
          cpu_milli: 1000
          memory_bytes: 536870912
          pid_limit: 64
        timeouts:
          launch_timeout: 1s
          exec_timeout: 5s
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Security.Sandbox.Executor.Backend
	if before != SecuritySandboxBackendWindowsJob {
		t.Fatalf("before sandbox backend = %q, want %q", before, SecuritySandboxBackendWindowsJob)
	}

	writeConfig(t, file, `
security:
  sandbox:
    enabled: true
    mode: observe
    policy:
      default_action: host
      profile: default
      fallback_action: allow_and_record
    executor:
      backend: sandboxie
      session_mode: per_call
      required_capabilities: [stdout_stderr_capture]
    profiles:
      default:
        network:
          mode: network_off
        filesystem:
          readonly_root: true
        mounts: []
        resource_limits:
          cpu_milli: 1000
          memory_bytes: 536870912
          pid_limit: 64
        timeouts:
          launch_timeout: 1s
          exec_timeout: 5s
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Security.Sandbox.Executor.Backend
	if after != before {
		t.Fatalf("invalid sandbox reload should rollback, backend = %q, want %q", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestSecuritySandboxRolloutPhaseTransitionReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
security:
  sandbox:
    enabled: true
    mode: observe
    policy:
      default_action: host
      profile: default
      fallback_action: allow_and_record
      fallback_action_by_tool:
        local+shell: deny
    executor:
      backend: windows_job
      session_mode: per_call
      required_capabilities: [stdout_stderr_capture]
    profiles:
      default:
        network:
          mode: network_off
        filesystem:
          readonly_root: true
        mounts: []
        resource_limits:
          cpu_milli: 1000
          memory_bytes: 536870912
          pid_limit: 64
        timeouts:
          launch_timeout: 1s
          exec_timeout: 5s
    rollout:
      phase: full
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Security.Sandbox.Rollout.Phase
	if before != SecuritySandboxRolloutPhaseFull {
		t.Fatalf("before sandbox rollout phase = %q, want %q", before, SecuritySandboxRolloutPhaseFull)
	}

	writeConfig(t, file, `
security:
  sandbox:
    enabled: true
    mode: observe
    policy:
      default_action: host
      profile: default
      fallback_action: allow_and_record
      fallback_action_by_tool:
        local+shell: deny
    executor:
      backend: windows_job
      session_mode: per_call
      required_capabilities: [stdout_stderr_capture]
    profiles:
      default:
        network:
          mode: network_off
        filesystem:
          readonly_root: true
        mounts: []
        resource_limits:
          cpu_milli: 1000
          memory_bytes: 536870912
          pid_limit: 64
        timeouts:
          launch_timeout: 1s
          exec_timeout: 5s
    rollout:
      phase: observe
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Security.Sandbox.Rollout.Phase
	if after != before {
		t.Fatalf("invalid sandbox rollout transition should rollback, phase = %q, want %q", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestSecuritySandboxEgressInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
security:
  sandbox:
    egress:
      enabled: true
      default_action: deny
      on_violation: deny
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Security.Sandbox.Egress.DefaultAction
	if before != SecuritySandboxEgressActionDeny {
		t.Fatalf("before sandbox egress default_action = %q, want %q", before, SecuritySandboxEgressActionDeny)
	}

	writeConfig(t, file, `
security:
  sandbox:
    egress:
      enabled: true
      default_action: allow
      on_violation: deny
      allowlist:
        - api.example.com
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Security.Sandbox.Egress.DefaultAction
	if after != before {
		t.Fatalf("invalid sandbox egress reload should rollback, default_action = %q, want %q", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerSandboxRolloutGovernanceRecordRunAutoFreeze(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-a52-governance.yaml")
	writeConfig(t, file, `
security:
  sandbox:
    enabled: true
    rollout:
      phase: canary
      error_budget: 0.05
      freeze_on_breach: true
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_A52_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	mgr.RecordRun(runtimediag.RunRecord{
		RunID:                    "run-a52-breach",
		SandboxLaunchFailedTotal: 1,
	})
	state := mgr.SandboxRolloutRuntimeState()
	if state.HealthBudgetStatus != SandboxHealthBudgetBreached {
		t.Fatalf("health budget status=%q, want %q", state.HealthBudgetStatus, SandboxHealthBudgetBreached)
	}
	if !state.FreezeState {
		t.Fatalf("freeze_state=false, want true")
	}
	if state.FreezeReasonCode != ReadinessCodeSandboxRolloutHealthBreached {
		t.Fatalf("freeze_reason_code=%q, want %q", state.FreezeReasonCode, ReadinessCodeSandboxRolloutHealthBreached)
	}
	if state.CapacityAction != SandboxCapacityActionDeny {
		t.Fatalf("capacity_action=%q, want %q", state.CapacityAction, SandboxCapacityActionDeny)
	}
}

func TestManagerSandboxRolloutUnfreezeRequiresCooldownAndToken(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-a52-unfreeze.yaml")
	writeConfig(t, file, `
security:
  sandbox:
    enabled: true
    rollout:
      phase: frozen
      cooldown: 100ms
      manual_unfreeze_token: token-1
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_A52_TEST", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	mgr.SetSandboxRolloutRuntimeState(SandboxRolloutRuntimeState{
		FreezeState: true,
		UpdatedAt:   time.Now().UTC(),
	})
	writeConfig(t, file, `
security:
  sandbox:
    enabled: true
    rollout:
      phase: canary
      cooldown: 100ms
      manual_unfreeze_token: token-1
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	if got := mgr.EffectiveConfig().Security.Sandbox.Rollout.Phase; got != SecuritySandboxRolloutPhaseFrozen {
		t.Fatalf("phase should remain frozen before cooldown, got %q", got)
	}

	mgr.SetSandboxRolloutRuntimeState(SandboxRolloutRuntimeState{
		FreezeState: true,
		UpdatedAt:   time.Now().UTC().Add(-500 * time.Millisecond),
	})
	writeConfig(t, file, `
security:
  sandbox:
    enabled: true
    rollout:
      phase: canary
      cooldown: 100ms
      manual_unfreeze_token: token-1
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	if got := mgr.EffectiveConfig().Security.Sandbox.Rollout.Phase; got != SecuritySandboxRolloutPhaseCanary {
		t.Fatalf("phase should unfreeze to canary after cooldown with valid token, got %q", got)
	}
}

func TestManagerSandboxCapacityActionDeterministicFromQueueAndInflight(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-a52-capacity.yaml")
	writeConfig(t, file, `
security:
  sandbox:
    enabled: true
    rollout:
      phase: canary
    capacity:
      max_inflight: 10
      max_queue: 20
      throttle_threshold: 5
      deny_threshold: 15
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_A52_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	mgr.RecordRun(runtimediag.RunRecord{
		RunID:               "run-a52-capacity-allow",
		SchedulerQueueTotal: 3,
		InflightPeak:        2,
	})
	if got := mgr.SandboxRolloutRuntimeState().CapacityAction; got != SandboxCapacityActionAllow {
		t.Fatalf("capacity action (allow case)=%q, want %q", got, SandboxCapacityActionAllow)
	}

	mgr.RecordRun(runtimediag.RunRecord{
		RunID:               "run-a52-capacity-throttle",
		SchedulerQueueTotal: 6,
		InflightPeak:        2,
	})
	if got := mgr.SandboxRolloutRuntimeState().CapacityAction; got != SandboxCapacityActionThrottle {
		t.Fatalf("capacity action (throttle case)=%q, want %q", got, SandboxCapacityActionThrottle)
	}

	mgr.RecordRun(runtimediag.RunRecord{
		RunID:               "run-a52-capacity-deny",
		SchedulerQueueTotal: 16,
		InflightPeak:        2,
	})
	if got := mgr.SandboxRolloutRuntimeState().CapacityAction; got != SandboxCapacityActionDeny {
		t.Fatalf("capacity action (deny case)=%q, want %q", got, SandboxCapacityActionDeny)
	}
}

func TestManagerTeamsInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
teams:
  enabled: true
  default_strategy: serial
  task_timeout: 1s
  parallel:
    max_workers: 3
  vote:
    tie_break: highest_priority
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Teams.DefaultStrategy
	if before != TeamsStrategySerial {
		t.Fatalf("before teams.default_strategy = %q, want %q", before, TeamsStrategySerial)
	}

	writeConfig(t, file, `
teams:
  enabled: true
  default_strategy: weighted
  task_timeout: 1s
  parallel:
    max_workers: 3
  vote:
    tie_break: highest_priority
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Teams.DefaultStrategy
	if after != before {
		t.Fatalf("invalid teams reload should rollback, default_strategy = %q, want %q", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerComposerCollabInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
composer:
  collab:
    enabled: true
    default_aggregation: all_settled
    failure_policy: fail_fast
    retry:
      enabled: false
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Composer.Collab.DefaultAggregation
	if before != ComposerCollabAggregationAllSettled {
		t.Fatalf("before default_aggregation = %q, want all_settled", before)
	}

	writeConfig(t, file, `
composer:
  collab:
    enabled: true
    default_aggregation: quorum
    failure_policy: fail_fast
    retry:
      enabled: false
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Composer.Collab.DefaultAggregation
	if after != before {
		t.Fatalf("invalid composer collab reload should rollback, default_aggregation = %q, want %q", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerTeamsRemoteInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
teams:
  enabled: true
  default_strategy: serial
  task_timeout: 1s
  parallel:
    max_workers: 3
  vote:
    tie_break: highest_priority
  remote:
    enabled: false
    require_peer_id: true
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Teams.Remote.Enabled
	if before {
		t.Fatal("before teams.remote.enabled = true, want false")
	}

	writeConfig(t, file, `
teams:
  enabled: false
  default_strategy: serial
  task_timeout: 1s
  parallel:
    max_workers: 3
  vote:
    tie_break: highest_priority
  remote:
    enabled: true
    require_peer_id: true
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Teams.Remote.Enabled
	if after != before {
		t.Fatalf("invalid teams remote reload should rollback, remote.enabled = %v, want %v", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerWorkflowInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
workflow:
  enabled: true
  planner_validation_mode: strict
  default_step_timeout: 1200ms
  checkpoint_backend: memory
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Workflow.PlannerValidationMode
	if before != WorkflowValidationModeStrict {
		t.Fatalf("before planner_validation_mode = %q, want %q", before, WorkflowValidationModeStrict)
	}

	writeConfig(t, file, `
workflow:
  enabled: true
  planner_validation_mode: strict
  default_step_timeout: 0s
  checkpoint_backend: memory
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Workflow.DefaultStepTimeout
	if after != 1200*time.Millisecond {
		t.Fatalf("invalid workflow reload should rollback, default_step_timeout = %v, want 1200ms", after)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerWorkflowRemoteInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
workflow:
  enabled: true
  planner_validation_mode: strict
  default_step_timeout: 1200ms
  checkpoint_backend: memory
  remote:
    enabled: true
    require_peer_id: true
    default_retry_max_attempts: 2
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Workflow.Remote.DefaultRetryMaxAttempts
	if before != 2 {
		t.Fatalf("before workflow.remote.default_retry_max_attempts = %d, want 2", before)
	}

	writeConfig(t, file, `
workflow:
  enabled: true
  planner_validation_mode: strict
  default_step_timeout: 1200ms
  checkpoint_backend: memory
  remote:
    enabled: true
    require_peer_id: true
    default_retry_max_attempts: -1
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Workflow.Remote.DefaultRetryMaxAttempts
	if after != before {
		t.Fatalf("invalid workflow remote reload should rollback, default_retry_max_attempts = %d, want %d", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerA2AInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
a2a:
  enabled: true
  client_timeout: 1200ms
  delivery:
    mode: callback
    fallback_mode: sse
    callback_retry:
      max_attempts: 3
      backoff: 80ms
    sse_reconnect:
      max_attempts: 3
      backoff: 80ms
  card:
    version_policy:
      mode: strict_major
      min_supported_minor: 0
  capability_discovery:
    enabled: true
    require_all: true
    max_candidates: 8
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().A2A.ClientTimeout
	if before != 1200*time.Millisecond {
		t.Fatalf("before a2a.client_timeout = %v, want 1200ms", before)
	}

	writeConfig(t, file, `
a2a:
  enabled: true
  client_timeout: 1200ms
  delivery:
    mode: callback
    fallback_mode: sse
    callback_retry:
      max_attempts: 3
      backoff: 80ms
    sse_reconnect:
      max_attempts: 0
      backoff: 80ms
  card:
    version_policy:
      mode: strict_major
      min_supported_minor: 0
  capability_discovery:
    enabled: true
    require_all: true
    max_candidates: 8
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().A2A.ClientTimeout
	if after != before {
		t.Fatalf("invalid a2a reload should rollback, client_timeout = %v, want %v", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerComposerCollabRetryInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
composer:
  collab:
    enabled: true
    default_aggregation: all_settled
    failure_policy: fail_fast
    retry:
      enabled: true
      max_attempts: 3
      backoff_initial: 100ms
      backoff_max: 2s
      multiplier: 2
      jitter_ratio: 0.2
      retry_on: transport_only
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Composer.Collab.Retry.MaxAttempts
	if before != 3 {
		t.Fatalf("before composer.collab.retry.max_attempts = %d, want 3", before)
	}

	writeConfig(t, file, `
composer:
  collab:
    enabled: true
    default_aggregation: all_settled
    failure_policy: fail_fast
    retry:
      enabled: true
      max_attempts: 3
      backoff_initial: 100ms
      backoff_max: 50ms
      multiplier: 2
      jitter_ratio: 0.2
      retry_on: transport_only
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Composer.Collab.Retry.MaxAttempts
	if after != before {
		t.Fatalf("invalid collab retry reload should rollback, max_attempts = %d, want %d", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerA2AVersionPolicyInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
a2a:
  enabled: true
  client_timeout: 1200ms
  delivery:
    mode: callback
    fallback_mode: callback
    callback_retry:
      max_attempts: 3
      backoff: 80ms
    sse_reconnect:
      max_attempts: 3
      backoff: 80ms
  card:
    version_policy:
      mode: strict_major
      min_supported_minor: 1
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().A2A.Card.VersionPolicy.MinSupportedMinor
	if before != 1 {
		t.Fatalf("before min_supported_minor = %d, want 1", before)
	}

	writeConfig(t, file, `
a2a:
  enabled: true
  client_timeout: 1200ms
  delivery:
    mode: callback
    fallback_mode: callback
    callback_retry:
      max_attempts: 3
      backoff: 80ms
    sse_reconnect:
      max_attempts: 3
      backoff: 80ms
  card:
    version_policy:
      mode: compat
      min_supported_minor: 1
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().A2A.Card.VersionPolicy.MinSupportedMinor
	if after != before {
		t.Fatalf("invalid a2a version policy reload should rollback, min_supported_minor = %d, want %d", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerA2AAsyncReportingInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
a2a:
  enabled: true
  client_timeout: 1200ms
  delivery:
    mode: callback
    fallback_mode: callback
    callback_retry:
      max_attempts: 3
      backoff: 80ms
    sse_reconnect:
      max_attempts: 3
      backoff: 80ms
  card:
    version_policy:
      mode: strict_major
      min_supported_minor: 1
  async_reporting:
    enabled: true
    sink: callback
    retry:
      max_attempts: 3
      backoff_initial: 50ms
      backoff_max: 400ms
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().A2A.AsyncReporting.Retry.MaxAttempts
	if before != 3 {
		t.Fatalf("before async_reporting.retry.max_attempts = %d, want 3", before)
	}

	writeConfig(t, file, `
a2a:
  enabled: true
  client_timeout: 1200ms
  delivery:
    mode: callback
    fallback_mode: callback
    callback_retry:
      max_attempts: 3
      backoff: 80ms
    sse_reconnect:
      max_attempts: 3
      backoff: 80ms
  card:
    version_policy:
      mode: strict_major
      min_supported_minor: 1
  async_reporting:
    enabled: true
    sink: callback
    retry:
      max_attempts: 3
      backoff_initial: 500ms
      backoff_max: 200ms
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().A2A.AsyncReporting.Retry.MaxAttempts
	if after != before {
		t.Fatalf("invalid async reporting reload should rollback, max_attempts = %d, want %d", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerSchedulerInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
scheduler:
  enabled: true
  backend: memory
  lease_timeout: 2s
  heartbeat_interval: 500ms
  queue_limit: 1024
  retry_max_attempts: 3
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Scheduler.HeartbeatInterval
	if before != 500*time.Millisecond {
		t.Fatalf("before scheduler.heartbeat_interval = %v, want 500ms", before)
	}

	writeConfig(t, file, `
scheduler:
  enabled: true
  backend: memory
  lease_timeout: 2s
  heartbeat_interval: 2s
  queue_limit: 1024
  retry_max_attempts: 3
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Scheduler.HeartbeatInterval
	if after != before {
		t.Fatalf("invalid scheduler reload should rollback, heartbeat_interval = %v, want %v", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerSchedulerQoSInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
scheduler:
  enabled: true
  backend: memory
  lease_timeout: 2s
  heartbeat_interval: 500ms
  queue_limit: 1024
  retry_max_attempts: 3
  qos:
    mode: fifo
    fairness:
      max_consecutive_claims_per_priority: 3
  retry:
    backoff:
      initial: 50ms
      max: 2s
      multiplier: 2
      jitter_ratio: 0.2
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Scheduler.QoS.Fairness.MaxConsecutiveClaimsPerPriority
	if before != 3 {
		t.Fatalf("before scheduler.qos.fairness.max_consecutive_claims_per_priority = %d, want 3", before)
	}

	writeConfig(t, file, `
scheduler:
  enabled: true
  backend: memory
  lease_timeout: 2s
  heartbeat_interval: 500ms
  queue_limit: 1024
  retry_max_attempts: 3
  qos:
    mode: priority
    fairness:
      max_consecutive_claims_per_priority: 0
  retry:
    backoff:
      initial: 50ms
      max: 2s
      multiplier: 2
      jitter_ratio: 0.2
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Scheduler.QoS.Fairness.MaxConsecutiveClaimsPerPriority
	if after != before {
		t.Fatalf("invalid scheduler qos reload should rollback, fairness.max_consecutive_claims_per_priority = %d, want %d", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerSchedulerAsyncAwaitInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
scheduler:
  enabled: true
  backend: memory
  lease_timeout: 2s
  heartbeat_interval: 500ms
  queue_limit: 1024
  retry_max_attempts: 3
  async_await:
    report_timeout: 15m
    late_report_policy: drop_and_record
    timeout_terminal: failed
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Scheduler.AsyncAwait.ReportTimeout
	if before != 15*time.Minute {
		t.Fatalf("before scheduler.async_await.report_timeout = %v, want 15m", before)
	}

	writeConfig(t, file, `
scheduler:
  enabled: true
  backend: memory
  lease_timeout: 2s
  heartbeat_interval: 500ms
  queue_limit: 1024
  retry_max_attempts: 3
  async_await:
    report_timeout: 0s
    late_report_policy: overwrite
    timeout_terminal: timeout
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Scheduler.AsyncAwait.ReportTimeout
	if after != before {
		t.Fatalf("invalid scheduler async_await reload should rollback, report_timeout = %v, want %v", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerSchedulerTaskBoardControlInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
scheduler:
  enabled: true
  backend: memory
  lease_timeout: 2s
  heartbeat_interval: 500ms
  queue_limit: 1024
  retry_max_attempts: 3
  task_board:
    control:
      enabled: true
      max_manual_retry_per_task: 4
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Scheduler.TaskBoard.Control.MaxManualRetryPerTask
	if before != 4 {
		t.Fatalf("before scheduler.task_board.control.max_manual_retry_per_task = %d, want 4", before)
	}

	writeConfig(t, file, `
scheduler:
  enabled: true
  backend: memory
  lease_timeout: 2s
  heartbeat_interval: 500ms
  queue_limit: 1024
  retry_max_attempts: 3
  task_board:
    control:
      enabled: true
      max_manual_retry_per_task: 0
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Scheduler.TaskBoard.Control.MaxManualRetryPerTask
	if after != before {
		t.Fatalf(
			"invalid scheduler task_board control reload should rollback, max_manual_retry_per_task = %d, want %d",
			after,
			before,
		)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerRecoveryInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
recovery:
  enabled: true
  backend: memory
  conflict_policy: fail_fast
  resume_boundary: next_attempt_only
  inflight_policy: no_rewind
  timeout_reentry_policy: single_reentry_then_fail
  timeout_reentry_max_per_task: 1
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Recovery.ConflictPolicy
	if before != RecoveryConflictPolicyFailFast {
		t.Fatalf("before recovery.conflict_policy = %q, want fail_fast", before)
	}
	beforeBoundary := mgr.EffectiveConfig().Recovery.ResumeBoundary
	if beforeBoundary != RecoveryResumeBoundaryNextAttemptOnly {
		t.Fatalf("before recovery.resume_boundary = %q, want %q", beforeBoundary, RecoveryResumeBoundaryNextAttemptOnly)
	}

	writeConfig(t, file, `
recovery:
  enabled: true
  backend: memory
  conflict_policy: best_effort
  resume_boundary: next_attempt_only
  inflight_policy: no_rewind
  timeout_reentry_policy: single_reentry_then_fail
  timeout_reentry_max_per_task: 1
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Recovery.ConflictPolicy
	if after != before {
		t.Fatalf("invalid recovery reload should rollback, conflict_policy = %q, want %q", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerRecoveryBoundaryInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
recovery:
  enabled: true
  backend: memory
  conflict_policy: fail_fast
  resume_boundary: next_attempt_only
  inflight_policy: no_rewind
  timeout_reentry_policy: single_reentry_then_fail
  timeout_reentry_max_per_task: 1
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Recovery.TimeoutReentryMaxPerTask
	if before != 1 {
		t.Fatalf("before recovery.timeout_reentry_max_per_task = %d, want 1", before)
	}

	writeConfig(t, file, `
recovery:
  enabled: true
  backend: memory
  conflict_policy: fail_fast
  resume_boundary: next_attempt_only
  inflight_policy: no_rewind
  timeout_reentry_policy: single_reentry_then_fail
  timeout_reentry_max_per_task: 2
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Recovery.TimeoutReentryMaxPerTask
	if after != before {
		t.Fatalf("invalid recovery boundary reload should rollback, timeout_reentry_max_per_task = %d, want %d", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerSubagentInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
subagent:
  max_depth: 4
  max_active_children: 8
  child_timeout_budget: 5s
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Subagent.MaxDepth
	if before != 4 {
		t.Fatalf("before subagent.max_depth = %d, want 4", before)
	}

	writeConfig(t, file, `
subagent:
  max_depth: 0
  max_active_children: 8
  child_timeout_budget: 5s
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Subagent.MaxDepth
	if after != before {
		t.Fatalf("invalid subagent reload should rollback, max_depth = %d, want %d", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerMailboxDiagnosticsQueryAndAggregate(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
mailbox:
  enabled: true
  backend: memory
  retry:
    max_attempts: 3
    backoff_initial: 50ms
    backoff_max: 500ms
    jitter_ratio: 0.2
  ttl: 15m
  dlq:
    enabled: false
  query:
    page_size_default: 50
    page_size_max: 200
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	base := time.Now().UTC()
	mgr.RecordMailbox(runtimediag.MailboxRecord{
		Time:       base,
		MessageID:  "msg-a",
		Kind:       "command",
		State:      "queued",
		RunID:      "run-mailbox-1",
		TaskID:     "task-mailbox-1",
		WorkflowID: "wf-mailbox-1",
		TeamID:     "team-mailbox-1",
	})
	mgr.RecordMailbox(runtimediag.MailboxRecord{
		Time:       base.Add(10 * time.Millisecond),
		MessageID:  "msg-b",
		Kind:       "result",
		State:      "dead_letter",
		RunID:      "run-mailbox-1",
		TaskID:     "task-mailbox-1",
		WorkflowID: "wf-mailbox-1",
		TeamID:     "team-mailbox-1",
		ReasonCode: "retry_exhausted",
		Attempt:    3,
	})
	mgr.RecordMailboxDiagnostic(MailboxDiagnosticRecord{
		Time:           base.Add(20 * time.Millisecond),
		MessageID:      "msg-c",
		Kind:           "command",
		State:          "queued",
		RunID:          "run-mailbox-1",
		TaskID:         "task-mailbox-1",
		WorkflowID:     "wf-mailbox-1",
		TeamID:         "team-mailbox-1",
		ReasonCode:     "lease_expired",
		Attempt:        2,
		Reclaimed:      true,
		PanicRecovered: true,
	})

	page, err := mgr.QueryMailbox(runtimediag.MailboxQueryRequest{
		RunID: "run-mailbox-1",
	})
	if err != nil {
		t.Fatalf("QueryMailbox failed: %v", err)
	}
	if len(page.Items) != 3 {
		t.Fatalf("query mailbox items len = %d, want 3", len(page.Items))
	}
	var reclaimSeen bool
	var panicSeen bool
	for _, rec := range page.Items {
		if rec.RunID == "" || rec.TaskID == "" || rec.WorkflowID == "" || rec.TeamID == "" {
			t.Fatalf("correlation fields must be preserved: %#v", rec)
		}
		if rec.Reclaimed {
			reclaimSeen = true
		}
		if rec.PanicRecovered {
			panicSeen = true
		}
	}
	if !reclaimSeen || !panicSeen {
		t.Fatalf("mailbox reclaim/panic recovered flags missing: %#v", page.Items)
	}
	agg := mgr.MailboxAggregates(runtimediag.MailboxAggregateRequest{
		RunID: "run-mailbox-1",
	})
	if agg.TotalMessages != 3 ||
		agg.ByState["dead_letter"] != 1 ||
		agg.ReasonCodeTotals["retry_exhausted"] != 1 ||
		agg.ReasonCodeTotals["lease_expired"] != 1 {
		t.Fatalf("mailbox aggregate mismatch: %#v", agg)
	}
}

func TestManagerMailboxInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
mailbox:
  enabled: true
  backend: memory
  retry:
    max_attempts: 3
    backoff_initial: 50ms
    backoff_max: 500ms
    jitter_ratio: 0.2
  ttl: 15m
  dlq:
    enabled: false
  query:
    page_size_default: 50
    page_size_max: 200
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Mailbox.Query.PageSizeMax
	if before != 200 {
		t.Fatalf("before mailbox.query.page_size_max = %d, want 200", before)
	}

	writeConfig(t, file, `
mailbox:
  enabled: true
  backend: memory
  retry:
    max_attempts: 3
    backoff_initial: 50ms
    backoff_max: 500ms
    jitter_ratio: 0.2
  ttl: 15m
  dlq:
    enabled: false
  query:
    page_size_default: 50
    page_size_max: 500
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Mailbox.Query.PageSizeMax
	if after != before {
		t.Fatalf("invalid mailbox reload should rollback, page_size_max = %d, want %d", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerMailboxWorkerInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
mailbox:
  enabled: true
  backend: memory
  retry:
    max_attempts: 3
    backoff_initial: 50ms
    backoff_max: 500ms
    jitter_ratio: 0.2
  ttl: 15m
  dlq:
    enabled: false
  query:
    page_size_default: 50
    page_size_max: 200
  worker:
    enabled: false
    poll_interval: 100ms
    handler_error_policy: requeue
    inflight_timeout: 30s
    heartbeat_interval: 5s
    reclaim_on_consume: true
    panic_policy: follow_handler_error_policy
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Mailbox.Worker.HeartbeatInterval
	if before != 5*time.Second {
		t.Fatalf("before mailbox.worker.heartbeat_interval = %v, want 5s", before)
	}

	writeConfig(t, file, `
mailbox:
  enabled: true
  backend: memory
  retry:
    max_attempts: 3
    backoff_initial: 50ms
    backoff_max: 500ms
    jitter_ratio: 0.2
  ttl: 15m
  dlq:
    enabled: false
  query:
    page_size_default: 50
    page_size_max: 200
  worker:
    enabled: true
    poll_interval: 100ms
    handler_error_policy: requeue
    inflight_timeout: 8s
    heartbeat_interval: 8s
    reclaim_on_consume: true
    panic_policy: follow_handler_error_policy
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Mailbox.Worker.HeartbeatInterval
	if after != before {
		t.Fatalf("invalid mailbox.worker reload should rollback, heartbeat_interval = %v, want %v", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerRuntimeReadinessInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Runtime.Readiness.Strict
	if before {
		t.Fatal("before runtime.readiness.strict = true, want false")
	}

	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: absolutely-no
    remote_probe_enabled: false
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Runtime.Readiness.Strict
	if after != before {
		t.Fatalf("invalid readiness reload should rollback, strict = %v, want %v", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerRuntimeReadinessAdmissionInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
    admission:
      enabled: true
      mode: fail_fast
      block_on: blocked_only
      degraded_policy: allow_and_record
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Runtime.Readiness.Admission.DegradedPolicy
	if before != ReadinessAdmissionDegradedPolicyAllowAndRecord {
		t.Fatalf(
			"before runtime.readiness.admission.degraded_policy = %q, want %q",
			before,
			ReadinessAdmissionDegradedPolicyAllowAndRecord,
		)
	}

	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
    admission:
      enabled: true
      mode: fail_fast
      block_on: blocked_only
      degraded_policy: shadow_deny
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Runtime.Readiness.Admission.DegradedPolicy
	if after != before {
		t.Fatalf("invalid readiness admission reload should rollback, degraded_policy = %q, want %q", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerRuntimeArbitrationVersionInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  arbitration:
    version:
      enabled: true
      default: a49.v1
      compat_window: 1
      on_unsupported: fail_fast
      on_mismatch: fail_fast
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Runtime.Arbitration.Version.OnMismatch
	if before != RuntimeArbitrationVersionMismatchPolicyFailFast {
		t.Fatalf("before runtime.arbitration.version.on_mismatch = %q, want %q", before, RuntimeArbitrationVersionMismatchPolicyFailFast)
	}

	writeConfig(t, file, `
runtime:
  arbitration:
    version:
      enabled: true
      default: a49.v1
      compat_window: 1
      on_unsupported: fail_fast
      on_mismatch: best_effort
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Runtime.Arbitration.Version.OnMismatch
	if after != before {
		t.Fatalf("invalid arbitration version reload should rollback, on_mismatch = %q, want %q", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerAdapterHealthInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
adapter:
  health:
    enabled: true
    strict: false
    probe_timeout: 500ms
    cache_ttl: 30s
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Adapter.Health.ProbeTimeout
	if before != 500*time.Millisecond {
		t.Fatalf("before adapter.health.probe_timeout = %v, want 500ms", before)
	}

	writeConfig(t, file, `
adapter:
  health:
    enabled: true
    strict: false
    probe_timeout: 0s
    cache_ttl: 30s
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Adapter.Health.ProbeTimeout
	if after != before {
		t.Fatalf("invalid adapter health reload should rollback, probe_timeout = %v, want %v", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerAdapterHealthGovernanceInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
adapter:
  health:
    enabled: true
    strict: false
    probe_timeout: 500ms
    cache_ttl: 30s
    backoff:
      enabled: true
      initial: 200ms
      max: 5s
      multiplier: 2
      jitter_ratio: 0.2
    circuit:
      enabled: true
      failure_threshold: 3
      open_duration: 30s
      half_open_max_probe: 1
      half_open_success_threshold: 2
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Adapter.Health.Backoff.Multiplier
	if before != 2 {
		t.Fatalf("before adapter.health.backoff.multiplier = %v, want 2", before)
	}

	writeConfig(t, file, `
adapter:
  health:
    enabled: true
    strict: false
    probe_timeout: 500ms
    cache_ttl: 30s
    backoff:
      enabled: true
      initial: 200ms
      max: 5s
      multiplier: 1
      jitter_ratio: 0.2
    circuit:
      enabled: true
      failure_threshold: 3
      open_duration: 30s
      half_open_max_probe: 1
      half_open_success_threshold: 2
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Adapter.Health.Backoff.Multiplier
	if after != before {
		t.Fatalf("invalid adapter health governance reload should rollback, multiplier=%v want=%v", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerAdapterAllowlistInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
adapter:
  allowlist:
    enabled: true
    enforcement_mode: enforce
    on_unknown_signature: deny
    entries:
      - adapter_id: adapter.one
        publisher: acme
        version: 1.0.0
        signature_status: valid
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := len(mgr.EffectiveConfig().Adapter.Allowlist.Entries)
	if before != 1 {
		t.Fatalf("before adapter.allowlist.entries len = %d, want 1", before)
	}

	writeConfig(t, file, `
adapter:
  allowlist:
    enabled: true
    enforcement_mode: enforce
    on_unknown_signature: deny
    entries:
      - adapter_id: adapter.one
        version: 1.0.0
        signature_status: valid
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := len(mgr.EffectiveConfig().Adapter.Allowlist.Entries)
	if after != before {
		t.Fatalf("invalid adapter allowlist reload should rollback, entries len = %d, want %d", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerRuntimeOperationProfilesInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  operation_profiles:
    default_profile: legacy
    legacy:
      timeout: 3s
    interactive:
      timeout: 10s
    background:
      timeout: 30s
    batch:
      timeout: 2m
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Runtime.OperationProfiles.DefaultProfile
	if before != OperationProfileLegacy {
		t.Fatalf("before runtime.operation_profiles.default_profile = %q, want %q", before, OperationProfileLegacy)
	}

	writeConfig(t, file, `
runtime:
  operation_profiles:
    default_profile: realtime
    legacy:
      timeout: 3s
    interactive:
      timeout: 10s
    background:
      timeout: 30s
    batch:
      timeout: 2m
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Runtime.OperationProfiles.DefaultProfile
	if after != before {
		t.Fatalf(
			"invalid runtime.operation_profiles reload should rollback, default_profile = %q, want %q",
			after,
			before,
		)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func writeConfig(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("condition not met within %v", timeout)
}
