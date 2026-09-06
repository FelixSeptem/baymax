package config

import (
	"strings"

	"github.com/spf13/viper"
)

func registerRuntimeContextHandoffDefaults(v *viper.Viper, base Config) {
	v.SetDefault("runtime.context.jit.compaction.handoff.enabled", base.Runtime.Context.JIT.Compaction.Handoff.Enabled)
	v.SetDefault("runtime.context.jit.compaction.handoff.max_serialized_bytes", base.Runtime.Context.JIT.Compaction.Handoff.MaxSerializedBytes)
	v.SetDefault("runtime.context.jit.compaction.handoff.max_latency_ms", base.Runtime.Context.JIT.Compaction.Handoff.MaxLatencyMS)
	v.SetDefault("runtime.context.jit.compaction.handoff.quality_threshold", base.Runtime.Context.JIT.Compaction.Handoff.QualityThreshold)
	v.SetDefault("runtime.context.jit.compaction.handoff.failure_policy", base.Runtime.Context.JIT.Compaction.Handoff.FailurePolicy)
}

func readRuntimeContextHandoffConfig(v *viper.Viper) (RuntimeContextJITHandoffConfig, error) {
	enabled, err := strictBoolConfigValue(v, "runtime.context.jit.compaction.handoff.enabled")
	if err != nil {
		return RuntimeContextJITHandoffConfig{}, err
	}
	maxSerializedBytes, err := strictIntConfigValue(v, "runtime.context.jit.compaction.handoff.max_serialized_bytes")
	if err != nil {
		return RuntimeContextJITHandoffConfig{}, err
	}
	maxLatencyMS, err := strictIntConfigValue(v, "runtime.context.jit.compaction.handoff.max_latency_ms")
	if err != nil {
		return RuntimeContextJITHandoffConfig{}, err
	}
	qualityThreshold, err := strictFloatConfigValue(v, "runtime.context.jit.compaction.handoff.quality_threshold")
	if err != nil {
		return RuntimeContextJITHandoffConfig{}, err
	}
	return RuntimeContextJITHandoffConfig{
		Enabled:            enabled,
		MaxSerializedBytes: maxSerializedBytes,
		MaxLatencyMS:       maxLatencyMS,
		QualityThreshold:   qualityThreshold,
		FailurePolicy:      strings.ToLower(strings.TrimSpace(v.GetString("runtime.context.jit.compaction.handoff.failure_policy"))),
	}, nil
}

func readRuntimeContextJITCompactionConfig(v *viper.Viper) (RuntimeContextJITCompactionConfig, error) {
	qualityThreshold, err := strictFloatConfigValue(v, "runtime.context.jit.compaction.quality_threshold")
	if err != nil {
		return RuntimeContextJITCompactionConfig{}, err
	}
	allowOldestToolResult, err := strictBoolConfigValue(v, "runtime.context.jit.compaction.rule_eligibility.allow_oldest_tool_result")
	if err != nil {
		return RuntimeContextJITCompactionConfig{}, err
	}
	minRetainedEvidence, err := strictIntConfigValue(v, "runtime.context.jit.compaction.rule_eligibility.min_retained_evidence")
	if err != nil {
		return RuntimeContextJITCompactionConfig{}, err
	}
	handoff, err := readRuntimeContextHandoffConfig(v)
	if err != nil {
		return RuntimeContextJITCompactionConfig{}, err
	}
	return RuntimeContextJITCompactionConfig{
		QualityThreshold: qualityThreshold,
		FallbackPolicy:   strings.ToLower(strings.TrimSpace(v.GetString("runtime.context.jit.compaction.fallback_policy"))),
		RuleEligibility: RuntimeContextJITCompactionRuleEligibility{
			AllowOldestToolResult: allowOldestToolResult,
			MinRetainedEvidence:   minRetainedEvidence,
		},
		Handoff: handoff,
	}, nil
}
