package runtime

import (
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

type ProfileName string

const (
	ProfileDev            ProfileName = "dev"
	ProfileDefault        ProfileName = "default"
	ProfileHighThroughput ProfileName = "high-throughput"
	ProfileHighReliab     ProfileName = "high-reliability"
)

func ResolvePolicy(profile ProfileName, override *types.MCPRuntimePolicy) types.MCPRuntimePolicy {
	base := defaultPolicyFor(profile)
	return applyPolicyOverride(base, override)
}

func defaultPolicyFor(profile ProfileName) types.MCPRuntimePolicy {
	switch profile {
	case ProfileDev:
		return types.MCPRuntimePolicy{
			CallTimeout:   5 * time.Second,
			Retry:         0,
			Backoff:       20 * time.Millisecond,
			QueueSize:     16,
			Backpressure:  types.BackpressureBlock,
			ReadPoolSize:  2,
			WritePoolSize: 1,
		}
	case ProfileHighThroughput:
		return types.MCPRuntimePolicy{
			CallTimeout:   8 * time.Second,
			Retry:         1,
			Backoff:       20 * time.Millisecond,
			QueueSize:     128,
			Backpressure:  types.BackpressureReject,
			ReadPoolSize:  16,
			WritePoolSize: 2,
		}
	case ProfileHighReliab:
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
