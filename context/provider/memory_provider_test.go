package provider

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestMemoryProviderBackfillsFromLegacyFilePath(t *testing.T) {
	stage2Path := filepath.Join(t.TempDir(), "stage2.jsonl")
	content := strings.Join([]string{
		`{"session_id":"s1","content":"legacy-c1"}`,
		`{"session_id":"s1","content":"legacy-c2"}`,
	}, "\n")
	if err := os.WriteFile(stage2Path, []byte(content), 0o600); err != nil {
		t.Fatalf("write stage2 file failed: %v", err)
	}
	root := filepath.Join(t.TempDir(), "memory-store")
	p, err := NewWithConfig(Config{
		Name:     runtimeconfig.ContextStage2ProviderMemory,
		FilePath: stage2Path,
		Memory: runtimeconfig.RuntimeMemoryConfig{
			Mode: runtimeconfig.RuntimeMemoryModeBuiltinFilesystem,
			External: runtimeconfig.RuntimeMemoryExternalConfig{
				ContractVersion: runtimeconfig.RuntimeMemoryContractVersionV1,
			},
			Builtin: runtimeconfig.RuntimeMemoryBuiltinConfig{
				RootDir: root,
				Compaction: runtimeconfig.RuntimeMemoryBuiltinCompactionConfig{
					Enabled:     true,
					MinOps:      8,
					MaxWALBytes: 1024,
				},
			},
			Fallback: runtimeconfig.RuntimeMemoryFallbackConfig{
				Policy: runtimeconfig.RuntimeMemoryFallbackPolicyFailFast,
			},
		},
	})
	if err != nil {
		t.Fatalf("NewWithConfig failed: %v", err)
	}
	if mp, ok := p.(*memoryProvider); ok && mp.facade != nil {
		defer func() { _ = mp.facade.Close() }()
	}
	first, err := p.Fetch(context.Background(), Request{
		SessionID: "s1",
		MaxItems:  2,
	})
	if err != nil {
		t.Fatalf("first Fetch failed: %v", err)
	}
	if len(first.Chunks) != 2 {
		t.Fatalf("first fetch chunks = %#v, want 2 items", first.Chunks)
	}
	if first.Meta["source"] != "file" {
		t.Fatalf("first fetch source = %#v, want file", first.Meta["source"])
	}

	second, err := p.Fetch(context.Background(), Request{
		SessionID: "s1",
		MaxItems:  2,
	})
	if err != nil {
		t.Fatalf("second Fetch failed: %v", err)
	}
	if len(second.Chunks) != 2 {
		t.Fatalf("second fetch chunks = %#v, want 2 items", second.Chunks)
	}
	if second.Meta["source"] != "memory" {
		t.Fatalf("second fetch source = %#v, want memory", second.Meta["source"])
	}
	if second.Meta["memory_scope_selected"] == "" {
		t.Fatalf("memory_scope_selected should be populated, got %#v", second.Meta["memory_scope_selected"])
	}
	if _, ok := second.Meta["memory_budget_used"]; !ok {
		t.Fatalf("memory_budget_used should be populated, got %#v", second.Meta)
	}
	if _, ok := second.Meta["memory_hits"]; !ok {
		t.Fatalf("memory_hits should be populated, got %#v", second.Meta)
	}
	if _, ok := second.Meta["memory_rerank_stats"]; !ok {
		t.Fatalf("memory_rerank_stats should be populated, got %#v", second.Meta)
	}
}

func TestMemoryProviderExternalModeRequiresEndpoint(t *testing.T) {
	_, err := NewWithConfig(Config{
		Name: runtimeconfig.ContextStage2ProviderMemory,
		Memory: runtimeconfig.RuntimeMemoryConfig{
			Mode: runtimeconfig.RuntimeMemoryModeExternalSPI,
			External: runtimeconfig.RuntimeMemoryExternalConfig{
				Provider:        "mem0",
				Profile:         "mem0",
				ContractVersion: runtimeconfig.RuntimeMemoryContractVersionV1,
			},
			Builtin: runtimeconfig.RuntimeMemoryBuiltinConfig{
				RootDir: filepath.Join(t.TempDir(), "memory-store"),
				Compaction: runtimeconfig.RuntimeMemoryBuiltinCompactionConfig{
					Enabled:     true,
					MinOps:      8,
					MaxWALBytes: 1024,
				},
			},
			Fallback: runtimeconfig.RuntimeMemoryFallbackConfig{
				Policy: runtimeconfig.RuntimeMemoryFallbackPolicyFailFast,
			},
		},
	})
	if err == nil {
		t.Fatal("expected init error when external_spi mode lacks endpoint")
	}
	if !strings.Contains(err.Error(), "external endpoint") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMemoryProviderPassesGovernanceConfigToFacade(t *testing.T) {
	_, err := NewWithConfig(Config{
		Name: runtimeconfig.ContextStage2ProviderMemory,
		Memory: runtimeconfig.RuntimeMemoryConfig{
			Mode: runtimeconfig.RuntimeMemoryModeBuiltinFilesystem,
			External: runtimeconfig.RuntimeMemoryExternalConfig{
				ContractVersion: runtimeconfig.RuntimeMemoryContractVersionV1,
			},
			Builtin: runtimeconfig.RuntimeMemoryBuiltinConfig{
				RootDir: filepath.Join(t.TempDir(), "memory-store"),
				Compaction: runtimeconfig.RuntimeMemoryBuiltinCompactionConfig{
					Enabled:     true,
					MinOps:      8,
					MaxWALBytes: 1024,
				},
			},
			Fallback: runtimeconfig.RuntimeMemoryFallbackConfig{
				Policy: runtimeconfig.RuntimeMemoryFallbackPolicyFailFast,
			},
			Scope: runtimeconfig.RuntimeMemoryScopeConfig{
				Default:         "invalid-scope",
				Allowed:         []string{runtimeconfig.RuntimeMemoryScopeSession},
				AllowOverride:   true,
				GlobalNamespace: "global",
			},
		},
	})
	if err == nil {
		t.Fatal("expected NewWithConfig to fail for invalid runtime.memory.scope.default")
	}
	if !strings.Contains(err.Error(), "scope.default") {
		t.Fatalf("unexpected error for scope governance validation: %v", err)
	}
}
