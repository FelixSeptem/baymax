package profile

import (
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

type Name = string

const (
	Dev            Name = "dev"
	Default        Name = "default"
	HighThroughput Name = "high-throughput"
	HighReliab     Name = "high-reliability"
)

func Resolve(profile Name, override *types.MCPRuntimePolicy) types.MCPRuntimePolicy {
	base := defaultPolicyFor(profile)
	return applyPolicyOverride(base, override)
}

func defaultPolicyFor(profile Name) types.MCPRuntimePolicy {
	switch profile {
	case Dev:
		return types.MCPRuntimePolicy{
			CallTimeout:   5 * time.Second,
			Retry:         0,
			Backoff:       20 * time.Millisecond,
			QueueSize:     16,
			Backpressure:  types.BackpressureBlock,
			ReadPoolSize:  2,
			WritePoolSize: 1,
		}
	case HighThroughput:
		return types.MCPRuntimePolicy{
			CallTimeout:   8 * time.Second,
			Retry:         1,
			Backoff:       20 * time.Millisecond,
			QueueSize:     128,
			Backpressure:  types.BackpressureReject,
			ReadPoolSize:  16,
			WritePoolSize: 2,
		}
	case HighReliab:
		return types.MCPRuntimePolicy{
			CallTimeout:   15 * time.Second,
			Retry:         3,
			Backoff:       80 * time.Millisecond,
			QueueSize:     64,
			Backpressure:  types.BackpressureBlock,
			ReadPoolSize:  8,
			WritePoolSize: 1,
		}
	default:
		return types.DefaultMCPRuntimePolicy()
	}
}

func applyPolicyOverride(base types.MCPRuntimePolicy, override *types.MCPRuntimePolicy) types.MCPRuntimePolicy {
	if override == nil {
		return base
	}
	if override.CallTimeout > 0 {
		base.CallTimeout = override.CallTimeout
	}
	if override.Retry >= 0 {
		base.Retry = override.Retry
	}
	if override.Backoff > 0 {
		base.Backoff = override.Backoff
	}
	if override.QueueSize > 0 {
		base.QueueSize = override.QueueSize
	}
	if override.Backpressure != "" {
		base.Backpressure = override.Backpressure
	}
	if override.ReadPoolSize > 0 {
		base.ReadPoolSize = override.ReadPoolSize
	}
	if override.WritePoolSize > 0 {
		base.WritePoolSize = override.WritePoolSize
	}
	return base
}
