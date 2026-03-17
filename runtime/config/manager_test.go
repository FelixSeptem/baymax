package config

import (
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
