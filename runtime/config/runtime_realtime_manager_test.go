package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestManagerRuntimeRealtimeInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  realtime:
    protocol:
      enabled: true
      version: realtime_event_protocol.v1
      max_buffered_events: 512
    interrupt_resume:
      enabled: true
      resume_cursor_ttl_ms: 300000
      idempotency_window_ms: 120000
reload:
  enabled: true
  debounce: 20ms
`)

	mgr, err := NewManager(ManagerOptions{
		FilePath:        file,
		EnvPrefix:       "BAYMAX_A68_REALTIME_MANAGER_TEST",
		EnableHotReload: true,
	})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Runtime.Realtime
	if before.Protocol.Version != RuntimeRealtimeProtocolVersionV1 {
		t.Fatalf("before runtime.realtime.protocol.version = %q, want %q", before.Protocol.Version, RuntimeRealtimeProtocolVersionV1)
	}

	writeConfig(t, file, `
runtime:
  realtime:
    protocol:
      enabled: true
      version: realtime_event_protocol.v2
      max_buffered_events: 512
    interrupt_resume:
      enabled: true
      resume_cursor_ttl_ms: 300000
      idempotency_window_ms: 120000
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)

	after := mgr.EffectiveConfig().Runtime.Realtime
	if after.Protocol.Version != before.Protocol.Version {
		t.Fatalf(
			"invalid runtime.realtime.protocol.version should rollback, got %q want %q",
			after.Protocol.Version,
			before.Protocol.Version,
		)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerRuntimeRealtimeInvalidReloadRollsBackWithEnvPrecedence(t *testing.T) {
	t.Setenv("BAYMAX_A68_REALTIME_ENV_TEST_RUNTIME_REALTIME_PROTOCOL_MAX_BUFFERED_EVENTS", "777")
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  realtime:
    protocol:
      enabled: true
      version: realtime_event_protocol.v1
      max_buffered_events: 256
    interrupt_resume:
      enabled: true
      resume_cursor_ttl_ms: 300000
      idempotency_window_ms: 120000
reload:
  enabled: true
  debounce: 20ms
`)

	mgr, err := NewManager(ManagerOptions{
		FilePath:        file,
		EnvPrefix:       "BAYMAX_A68_REALTIME_ENV_TEST",
		EnableHotReload: true,
	})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Runtime.Realtime
	if before.Protocol.MaxBufferedEvents != 777 {
		t.Fatalf("env precedence max_buffered_events=%d, want 777", before.Protocol.MaxBufferedEvents)
	}

	writeConfig(t, file, `
runtime:
  realtime:
    protocol:
      enabled: true
      version: realtime_event_protocol.v1
      max_buffered_events: 256
    interrupt_resume:
      enabled: true
      resume_cursor_ttl_ms: 0
      idempotency_window_ms: 120000
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)

	after := mgr.EffectiveConfig().Runtime.Realtime
	if after.Protocol.MaxBufferedEvents != 777 {
		t.Fatalf("env-derived max_buffered_events should remain 777 after failed reload, got %d", after.Protocol.MaxBufferedEvents)
	}
	if after.InterruptResume.ResumeCursorTTLMS != before.InterruptResume.ResumeCursorTTLMS {
		t.Fatalf(
			"invalid runtime.realtime.interrupt_resume.resume_cursor_ttl_ms should rollback, got %d want %d",
			after.InterruptResume.ResumeCursorTTLMS,
			before.InterruptResume.ResumeCursorTTLMS,
		)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}
