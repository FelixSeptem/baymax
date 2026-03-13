package config

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
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
