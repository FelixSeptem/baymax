package config

import (
	"fmt"
	"strings"
)

const (
	RuntimeReactOnBudgetExhaustedFailFast = "fail_fast"
)

const (
	RuntimeReactPlanNotebookRecoverConflictReject       = "reject"
	RuntimeReactPlanNotebookRecoverConflictPreferLatest = "prefer_latest"
)

const (
	RuntimeReactPlanChangeHookFailModeFailFast = "fail_fast"
	RuntimeReactPlanChangeHookFailModeDegrade  = "degrade"
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
	Enabled                   bool                             `json:"enabled"`
	MaxIterations             int                              `json:"max_iterations"`
	ToolCallLimit             int                              `json:"tool_call_limit"`
	StreamToolDispatchEnabled bool                             `json:"stream_tool_dispatch_enabled"`
	OnBudgetExhausted         string                           `json:"on_budget_exhausted"`
	PlanNotebook              RuntimeReactPlanNotebookConfig   `json:"plan_notebook"`
	PlanChangeHook            RuntimeReactPlanChangeHookConfig `json:"plan_change_hook"`
}

type RuntimeReactPlanNotebookConfig struct {
	Enabled           bool   `json:"enabled"`
	MaxHistory        int    `json:"max_history"`
	OnRecoverConflict string `json:"on_recover_conflict"`
}

type RuntimeReactPlanChangeHookConfig struct {
	Enabled   bool   `json:"enabled"`
	FailMode  string `json:"fail_mode"`
	TimeoutMs int    `json:"timeout_ms"`
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
	out.PlanNotebook = normalizeRuntimeReactPlanNotebookConfig(out.PlanNotebook)
	out.PlanChangeHook = normalizeRuntimeReactPlanChangeHookConfig(out.PlanChangeHook)
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
	if err := ValidateRuntimeReactPlanNotebookConfig(cfg.PlanNotebook); err != nil {
		return err
	}
	if err := ValidateRuntimeReactPlanChangeHookConfig(cfg.PlanChangeHook); err != nil {
		return err
	}
	if cfg.PlanChangeHook.Enabled && !cfg.PlanNotebook.Enabled {
		return fmt.Errorf(
			"runtime.react.plan_change_hook.enabled requires runtime.react.plan_notebook.enabled=true",
		)
	}
	return nil
}

func normalizeRuntimeReactPlanNotebookConfig(in RuntimeReactPlanNotebookConfig) RuntimeReactPlanNotebookConfig {
	base := DefaultConfig().Runtime.React.PlanNotebook
	out := in
	if out.MaxHistory <= 0 {
		out.MaxHistory = base.MaxHistory
	}
	out.OnRecoverConflict = strings.ToLower(strings.TrimSpace(out.OnRecoverConflict))
	if out.OnRecoverConflict == "" {
		out.OnRecoverConflict = strings.ToLower(strings.TrimSpace(base.OnRecoverConflict))
	}
	return out
}

func normalizeRuntimeReactPlanChangeHookConfig(in RuntimeReactPlanChangeHookConfig) RuntimeReactPlanChangeHookConfig {
	base := DefaultConfig().Runtime.React.PlanChangeHook
	out := in
	out.FailMode = strings.ToLower(strings.TrimSpace(out.FailMode))
	if out.FailMode == "" {
		out.FailMode = strings.ToLower(strings.TrimSpace(base.FailMode))
	}
	if out.TimeoutMs <= 0 {
		out.TimeoutMs = base.TimeoutMs
	}
	return out
}

func ValidateRuntimeReactPlanNotebookConfig(cfg RuntimeReactPlanNotebookConfig) error {
	normalized := normalizeRuntimeReactPlanNotebookConfig(cfg)
	if cfg.MaxHistory <= 0 {
		return fmt.Errorf("runtime.react.plan_notebook.max_history must be > 0")
	}
	switch normalized.OnRecoverConflict {
	case RuntimeReactPlanNotebookRecoverConflictReject, RuntimeReactPlanNotebookRecoverConflictPreferLatest:
	default:
		return fmt.Errorf(
			"runtime.react.plan_notebook.on_recover_conflict must be one of [%s,%s], got %q",
			RuntimeReactPlanNotebookRecoverConflictReject,
			RuntimeReactPlanNotebookRecoverConflictPreferLatest,
			cfg.OnRecoverConflict,
		)
	}
	return nil
}

func ValidateRuntimeReactPlanChangeHookConfig(cfg RuntimeReactPlanChangeHookConfig) error {
	normalized := normalizeRuntimeReactPlanChangeHookConfig(cfg)
	if cfg.TimeoutMs <= 0 {
		return fmt.Errorf("runtime.react.plan_change_hook.timeout_ms must be > 0")
	}
	switch normalized.FailMode {
	case RuntimeReactPlanChangeHookFailModeFailFast, RuntimeReactPlanChangeHookFailModeDegrade:
	default:
		return fmt.Errorf(
			"runtime.react.plan_change_hook.fail_mode must be one of [%s,%s], got %q",
			RuntimeReactPlanChangeHookFailModeFailFast,
			RuntimeReactPlanChangeHookFailModeDegrade,
			cfg.FailMode,
		)
	}
	return nil
}
