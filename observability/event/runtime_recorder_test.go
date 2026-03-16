package event

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestRuntimeRecorderRecordsSkillEvents(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := NewRuntimeRecorder(mgr)
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "skill.loaded",
		Time:    time.Now(),
		RunID:   "run-1",
		Payload: map[string]any{"name": "skill-a"},
	})
	items := mgr.RecentSkills(1)
	if len(items) != 1 {
		t.Fatalf("skills len = %d, want 1", len(items))
	}
	if items[0].SkillName != "skill-a" || items[0].Status != "success" {
		t.Fatalf("unexpected skill record: %#v", items[0])
	}
}

func TestRuntimeRecorderRecordsRunFinishedAndDedup(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := NewRuntimeRecorder(mgr)
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		Time:    time.Now(),
		RunID:   "run-1",
		Payload: map[string]any{
			"phase":    "model",
			"status":   "running",
			"sequence": int64(1),
		},
	})
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		Time:    time.Now().Add(10 * time.Millisecond),
		RunID:   "run-1",
		Payload: map[string]any{
			"phase":    "model",
			"status":   "failed",
			"sequence": int64(2),
		},
	})
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		Time:    time.Now().Add(10 * time.Millisecond),
		RunID:   "run-1",
		Payload: map[string]any{
			"phase":    "model",
			"status":   "failed",
			"sequence": int64(2),
		},
	})
	ev := types.Event{
		Version:   types.EventSchemaVersionV1,
		Type:      "run.finished",
		Time:      time.Now(),
		RunID:     "run-1",
		Iteration: 2,
		Payload: map[string]any{
			"status":                  "failed",
			"latency_ms":              int64(120),
			"tool_calls":              3,
			"error_class":             "ErrTool",
			"prefix_hash":             "abc123",
			"assemble_latency_ms":     int64(8),
			"assemble_status":         "success",
			"guard_violation":         "",
			"assemble_stage_status":   "stage1_only",
			"stage2_skip_reason":      "routing.threshold.not_met",
			"stage1_latency_ms":       int64(3),
			"stage2_latency_ms":       int64(0),
			"stage2_provider":         "file",
			"stage2_profile":          "http_generic",
			"stage2_hit_count":        2,
			"stage2_source":           "http",
			"stage2_reason":           "ok",
			"stage2_reason_code":      "ok",
			"stage2_error_layer":      "",
			"ca3_pressure_zone":       "warning",
			"ca3_pressure_reason":     "usage_percent_trigger",
			"ca3_pressure_trigger":    "warning",
			"ca3_zone_residency_ms":   map[string]any{"safe": float64(12), "warning": float64(8)},
			"ca3_trigger_counts":      map[string]any{"warning": float64(2)},
			"ca3_compression_ratio":   0.42,
			"ca3_spill_count":         1,
			"ca3_swap_back_count":     1,
			"recap_status":            "appended",
			"gate_checks":             4,
			"gate_denied_count":       2,
			"gate_timeout_count":      1,
			"gate_rule_hit_count":     2,
			"gate_rule_last_id":       "allow-echoloop",
			"await_count":             2,
			"resume_count":            1,
			"cancel_by_user_count":    1,
			"cancel_propagated_count": 3,
			"backpressure_drop_count": 0,
			"inflight_peak":           8,
		},
	}
	rec.OnEvent(context.Background(), ev)
	rec.OnEvent(context.Background(), ev)

	items := mgr.RecentRuns(10)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	if items[0].Status != "failed" || items[0].ErrorClass != "ErrTool" || items[0].ToolCalls != 3 {
		t.Fatalf("unexpected run record: %#v", items[0])
	}
	if items[0].PrefixHash != "abc123" || items[0].AssembleLatencyMs != 8 || items[0].AssembleStatus != "success" {
		t.Fatalf("assembler fields mismatch: %#v", items[0])
	}
	if items[0].AssembleStageStatus != "stage1_only" || items[0].Stage2SkipReason == "" || items[0].RecapStatus != "appended" {
		t.Fatalf("ca2 fields mismatch: %#v", items[0])
	}
	if items[0].Stage2HitCount != 2 || items[0].Stage2Source != "http" || items[0].Stage2Reason != "ok" {
		t.Fatalf("ca2 retrieval fields mismatch: %#v", items[0])
	}
	if items[0].Stage2ReasonCode != "ok" || items[0].Stage2Profile != "http_generic" {
		t.Fatalf("ca2 retrieval extended fields mismatch: %#v", items[0])
	}
	if items[0].CA3PressureZone != "warning" || items[0].CA3PressureReason == "" {
		t.Fatalf("ca3 fields mismatch: %#v", items[0])
	}
	if items[0].CA3PressureTrigger != "warning" {
		t.Fatalf("ca3 trigger mismatch: %#v", items[0])
	}
	if items[0].CA3CompressionRatio == 0 || items[0].CA3SpillCount != 1 || items[0].CA3SwapBackCount != 1 {
		t.Fatalf("ca3 metrics mismatch: %#v", items[0])
	}
	if items[0].GateChecks != 4 || items[0].GateDeniedCount != 2 || items[0].GateTimeoutCount != 1 {
		t.Fatalf("action gate metrics mismatch: %#v", items[0])
	}
	if items[0].GateRuleHitCount != 2 || items[0].GateRuleLastID != "allow-echoloop" {
		t.Fatalf("action gate rule metrics mismatch: %#v", items[0])
	}
	if items[0].AwaitCount != 2 || items[0].ResumeCount != 1 || items[0].CancelByUserCount != 1 {
		t.Fatalf("clarification metrics mismatch: %#v", items[0])
	}
	if items[0].CancelPropagated != 3 || items[0].BackpressureDrop != 0 || items[0].InflightPeak != 8 {
		t.Fatalf("concurrency metrics mismatch: %#v", items[0])
	}
	modelAgg, ok := items[0].TimelinePhases["model"]
	if !ok {
		t.Fatalf("timeline model aggregate missing: %#v", items[0].TimelinePhases)
	}
	if modelAgg.CountTotal != 1 || modelAgg.FailedTotal != 1 {
		t.Fatalf("timeline model aggregate mismatch: %#v", modelAgg)
	}
}

