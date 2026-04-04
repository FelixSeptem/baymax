package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeRealtimeConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Runtime.Realtime.Protocol.Enabled {
		t.Fatal("runtime.realtime.protocol.enabled = true, want false")
	}
	if cfg.Runtime.Realtime.Protocol.Version != RuntimeRealtimeProtocolVersionV1 {
		t.Fatalf(
			"runtime.realtime.protocol.version = %q, want %q",
			cfg.Runtime.Realtime.Protocol.Version,
			RuntimeRealtimeProtocolVersionV1,
		)
	}
	if cfg.Runtime.Realtime.Protocol.MaxBufferedEvents != 512 {
		t.Fatalf(
			"runtime.realtime.protocol.max_buffered_events = %d, want 512",
			cfg.Runtime.Realtime.Protocol.MaxBufferedEvents,
		)
	}
	if cfg.Runtime.Realtime.InterruptResume.Enabled {
		t.Fatal("runtime.realtime.interrupt_resume.enabled = true, want false")
	}
	if cfg.Runtime.Realtime.InterruptResume.ResumeCursorTTLMS != 300000 {
		t.Fatalf(
			"runtime.realtime.interrupt_resume.resume_cursor_ttl_ms = %d, want 300000",
			cfg.Runtime.Realtime.InterruptResume.ResumeCursorTTLMS,
		)
	}
	if cfg.Runtime.Realtime.InterruptResume.IdempotencyWindowMS != 120000 {
		t.Fatalf(
			"runtime.realtime.interrupt_resume.idempotency_window_ms = %d, want 120000",
			cfg.Runtime.Realtime.InterruptResume.IdempotencyWindowMS,
		)
	}
}

func TestRuntimeRealtimeConfigEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_REALTIME_PROTOCOL_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_REALTIME_PROTOCOL_VERSION", RuntimeRealtimeProtocolVersionV1)
	t.Setenv("BAYMAX_RUNTIME_REALTIME_PROTOCOL_MAX_BUFFERED_EVENTS", "1024")
	t.Setenv("BAYMAX_RUNTIME_REALTIME_INTERRUPT_RESUME_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_REALTIME_INTERRUPT_RESUME_RESUME_CURSOR_TTL_MS", "450000")
	t.Setenv("BAYMAX_RUNTIME_REALTIME_INTERRUPT_RESUME_IDEMPOTENCY_WINDOW_MS", "180000")

	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
runtime:
  realtime:
    protocol:
      enabled: false
      version: realtime_event_protocol.v1
      max_buffered_events: 64
    interrupt_resume:
      enabled: false
      resume_cursor_ttl_ms: 90000
      idempotency_window_ms: 60000
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !cfg.Runtime.Realtime.Protocol.Enabled {
		t.Fatal("runtime.realtime.protocol.enabled = false, want true from env")
	}
	if cfg.Runtime.Realtime.Protocol.Version != RuntimeRealtimeProtocolVersionV1 {
		t.Fatalf(
			"runtime.realtime.protocol.version = %q, want %q from env",
			cfg.Runtime.Realtime.Protocol.Version,
			RuntimeRealtimeProtocolVersionV1,
		)
	}
	if cfg.Runtime.Realtime.Protocol.MaxBufferedEvents != 1024 {
		t.Fatalf(
			"runtime.realtime.protocol.max_buffered_events = %d, want 1024 from env",
			cfg.Runtime.Realtime.Protocol.MaxBufferedEvents,
		)
	}
	if !cfg.Runtime.Realtime.InterruptResume.Enabled {
		t.Fatal("runtime.realtime.interrupt_resume.enabled = false, want true from env")
	}
	if cfg.Runtime.Realtime.InterruptResume.ResumeCursorTTLMS != 450000 {
		t.Fatalf(
			"runtime.realtime.interrupt_resume.resume_cursor_ttl_ms = %d, want 450000 from env",
			cfg.Runtime.Realtime.InterruptResume.ResumeCursorTTLMS,
		)
	}
	if cfg.Runtime.Realtime.InterruptResume.IdempotencyWindowMS != 180000 {
		t.Fatalf(
			"runtime.realtime.interrupt_resume.idempotency_window_ms = %d, want 180000 from env",
			cfg.Runtime.Realtime.InterruptResume.IdempotencyWindowMS,
		)
	}
}

func TestRuntimeRealtimeConfigValidationRejectsInvalidValues(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Runtime.Realtime.Protocol.MaxBufferedEvents = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.realtime.protocol.max_buffered_events") {
		t.Fatalf("expected runtime.realtime.protocol.max_buffered_events validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Realtime.Protocol.Enabled = true
	cfg.Runtime.Realtime.Protocol.Version = "v2"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.realtime.protocol.version") {
		t.Fatalf("expected runtime.realtime.protocol.version validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Realtime.InterruptResume.Enabled = true
	cfg.Runtime.Realtime.Protocol.Enabled = false
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.realtime.interrupt_resume.enabled requires runtime.realtime.protocol.enabled=true") {
		t.Fatalf("expected interrupt/resume compatibility validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Realtime.InterruptResume.ResumeCursorTTLMS = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.realtime.interrupt_resume.resume_cursor_ttl_ms") {
		t.Fatalf("expected runtime.realtime.interrupt_resume.resume_cursor_ttl_ms validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Realtime.InterruptResume.IdempotencyWindowMS = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.realtime.interrupt_resume.idempotency_window_ms") {
		t.Fatalf("expected runtime.realtime.interrupt_resume.idempotency_window_ms validation error, got %v", err)
	}
}

func TestRuntimeRealtimeConfigInvalidValuesFailFast(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_REALTIME_PROTOCOL_ENABLED", "not-a-bool")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.realtime.protocol.enabled") {
		t.Fatalf("expected strict bool parse error for runtime.realtime.protocol.enabled, got %v", err)
	}

	t.Setenv("BAYMAX_RUNTIME_REALTIME_PROTOCOL_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_REALTIME_PROTOCOL_MAX_BUFFERED_EVENTS", "NaN")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.realtime.protocol.max_buffered_events") {
		t.Fatalf("expected strict int parse error for runtime.realtime.protocol.max_buffered_events, got %v", err)
	}

	t.Setenv("BAYMAX_RUNTIME_REALTIME_PROTOCOL_MAX_BUFFERED_EVENTS", "64")
	t.Setenv("BAYMAX_RUNTIME_REALTIME_INTERRUPT_RESUME_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_REALTIME_INTERRUPT_RESUME_RESUME_CURSOR_TTL_MS", "bad-int")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.realtime.interrupt_resume.resume_cursor_ttl_ms") {
		t.Fatalf("expected strict int parse error for runtime.realtime.interrupt_resume.resume_cursor_ttl_ms, got %v", err)
	}
}
