package config

import (
	"fmt"
	"strings"
)

const (
	RuntimeReactOnBudgetExhaustedFailFast = "fail_fast"
)

const (
	RuntimeReactTerminationCompleted             = "react.completed"
	RuntimeReactTerminationMaxIterationsExceeded = "react.max_iterations_exceeded"
	RuntimeReactTerminationToolCallLimitExceeded = "react.tool_call_limit_exceeded"
	RuntimeReactTerminationToolDispatchFailed    = "react.tool_dispatch_failed"
	RuntimeReactTerminationProviderError         = "react.provider_error"
	RuntimeReactTerminationContextCanceled       = "react.context_canceled"
)

type RuntimeReactConfig struct {
	Enabled                   bool   `json:"enabled"`
	MaxIterations             int    `json:"max_iterations"`
	ToolCallLimit             int    `json:"tool_call_limit"`
	StreamToolDispatchEnabled bool   `json:"stream_tool_dispatch_enabled"`
	OnBudgetExhausted         string `json:"on_budget_exhausted"`
}

func normalizeRuntimeReactConfig(in RuntimeReactConfig) RuntimeReactConfig {
	base := DefaultConfig().Runtime.React
	out := in
	if out.MaxIterations <= 0 {
		out.MaxIterations = base.MaxIterations
	}
	if out.ToolCallLimit <= 0 {
		out.ToolCallLimit = base.ToolCallLimit
	}
	out.OnBudgetExhausted = strings.ToLower(strings.TrimSpace(out.OnBudgetExhausted))
	if out.OnBudgetExhausted == "" {
		out.OnBudgetExhausted = base.OnBudgetExhausted
	}
	return out
}

func ValidateRuntimeReactConfig(cfg RuntimeReactConfig) error {
	normalized := normalizeRuntimeReactConfig(cfg)
	if cfg.MaxIterations <= 0 {
		return fmt.Errorf("runtime.react.max_iterations must be > 0")
	}
	if cfg.ToolCallLimit <= 0 {
		return fmt.Errorf("runtime.react.tool_call_limit must be > 0")
	}
	switch normalized.OnBudgetExhausted {
	case RuntimeReactOnBudgetExhaustedFailFast:
	default:
		return fmt.Errorf(
			"runtime.react.on_budget_exhausted must be one of [%s], got %q",
			RuntimeReactOnBudgetExhaustedFailFast,
			cfg.OnBudgetExhausted,
		)
	}
	if cfg.StreamToolDispatchEnabled && !cfg.Enabled {
		return fmt.Errorf(
			"runtime.react.stream_tool_dispatch_enabled requires runtime.react.enabled=true",
		)
	}
	return nil
}
