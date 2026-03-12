package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/context/assembler"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/observability/event"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestSecurityRedactionAcrossDiagnosticsEventAndAssembler(t *testing.T) {
	root := t.TempDir()
	cfgPath := filepath.Join(root, "runtime.yaml")
	journalPath := filepath.Join(root, "journal.jsonl")
	stage2Path := filepath.Join(root, "stage2.jsonl")
	if err := os.WriteFile(stage2Path, []byte(`{"session_id":"session-1","content":"{\"api_key\":\"stage2-secret\"}"}`), 0o600); err != nil {
		t.Fatalf("write stage2 fixture: %v", err)
	}
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
context_assembler:
  enabled: true
  journal_path: ` + journalPath + `
  prefix_version: ca1
  storage:
    backend: file
  guard:
    fail_fast: true
  ca2:
    enabled: true
    routing_mode: rules
    stage_policy:
      stage1: fail_fast
      stage2: fail_fast
    timeout:
      stage1: 80ms
      stage2: 120ms
    stage2:
      provider: file
      file_path: ` + stage2Path + `
    routing:
      min_input_chars: 1
      trigger_keywords: [lookup]
      require_system_guard: false
    tail_recap:
      enabled: true
      max_items: 4
      max_field_chars: 256
security:
  redaction:
    enabled: true
    strategy: keyword
    keywords: [token,api_key]
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	// diagnostics path through runtime recorder
	recorder := event.NewRuntimeRecorder(mgr)
	recorder.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "skill.loaded",
		Time:    time.Now(),
		RunID:   "run-1",
		Payload: map[string]any{"name": "skill-a", "access_token": "diag-secret"},
	})
	skills := mgr.RecentSkills(1)
	if len(skills) != 1 {
		t.Fatalf("skills len = %d, want 1", len(skills))
	}
	if skills[0].Payload["access_token"] != "***" {
		t.Fatalf("diagnostics payload not redacted: %#v", skills[0].Payload)
	}

	// event output path through json logger
	var b strings.Builder
	logger := event.NewJSONLoggerWithRuntimeManager(&b, mgr)
	logger.OnEvent(context.Background(), types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "skill.loaded",
		Time:    time.Now(),
		RunID:   "run-1",
		Payload: map[string]any{"api_key": "event-secret"},
	})
	logLine := strings.TrimSpace(b.String())
	var entry map[string]any
	if err := json.Unmarshal([]byte(logLine), &entry); err != nil {
		t.Fatalf("invalid logger output: %v", err)
	}
	payload, _ := entry["payload"].(map[string]any)
	if payload["api_key"] != "***" {
		t.Fatalf("event payload not redacted: %#v", payload)
	}

	// context assembler path
	a := assembler.New(
		func() runtimeconfig.ContextAssemblerConfig { return mgr.EffectiveConfig().ContextAssembler },
		assembler.WithRedactionConfigProvider(func() runtimeconfig.SecurityRedactionConfig {
			return mgr.EffectiveConfig().Security.Redaction
		}),
	)
	outReq, result, err := a.Assemble(context.Background(), types.ContextAssembleRequest{
		RunID:         "run-1",
		SessionID:     "session-1",
		PrefixVersion: "ca1",
		Input:         "lookup context",
		Messages:      []types.Message{{Role: "system", Content: "s"}},
	}, types.ModelRequest{
		RunID:    "run-1",
		Input:    "lookup context",
		Messages: []types.Message{{Role: "system", Content: "s"}},
	})
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if result.Stage.Status != types.AssembleStageStatusStage2Used {
		t.Fatalf("stage status = %q, want stage2_used", result.Stage.Status)
	}
	joined := strings.Join(extractMessageContents(outReq.Messages), "\n")
	if strings.Contains(joined, "stage2-secret") {
		t.Fatalf("assembler output leaked sensitive value: %s", joined)
	}
	if !strings.Contains(joined, `"api_key":"***"`) {
		t.Fatalf("assembler output missing redacted field: %s", joined)
	}
}

func extractMessageContents(messages []types.Message) []string {
	out := make([]string, 0, len(messages))
	for _, msg := range messages {
		out = append(out, msg.Content)
	}
	return out
}
