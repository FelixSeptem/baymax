package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestReadRuntimeContextHandoffConfigNormalizesValues(t *testing.T) {
	v := viper.New()
	v.Set("runtime.context.jit.compaction.handoff.enabled", true)
	v.Set("runtime.context.jit.compaction.handoff.max_serialized_bytes", 4096)
	v.Set("runtime.context.jit.compaction.handoff.max_latency_ms", 55)
	v.Set("runtime.context.jit.compaction.handoff.quality_threshold", 0.75)
	v.Set("runtime.context.jit.compaction.handoff.failure_policy", "FAIL_FAST")

	got, err := readRuntimeContextHandoffConfig(v)
	if err != nil {
		t.Fatalf("readRuntimeContextHandoffConfig() error = %v", err)
	}
	if !got.Enabled || got.MaxSerializedBytes != 4096 || got.MaxLatencyMS != 55 || got.QualityThreshold != 0.75 || got.FailurePolicy != RuntimeContextJITCompactionFallbackPolicyFailFast {
		t.Fatalf("handoff config = %#v", got)
	}
}
