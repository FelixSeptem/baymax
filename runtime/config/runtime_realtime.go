package config

import (
	"fmt"
	"strings"
)

const (
	RuntimeRealtimeProtocolVersionV1 = "realtime_event_protocol.v1"
)

type RuntimeRealtimeConfig struct {
	Protocol        RuntimeRealtimeProtocolConfig        `json:"protocol"`
	InterruptResume RuntimeRealtimeInterruptResumeConfig `json:"interrupt_resume"`
}

type RuntimeRealtimeProtocolConfig struct {
	Enabled           bool   `json:"enabled"`
	Version           string `json:"version"`
	MaxBufferedEvents int    `json:"max_buffered_events"`
}

type RuntimeRealtimeInterruptResumeConfig struct {
	Enabled             bool `json:"enabled"`
	ResumeCursorTTLMS   int  `json:"resume_cursor_ttl_ms"`
	IdempotencyWindowMS int  `json:"idempotency_window_ms"`
}

func normalizeRuntimeRealtimeConfig(in RuntimeRealtimeConfig) RuntimeRealtimeConfig {
	base := DefaultConfig().Runtime.Realtime
	out := in
	out.Protocol = normalizeRuntimeRealtimeProtocolConfig(out.Protocol)
	out.InterruptResume = normalizeRuntimeRealtimeInterruptResumeConfig(out.InterruptResume)
	if strings.TrimSpace(out.Protocol.Version) == "" {
		out.Protocol.Version = base.Protocol.Version
	}
	return out
}

func normalizeRuntimeRealtimeProtocolConfig(in RuntimeRealtimeProtocolConfig) RuntimeRealtimeProtocolConfig {
	base := DefaultConfig().Runtime.Realtime.Protocol
	out := in
	out.Version = strings.ToLower(strings.TrimSpace(out.Version))
	if out.Version == "" {
		out.Version = strings.ToLower(strings.TrimSpace(base.Version))
	}
	if out.MaxBufferedEvents <= 0 {
		out.MaxBufferedEvents = base.MaxBufferedEvents
	}
	return out
}

func normalizeRuntimeRealtimeInterruptResumeConfig(in RuntimeRealtimeInterruptResumeConfig) RuntimeRealtimeInterruptResumeConfig {
	base := DefaultConfig().Runtime.Realtime.InterruptResume
	out := in
	if out.ResumeCursorTTLMS <= 0 {
		out.ResumeCursorTTLMS = base.ResumeCursorTTLMS
	}
	if out.IdempotencyWindowMS <= 0 {
		out.IdempotencyWindowMS = base.IdempotencyWindowMS
	}
	return out
}

func ValidateRuntimeRealtimeConfig(cfg RuntimeRealtimeConfig) error {
	normalized := normalizeRuntimeRealtimeConfig(cfg)
	if cfg.Protocol.MaxBufferedEvents <= 0 {
		return fmt.Errorf("runtime.realtime.protocol.max_buffered_events must be > 0")
	}
	if cfg.InterruptResume.ResumeCursorTTLMS <= 0 {
		return fmt.Errorf("runtime.realtime.interrupt_resume.resume_cursor_ttl_ms must be > 0")
	}
	if cfg.InterruptResume.IdempotencyWindowMS <= 0 {
		return fmt.Errorf("runtime.realtime.interrupt_resume.idempotency_window_ms must be > 0")
	}
	switch normalized.Protocol.Version {
	case RuntimeRealtimeProtocolVersionV1:
	default:
		return fmt.Errorf(
			"runtime.realtime.protocol.version must be one of [%s], got %q",
			RuntimeRealtimeProtocolVersionV1,
			cfg.Protocol.Version,
		)
	}
	if cfg.InterruptResume.Enabled && !cfg.Protocol.Enabled {
		return fmt.Errorf(
			"runtime.realtime.interrupt_resume.enabled requires runtime.realtime.protocol.enabled=true",
		)
	}
	return nil
}
