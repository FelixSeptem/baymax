package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type RuntimeContextConfig struct {
	JIT RuntimeContextJITConfig `json:"jit"`
}

type RuntimeContextJITConfig struct {
	ReferenceFirst   RuntimeContextJITReferenceFirstConfig   `json:"reference_first"`
	IsolateHandoff   RuntimeContextJITIsolateHandoffConfig   `json:"isolate_handoff"`
	EditGate         RuntimeContextJITEditGateConfig         `json:"edit_gate"`
	SwapBack         RuntimeContextJITSwapBackConfig         `json:"swap_back"`
	LifecycleTiering RuntimeContextJITLifecycleTieringConfig `json:"lifecycle_tiering"`
}

type RuntimeContextJITReferenceFirstConfig struct {
	Enabled          bool `json:"enabled"`
	MaxRefs          int  `json:"max_refs"`
	MaxResolveTokens int  `json:"max_resolve_tokens"`
}

type RuntimeContextJITIsolateHandoffConfig struct {
	Enabled       bool    `json:"enabled"`
	DefaultTTLMS  int     `json:"default_ttl_ms"`
	MinConfidence float64 `json:"min_confidence"`
}

type RuntimeContextJITEditGateConfig struct {
	Enabled            bool    `json:"enabled"`
	ClearAtLeastTokens int     `json:"clear_at_least_tokens"`
	MinGainRatio       float64 `json:"min_gain_ratio"`
}

type RuntimeContextJITSwapBackConfig struct {
	Enabled           bool    `json:"enabled"`
	MinRelevanceScore float64 `json:"min_relevance_score"`
}

type RuntimeContextJITLifecycleTieringConfig struct {
	Enabled   bool `json:"enabled"`
	HotTTLMS  int  `json:"hot_ttl_ms"`
	WarmTTLMS int  `json:"warm_ttl_ms"`
	ColdTTLMS int  `json:"cold_ttl_ms"`
}

func ValidateRuntimeContextConfig(cfg RuntimeContextConfig) error {
	return ValidateRuntimeContextJITConfig(cfg.JIT)
}

func ValidateRuntimeContextJITConfig(cfg RuntimeContextJITConfig) error {
	if cfg.ReferenceFirst.MaxRefs <= 0 {
		return fmt.Errorf("runtime.context.jit.reference_first.max_refs must be > 0")
	}
	if cfg.ReferenceFirst.MaxResolveTokens <= 0 {
		return fmt.Errorf("runtime.context.jit.reference_first.max_resolve_tokens must be > 0")
	}
	if cfg.IsolateHandoff.DefaultTTLMS <= 0 {
		return fmt.Errorf("runtime.context.jit.isolate_handoff.default_ttl_ms must be > 0")
	}
	if cfg.IsolateHandoff.MinConfidence < 0 || cfg.IsolateHandoff.MinConfidence > 1 {
		return fmt.Errorf("runtime.context.jit.isolate_handoff.min_confidence must be in [0,1]")
	}
	if cfg.EditGate.ClearAtLeastTokens <= 0 {
		return fmt.Errorf("runtime.context.jit.edit_gate.clear_at_least_tokens must be > 0")
	}
	if cfg.EditGate.MinGainRatio <= 0 {
		return fmt.Errorf("runtime.context.jit.edit_gate.min_gain_ratio must be > 0")
	}
	if cfg.SwapBack.MinRelevanceScore < 0 || cfg.SwapBack.MinRelevanceScore > 1 {
		return fmt.Errorf("runtime.context.jit.swap_back.min_relevance_score must be in [0,1]")
	}
	if cfg.LifecycleTiering.HotTTLMS <= 0 {
		return fmt.Errorf("runtime.context.jit.lifecycle_tiering.hot_ttl_ms must be > 0")
	}
	if cfg.LifecycleTiering.WarmTTLMS <= 0 {
		return fmt.Errorf("runtime.context.jit.lifecycle_tiering.warm_ttl_ms must be > 0")
	}
	if cfg.LifecycleTiering.ColdTTLMS <= 0 {
		return fmt.Errorf("runtime.context.jit.lifecycle_tiering.cold_ttl_ms must be > 0")
	}
	if cfg.LifecycleTiering.HotTTLMS > cfg.LifecycleTiering.WarmTTLMS {
		return fmt.Errorf("runtime.context.jit.lifecycle_tiering.hot_ttl_ms must be <= runtime.context.jit.lifecycle_tiering.warm_ttl_ms")
	}
	if cfg.LifecycleTiering.WarmTTLMS > cfg.LifecycleTiering.ColdTTLMS {
		return fmt.Errorf("runtime.context.jit.lifecycle_tiering.warm_ttl_ms must be <= runtime.context.jit.lifecycle_tiering.cold_ttl_ms")
	}
	return nil
}

func strictFloatConfigValue(v *viper.Viper, key string) (float64, error) {
	raw := v.Get(key)
	if raw == nil {
		return v.GetFloat64(key), nil
	}
	return strictFloatAnyConfigValue(raw, key)
}

func strictFloatAnyConfigValue(raw any, key string) (float64, error) {
	switch value := raw.(type) {
	case float64:
		return value, nil
	case float32:
		return float64(value), nil
	case int:
		return float64(value), nil
	case int8:
		return float64(value), nil
	case int16:
		return float64(value), nil
	case int32:
		return float64(value), nil
	case int64:
		return float64(value), nil
	case uint:
		return float64(value), nil
	case uint8:
		return float64(value), nil
	case uint16:
		return float64(value), nil
	case uint32:
		return float64(value), nil
	case uint64:
		return float64(value), nil
	case string:
		trimmed := strings.TrimSpace(value)
		parsed, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return 0, fmt.Errorf("%s must be a number, got %q", key, value)
		}
		return parsed, nil
	default:
		trimmed := strings.TrimSpace(fmt.Sprint(raw))
		parsed, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return 0, fmt.Errorf("%s must be a number, got %v", key, raw)
		}
		return parsed, nil
	}
}
