package config

import (
	"encoding/json"
	"testing"
	"time"
)

func TestResolveOperationTimeoutPrecedence(t *testing.T) {
	cfg := DefaultConfig()

	result, err := ResolveOperationTimeout(cfg, TimeoutResolutionInput{})
	if err != nil {
		t.Fatalf("resolve default timeout failed: %v", err)
	}
	if result.EffectiveProfile != OperationProfileLegacy {
		t.Fatalf("effective profile = %q, want %q", result.EffectiveProfile, OperationProfileLegacy)
	}
	if result.Source != TimeoutResolutionSourceProfile {
		t.Fatalf("resolution source = %q, want %q", result.Source, TimeoutResolutionSourceProfile)
	}
	if result.EffectiveTimeout != cfg.Runtime.OperationProfiles.Legacy.Timeout {
		t.Fatalf("effective timeout = %v, want %v", result.EffectiveTimeout, cfg.Runtime.OperationProfiles.Legacy.Timeout)
	}

	result, err = ResolveOperationTimeout(cfg, TimeoutResolutionInput{
		RequestedProfile: OperationProfileInteractive,
	})
	if err != nil {
		t.Fatalf("resolve interactive profile timeout failed: %v", err)
	}
	if result.EffectiveProfile != OperationProfileInteractive {
		t.Fatalf("effective profile = %q, want %q", result.EffectiveProfile, OperationProfileInteractive)
	}
	if result.Source != TimeoutResolutionSourceProfile {
		t.Fatalf("resolution source = %q, want %q", result.Source, TimeoutResolutionSourceProfile)
	}
	if result.EffectiveTimeout != cfg.Runtime.OperationProfiles.Interactive.Timeout {
		t.Fatalf(
			"effective timeout = %v, want %v",
			result.EffectiveTimeout,
			cfg.Runtime.OperationProfiles.Interactive.Timeout,
		)
	}

	result, err = ResolveOperationTimeout(cfg, TimeoutResolutionInput{
		RequestedProfile: OperationProfileInteractive,
		DomainTimeout:    12 * time.Second,
	})
	if err != nil {
		t.Fatalf("resolve with domain override failed: %v", err)
	}
	if result.Source != TimeoutResolutionSourceDomain {
		t.Fatalf("resolution source = %q, want %q", result.Source, TimeoutResolutionSourceDomain)
	}
	if result.EffectiveTimeout != 12*time.Second {
		t.Fatalf("effective timeout = %v, want 12s", result.EffectiveTimeout)
	}

	result, err = ResolveOperationTimeout(cfg, TimeoutResolutionInput{
		RequestedProfile: OperationProfileInteractive,
		DomainTimeout:    12 * time.Second,
		RequestTimeout:   2 * time.Second,
	})
	if err != nil {
		t.Fatalf("resolve with request override failed: %v", err)
	}
	if result.Source != TimeoutResolutionSourceRequest {
		t.Fatalf("resolution source = %q, want %q", result.Source, TimeoutResolutionSourceRequest)
	}
	if result.EffectiveTimeout != 2*time.Second {
		t.Fatalf("effective timeout = %v, want 2s", result.EffectiveTimeout)
	}
}

func TestResolveOperationTimeoutRejectsUnsupportedProfile(t *testing.T) {
	cfg := DefaultConfig()
	if _, err := ResolveOperationTimeout(cfg, TimeoutResolutionInput{
		RequestedProfile: "realtime",
	}); err == nil {
		t.Fatal("expected unsupported operation profile to fail")
	}
}

func TestResolveOperationTimeoutTraceStableKeys(t *testing.T) {
	cfg := DefaultConfig()
	result, err := ResolveOperationTimeout(cfg, TimeoutResolutionInput{
		RequestedProfile: OperationProfileBackground,
		DomainTimeout:    18 * time.Second,
		RequestTimeout:   4 * time.Second,
	})
	if err != nil {
		t.Fatalf("resolve timeout failed: %v", err)
	}
	var trace TimeoutResolutionTrace
	if err := json.Unmarshal([]byte(result.Trace), &trace); err != nil {
		t.Fatalf("unmarshal trace failed: %v", err)
	}
	if trace.Version != "v1" {
		t.Fatalf("trace.version = %q, want v1", trace.Version)
	}
	if trace.Profile != OperationProfileBackground {
		t.Fatalf("trace.profile = %q, want %q", trace.Profile, OperationProfileBackground)
	}
	if trace.SelectedSource != TimeoutResolutionSourceRequest {
		t.Fatalf("trace.selected_source = %q, want %q", trace.SelectedSource, TimeoutResolutionSourceRequest)
	}
	if trace.EffectiveTimeoutMs != (4 * time.Second).Milliseconds() {
		t.Fatalf("trace.effective_timeout_ms = %d, want %d", trace.EffectiveTimeoutMs, (4 * time.Second).Milliseconds())
	}
}
