package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
)

func TestUnifiedQueryContractUnmatchedTaskIDEmptySet(t *testing.T) {
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

	base := time.Now()
	mgr.RecordRun(runtimediag.RunRecord{
		Time:       base.Add(1 * time.Second),
		RunID:      "run-a18-int-1",
		Status:     "success",
		TeamID:     "team-a",
		WorkflowID: "wf-a",
		TaskID:     "task-a18-int-1",
	})
	mgr.RecordRun(runtimediag.RunRecord{
		Time:       base.Add(2 * time.Second),
		RunID:      "run-a18-int-2",
		Status:     "failed",
		TeamID:     "team-a",
		WorkflowID: "wf-a",
		TaskID:     "task-a18-int-2",
	})

	result, err := mgr.QueryRuns(runtimediag.UnifiedRunQueryRequest{
		TaskID: "task-a18-int-missing",
	})
	if err != nil {
		t.Fatalf("query missing task_id should not error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("missing task_id should return empty set, got %#v", result.Items)
	}

	result, err = mgr.QueryRuns(runtimediag.UnifiedRunQueryRequest{
		TeamID:     "team-a",
		WorkflowID: "wf-a",
		Status:     "failed",
	})
	if err != nil {
		t.Fatalf("query AND filters failed: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].RunID != "run-a18-int-2" {
		t.Fatalf("AND query mismatch: %#v", result.Items)
	}
}

func TestUnifiedQueryContractReplayIdempotentSummaries(t *testing.T) {
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

	base := time.Now()
	record := runtimediag.RunRecord{
		Time:       base,
		RunID:      "run-a18-idempotent",
		Status:     "failed",
		LatencyMs:  10,
		ToolCalls:  1,
		TeamID:     "team-a",
		WorkflowID: "wf-a",
		TaskID:     "task-a18-idempotent",
	}
	mgr.RecordRun(record)

	first, err := mgr.QueryRuns(runtimediag.UnifiedRunQueryRequest{RunID: "run-a18-idempotent"})
	if err != nil {
		t.Fatalf("first query failed: %v", err)
	}
	if len(first.Items) != 1 {
		t.Fatalf("first query len = %d, want 1", len(first.Items))
	}

	record.LatencyMs = 99
	record.ToolCalls = 3
	mgr.RecordRun(record)
	second, err := mgr.QueryRuns(runtimediag.UnifiedRunQueryRequest{RunID: "run-a18-idempotent"})
	if err != nil {
		t.Fatalf("second query failed: %v", err)
	}
	if len(second.Items) != 1 {
		t.Fatalf("replayed query len = %d, want 1", len(second.Items))
	}
	if second.Items[0].LatencyMs != 99 || second.Items[0].ToolCalls != 3 {
		t.Fatalf("idempotent replay summary should keep latest logical record, got %#v", second.Items[0])
	}
}