func TestRuntimeRecorderRedactsSensitivePayload(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
security:
  redaction:
    enabled: true
    strategy: keyword
    keywords: [token]
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := NewRuntimeRecorder(mgr)
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "skill.loaded",
		Time:    time.Now(),
		RunID:   "run-1",
		Payload: map[string]any{"name": "skill-a", "access_token": "secret"},
	})
	items := mgr.RecentSkills(1)
	if len(items) != 1 {
		t.Fatalf("skills len = %d, want 1", len(items))
	}
	if items[0].Payload["access_token"] != "***" {
		t.Fatalf("access_token should be masked, got %#v", items[0].Payload["access_token"])
	}
}

func TestRuntimeRecorderIgnoresActionTimelineForRunAggregation(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := NewRuntimeRecorder(mgr)
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		RunID:   "run-1",
		Time:    time.Now(),
		Payload: map[string]any{
			"phase":    "run",
			"status":   "running",
			"sequence": int64(1),
		},
	})
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		RunID:   "run-1",
		Time:    time.Now().Add(15 * time.Millisecond),
		Payload: map[string]any{
			"phase":    "run",
			"status":   "success", // invalid status should be ignored
			"sequence": int64(2),
		},
	})
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   "run-1",
		Time:    time.Now(),
		Payload: map[string]any{
			"status":     "success",
			"latency_ms": int64(9),
		},
	})

	items := mgr.RecentRuns(5)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	if items[0].RunID != "run-1" || items[0].Status != "success" {
		t.Fatalf("unexpected run record: %#v", items[0])
	}
	runAgg := items[0].TimelinePhases["run"]
	if runAgg.CountTotal != 0 {
		t.Fatalf("invalid timeline status should not be aggregated: %#v", runAgg)
	}
}

func TestRuntimeRecorderTracksCancelPropagationAcrossMCPAndSkillPhases(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	rec := NewRuntimeRecorder(mgr)
	now := time.Now()
	for _, phase := range []string{"mcp", "skill"} {
		rec.OnEvent(context.Background(), types.Event{
			Version: types.EventSchemaVersionV1,
			Type:    types.EventTypeActionTimeline,
			RunID:   "run-phase-cancel",
			Time:    now,
			Payload: map[string]any{
				"phase":    phase,
				"status":   "running",
				"sequence": int64(len(phase)),
			},
		})
		rec.OnEvent(context.Background(), types.Event{
			Version: types.EventSchemaVersionV1,
			Type:    types.EventTypeActionTimeline,
			RunID:   "run-phase-cancel",
			Time:    now.Add(10 * time.Millisecond),
			Payload: map[string]any{
				"phase":    phase,
				"status":   "canceled",
				"reason":   "cancel.propagated",
				"sequence": int64(len(phase) + 100),
			},
		})
	}
	rec.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   "run-phase-cancel",
		Time:    now.Add(20 * time.Millisecond),
		Payload: map[string]any{
			"status":                  "failed",
			"error_class":             "ErrPolicyTimeout",
			"cancel_propagated_count": 2,
		},
	})

	items := mgr.RecentRuns(1)
	if len(items) != 1 {
		t.Fatalf("run records len = %d, want 1", len(items))
	}
	if items[0].CancelPropagated != 2 {
		t.Fatalf("cancel_propagated_count = %d, want 2", items[0].CancelPropagated)
	}
	mcpAgg := items[0].TimelinePhases["mcp"]
	skillAgg := items[0].TimelinePhases["skill"]
	if mcpAgg.CanceledTotal != 1 || skillAgg.CanceledTotal != 1 {
		t.Fatalf("mcp/skill cancel aggregates mismatch: mcp=%#v skill=%#v", mcpAgg, skillAgg)
	}
}
