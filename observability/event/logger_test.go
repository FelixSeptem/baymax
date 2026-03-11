package event

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

func TestJSONLoggerIncludesCorrelationFields(t *testing.T) {
	var b strings.Builder
	l := NewJSONLogger(&b)
	ev := types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.started",
		RunID:   "run-1",
		TraceID: "trace-1",
		SpanID:  "span-1",
		Time:    time.Unix(0, 0).UTC(),
	}
	l.OnEvent(context.Background(), ev)

	line := strings.TrimSpace(b.String())
	if line == "" {
		t.Fatal("expected log line")
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if got["run_id"] != "run-1" || got["trace_id"] != "trace-1" || got["span_id"] != "span-1" {
		t.Fatalf("missing correlation fields: %#v", got)
	}
}

func TestJSONLoggerWithRuntimeManagerAddsMetadata(t *testing.T) {
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

	var b strings.Builder
	l := NewJSONLoggerWithRuntimeManager(&b, mgr)
	ev := types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.started",
		RunID:   "run-1",
		Time:    time.Unix(0, 0).UTC(),
	}
	l.OnEvent(context.Background(), ev)
	var got map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(b.String())), &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if got["runtime_active_profile"] != "default" {
		t.Fatalf("runtime_active_profile = %#v, want default", got["runtime_active_profile"])
	}
}
