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
