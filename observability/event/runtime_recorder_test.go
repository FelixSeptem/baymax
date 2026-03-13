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
	ev := types.Event{
		Version:   types.EventSchemaVersionV1,
		Type:      "run.finished",
		Time:      time.Now(),
		RunID:     "run-1",
		Iteration: 2,
		Payload: map[string]any{
			"status":                "failed",
			"latency_ms":            int64(120),
			"tool_calls":            3,
			"error_class":           "ErrTool",
			"prefix_hash":           "abc123",
			"assemble_latency_ms":   int64(8),
			"assemble_status":       "success",
			"guard_violation":       "",
			"assemble_stage_status": "stage1_only",
			"stage2_skip_reason":    "routing.threshold.not_met",
			"stage1_latency_ms":     int64(3),
			"stage2_latency_ms":     int64(0),
			"stage2_provider":       "file",
			"stage2_hit_count":      2,
			"stage2_source":         "http",
			"stage2_reason":         "ok",
			"recap_status":          "appended",
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
