package config

import (
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

var (
	benchmarkRuntimeConfigDurationSink time.Duration
	benchmarkMCPPolicySink             types.MCPRuntimePolicy
)

func benchmarkNewManager(b *testing.B) *Manager {
	b.Helper()
	mgr, err := NewManager(ManagerOptions{})
	if err != nil {
		b.Fatalf("NewManager failed: %v", err)
	}
	b.Cleanup(func() { _ = mgr.Close() })
	return mgr
}

func BenchmarkRuntimeConfigReadPathEffectiveConfig(b *testing.B) {
	mgr := benchmarkNewManager(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg := mgr.EffectiveConfig()
		policy := cfg.MCP.Profiles[cfg.MCP.ActiveProfile]
		benchmarkRuntimeConfigDurationSink = policy.CallTimeout
	}
}

func BenchmarkRuntimeConfigReadPathEffectiveConfigRef(b *testing.B) {
	mgr := benchmarkNewManager(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg := mgr.EffectiveConfigRef()
		policy := cfg.MCP.Profiles[cfg.MCP.ActiveProfile]
		benchmarkRuntimeConfigDurationSink = policy.CallTimeout
	}
}

func BenchmarkMCPPolicyResolveDefaultProfileNoOverride(b *testing.B) {
	mgr := benchmarkNewManager(b)
	if _, err := mgr.ResolvePolicy(ProfileDefault, nil); err != nil {
		b.Fatalf("warm ResolvePolicy failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		policy, err := mgr.ResolvePolicy(ProfileDefault, nil)
		if err != nil {
			b.Fatalf("ResolvePolicy failed: %v", err)
		}
		benchmarkMCPPolicySink = policy
	}
}

func BenchmarkMCPPolicyResolveDefaultProfileWithOverride(b *testing.B) {
	mgr := benchmarkNewManager(b)
	override := &types.MCPRuntimePolicy{
		CallTimeout:   6 * time.Second,
		Retry:         3,
		Backoff:       25 * time.Millisecond,
		QueueSize:     64,
		Backpressure:  types.BackpressureReject,
		ReadPoolSize:  8,
		WritePoolSize: 2,
	}
	if _, err := mgr.ResolvePolicy(ProfileDefault, override); err != nil {
		b.Fatalf("warm ResolvePolicy failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		policy, err := mgr.ResolvePolicy(ProfileDefault, override)
		if err != nil {
			b.Fatalf("ResolvePolicy failed: %v", err)
		}
		benchmarkMCPPolicySink = policy
	}
}
