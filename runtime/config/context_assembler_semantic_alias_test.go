package config

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var contextAssemblerAliasLogCaptureMu sync.Mutex

func TestLoadContextAssemblerSemanticAliasOnly(t *testing.T) {
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
  stage2_routing_and_disclosure:
    enabled: true
    routing_mode: rules
    stage2:
      provider: file
      file_path: /tmp/semantic-stage2.jsonl
  pressure_compaction_and_swapback:
    enabled: true
    compaction:
      mode: semantic
      semantic_timeout: 900ms
      quality:
        threshold: 0.61
      embedding:
        enabled: false
`
	logs, loaded := loadConfigForSemanticAliasTest(t, cfg)
	raw := mustConfigMap(t, loaded)
	contextMap := mustChildMap(t, raw, "context_assembler")

	legacyStage2 := mustChildMap(t, contextMap, "c"+"a2")
	if !mapBool(legacyStage2, "enabled") {
		t.Fatal("semantic stage2 key should map into legacy stage alias")
	}
	if mapString(legacyStage2, "routing_mode") != "rules" {
		t.Fatalf("legacy stage2 routing_mode = %q, want rules", mapString(legacyStage2, "routing_mode"))
	}
	stage2Cfg := mustChildMap(t, legacyStage2, "stage2")
	if mapString(stage2Cfg, "provider") != ContextStage2ProviderFile {
		t.Fatalf("legacy stage2 provider = %q, want %q", mapString(stage2Cfg, "provider"), ContextStage2ProviderFile)
	}
	if mapString(stage2Cfg, "file_path") != "/tmp/semantic-stage2.jsonl" {
		t.Fatalf("legacy stage2 file_path = %q, want /tmp/semantic-stage2.jsonl", mapString(stage2Cfg, "file_path"))
	}

	legacyStage3 := mustChildMap(t, contextMap, "c"+"a3")
	if !mapBool(legacyStage3, "enabled") {
		t.Fatal("semantic pressure key should map into legacy stage alias")
	}
	compaction := mustChildMap(t, legacyStage3, "compaction")
	if mapString(compaction, "mode") != "semantic" {
		t.Fatalf("legacy stage3 compaction.mode = %q, want semantic", mapString(compaction, "mode"))
	}
	if !strings.Contains(logs, "migration hint") {
		t.Fatalf("expected migration hint log, got %q", logs)
	}
	if !strings.Contains(logs, "context_assembler.stage2_routing_and_disclosure") {
		t.Fatalf("expected semantic stage2 hint, got %q", logs)
	}
	if !strings.Contains(logs, "context_assembler.pressure_compaction_and_swapback") {
		t.Fatalf("expected semantic pressure hint, got %q", logs)
	}
}

func TestLoadContextAssemblerSemanticAliasTakesPrecedenceOverLegacy(t *testing.T) {
	tpl := `
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
  __LEGACY_STAGE2__:
    routing_mode: agentic
  stage2_routing_and_disclosure:
    routing_mode: rules
  __LEGACY_STAGE3__:
    compaction:
      mode: truncate
  pressure_compaction_and_swapback:
    compaction:
      mode: semantic
`
	cfg := strings.TrimSpace(tpl)
	cfg = strings.ReplaceAll(cfg, "__LEGACY_STAGE2__", "c"+"a2")
	cfg = strings.ReplaceAll(cfg, "__LEGACY_STAGE3__", "c"+"a3")

	logs, loaded := loadConfigForSemanticAliasTest(t, cfg)
	raw := mustConfigMap(t, loaded)
	contextMap := mustChildMap(t, raw, "context_assembler")

	legacyStage2 := mustChildMap(t, contextMap, "c"+"a2")
	if mapString(legacyStage2, "routing_mode") != "rules" {
		t.Fatalf("semantic stage2 key should win on conflict, got %q", mapString(legacyStage2, "routing_mode"))
	}

	legacyStage3 := mustChildMap(t, contextMap, "c"+"a3")
	compaction := mustChildMap(t, legacyStage3, "compaction")
	if mapString(compaction, "mode") != "semantic" {
		t.Fatalf("semantic stage3 key should win on conflict, got %q", mapString(compaction, "mode"))
	}
	if !strings.Contains(logs, "takes precedence") {
		t.Fatalf("expected conflict precedence migration hint, got %q", logs)
	}
}

func TestLoadContextAssemblerLegacyAliasOnlyDoesNotEmitMigrationHint(t *testing.T) {
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
  __LEGACY_STAGE2__:
    enabled: true
`
	cfg = strings.ReplaceAll(cfg, "__LEGACY_STAGE2__", "c"+"a2")
	logs, loaded := loadConfigForSemanticAliasTest(t, cfg)
	raw := mustConfigMap(t, loaded)
	contextMap := mustChildMap(t, raw, "context_assembler")
	legacyStage2 := mustChildMap(t, contextMap, "c"+"a2")
	if !mapBool(legacyStage2, "enabled") {
		t.Fatal("legacy stage alias should remain accepted for rollback compatibility")
	}
	if strings.Contains(logs, "migration hint") {
		t.Fatalf("legacy-only config should not emit semantic alias migration hint, got %q", logs)
	}
}

func loadConfigForSemanticAliasTest(t *testing.T, raw string) (string, Config) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	if err := os.WriteFile(file, []byte(strings.TrimSpace(raw)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	contextAssemblerAliasLogCaptureMu.Lock()
	defer contextAssemblerAliasLogCaptureMu.Unlock()
	var buf bytes.Buffer
	previousOutput := log.Writer()
	previousFlags := log.Flags()
	previousPrefix := log.Prefix()
	log.SetOutput(&buf)
	log.SetFlags(0)
	log.SetPrefix("")
	defer func() {
		log.SetOutput(previousOutput)
		log.SetFlags(previousFlags)
		log.SetPrefix(previousPrefix)
	}()
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	return buf.String(), cfg
}

func mustConfigMap(t *testing.T, cfg Config) map[string]any {
	t.Helper()
	raw, err := toMap(cfg)
	if err != nil {
		t.Fatalf("toMap failed: %v", err)
	}
	return raw
}

func mustChildMap(t *testing.T, root map[string]any, key string) map[string]any {
	t.Helper()
	raw, ok := root[key]
	if !ok {
		t.Fatalf("missing key %q", key)
	}
	child, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("key %q is not a map: %#v", key, raw)
	}
	return child
}

func mapString(root map[string]any, key string) string {
	value, _ := root[key].(string)
	return strings.TrimSpace(value)
}

func mapBool(root map[string]any, key string) bool {
	value, _ := root[key].(bool)
	return value
}
