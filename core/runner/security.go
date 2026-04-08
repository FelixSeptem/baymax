package runner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

const (
	securityPolicyKindPermission = "permission"
	securityPolicyKindRateLimit  = "rate_limit"
	securityPolicyKindIOFilter   = "io_filter"
	securityPolicyKindSandbox    = "sandbox"

	securityReasonPermissionDenied  = "security.permission_denied"
	securityReasonRateLimitDenied   = "security.rate_limit_exceeded"
	securityReasonIOFilterMatch     = "security.io_filter_match"
	securityReasonIOFilterDenied    = "security.io_filter_denied"
	securityReasonIOFilterError     = "security.io_filter_error"
	securityReasonIOFilterMissing   = "security.io_filter_missing"
	securityReasonSandboxPolicyDeny = "sandbox.policy_deny"

	securityAlertDispatchDisabled     = "disabled"
	securityAlertDispatchNotTriggered = "not_triggered"
	securityAlertDispatchSkipped      = "skipped"
	securityAlertDispatchSucceeded    = "succeeded"
	securityAlertDispatchFailed       = "failed"
	securityAlertDispatchQueued       = "queued"

	securityAlertFailureCallbackMissing = "alert.callback_missing"
	securityAlertFailureCallbackError   = "alert.callback_error"
	securityAlertFailureCallbackTimeout = "alert.callback_timeout"
	securityAlertFailureRetryExhausted  = "alert.retry_exhausted"
	securityAlertFailureCircuitOpen     = "alert.circuit_open"
)

type securityDecision struct {
	PolicyKind          string
	NamespaceTool       string
	FilterStage         string
	Decision            string
	ReasonCode          string
	Severity            string
	AlertDispatchStatus string
	AlertFailureReason  string
	AlertDeliveryMode   string
	AlertRetryCount     int
	AlertQueueDropped   bool
	AlertQueueDropCount int
	AlertCircuitState   string
	AlertCircuitReason  string
}

type toolRateWindow struct {
	windowStartedAt time.Time
	count           int
}

type processToolRateLimiter struct {
	mu      sync.Mutex
	windows map[string]toolRateWindow
}

func (l *processToolRateLimiter) allow(now time.Time, key string, window time.Duration, limit int) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.windows == nil {
		l.windows = map[string]toolRateWindow{}
	}
	state := l.windows[key]
	if state.windowStartedAt.IsZero() || now.Sub(state.windowStartedAt) >= window {
		state.windowStartedAt = now
		state.count = 0
	}
	if state.count+1 > limit {
		l.windows[key] = state
		return false, state.count
	}
	state.count++
	l.windows[key] = state
	return true, state.count
}

var processRateLimiterRegistry = struct {
	mu       sync.Mutex
	limiters map[string]*processToolRateLimiter
}{
	limiters: map[string]*processToolRateLimiter{},
}

func limiterForDomain(domain string) *processToolRateLimiter {
	processRateLimiterRegistry.mu.Lock()
	defer processRateLimiterRegistry.mu.Unlock()
	key := strings.TrimSpace(domain)
	if key == "" {
		key = "default"
	}
	if limiter, ok := processRateLimiterRegistry.limiters[key]; ok && limiter != nil {
		return limiter
	}
	limiter := &processToolRateLimiter{windows: map[string]toolRateWindow{}}
	processRateLimiterRegistry.limiters[key] = limiter
	return limiter
}

func (e *Engine) securityLimiterDomain() string {
	if e == nil {
		return "default"
	}
	if e.runtimeMgr != nil {
		return fmt.Sprintf("runtime-manager:%p", e.runtimeMgr)
	}
	return fmt.Sprintf("runner-engine:%p", e)
}

func (e *Engine) securityToolGovernanceConfig() runtimeconfig.SecurityToolGovernanceConfig {
	if e.runtimeMgr == nil {
		return runtimeconfig.DefaultConfig().Security.ToolGovernance
	}
	return e.runtimeMgr.EffectiveConfigRef().Security.ToolGovernance
}

func (e *Engine) runtimePolicyConfig() runtimeconfig.RuntimePolicyConfig {
	if e == nil || e.runtimeMgr == nil {
		return runtimeconfig.DefaultConfig().Runtime.Policy
	}
	return e.runtimeMgr.EffectiveConfigRef().Runtime.Policy
}

func (e *Engine) evaluateRuntimePolicyTrace(candidates []runtimeconfig.RuntimePolicyCandidate) (runtimeconfig.RuntimePolicyDecisionResult, bool) {
	if len(candidates) == 0 {
		return runtimeconfig.RuntimePolicyDecisionResult{}, false
	}
	trace, err := runtimeconfig.EvaluateRuntimePolicyDecision(e.runtimePolicyConfig(), candidates)
	if err != nil {
		return runtimeconfig.RuntimePolicyDecisionResult{}, false
	}
	return trace, true
}

func runtimePolicyStageForSecurityDecision(decision securityDecision) string {
	policyKind := strings.ToLower(strings.TrimSpace(decision.PolicyKind))
	reasonCode := strings.ToLower(strings.TrimSpace(decision.ReasonCode))
	if policyKind == securityPolicyKindSandbox {
		if strings.HasPrefix(reasonCode, "sandbox.egress") {
			return runtimeconfig.RuntimePolicyStageSandboxEgress
		}
		return runtimeconfig.RuntimePolicyStageSandboxAction
	}
	return runtimeconfig.RuntimePolicyStageSecurityS2
}

func (e *Engine) securitySandboxConfig() runtimeconfig.SecuritySandboxConfig {
	if e.runtimeMgr == nil {
		return runtimeconfig.DefaultConfig().Security.Sandbox
	}
	return e.runtimeMgr.EffectiveConfigRef().Security.Sandbox
}

func (e *Engine) securityModelIOFilteringConfig() runtimeconfig.SecurityModelIOFilteringConfig {
	if e.runtimeMgr == nil {
		return runtimeconfig.DefaultConfig().Security.ModelIOFiltering
	}
	return e.runtimeMgr.EffectiveConfigRef().Security.ModelIOFiltering
}

func (e *Engine) securityEventConfig() runtimeconfig.SecurityEventConfig {
	if e.runtimeMgr == nil {
		return runtimeconfig.DefaultConfig().Security.SecurityEvent
	}
	return e.runtimeMgr.EffectiveConfigRef().Security.SecurityEvent
}

func (e *Engine) enforceToolSecurityForCalls(
	ctx context.Context,
	h types.EventHandler,
	runID string,
	iteration int,
	seq *int64,
	calls []types.ToolCall,
) (*securityDecision, *types.ClassifiedError, error) {
	cfg := e.securityToolGovernanceConfig()
	sandboxCfg := e.securitySandboxConfig()
	if len(calls) == 0 {
		return nil, nil, nil
	}
	for _, call := range calls {
		namespaceTool, ok := namespaceToolKey(call.Name)
		if !ok {
			namespaceTool = "local+unknown"
		}
		sandboxMode := strings.ToLower(strings.TrimSpace(sandboxCfg.Mode))
		if sandboxCfg.Enabled && sandboxMode == runtimeconfig.SecuritySandboxModeEnforce {
			action := runtimeconfig.ResolveSandboxAction(sandboxCfg, namespaceTool)
			if action == runtimeconfig.SecuritySandboxActionDeny {
				decision := e.finalizeSecurityDecision(ctx, runID, iteration, securityDecision{
					PolicyKind:    securityPolicyKindSandbox,
					NamespaceTool: namespaceTool,
					Decision:      string(types.SecurityFilterDecisionDeny),
					ReasonCode:    securityReasonSandboxPolicyDeny,
				})
				e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, decision.ReasonCode)
				msg := fmt.Sprintf("tool call denied by sandbox policy: %s", namespaceTool)
				return &decision, e.securityDeniedError(msg, decision, map[string]any{
					"call_id":          strings.TrimSpace(call.CallID),
					"tool":             strings.TrimSpace(call.Name),
					"sandbox_mode":     sandboxMode,
					"sandbox_action":   action,
					"sandbox_profile":  runtimeconfig.ResolveSandboxProfile(sandboxCfg, namespaceTool),
					"sandbox_fallback": runtimeconfig.ResolveSandboxFallbackAction(sandboxCfg, namespaceTool),
				}), errors.New(msg)
			}
		}
		if !cfg.Enabled {
			continue
		}
		policy := resolvePermissionPolicy(cfg.Permission, namespaceTool)
		if policy == runtimeconfig.SecurityToolPolicyDeny {
			decision := e.finalizeSecurityDecision(ctx, runID, iteration, securityDecision{
				PolicyKind:    securityPolicyKindPermission,
				NamespaceTool: namespaceTool,
				Decision:      string(types.SecurityFilterDecisionDeny),
				ReasonCode:    securityReasonPermissionDenied,
			})
			e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, decision.ReasonCode)
			msg := fmt.Sprintf("tool call denied by permission policy: %s", namespaceTool)
			return &decision, e.securityDeniedError(msg, decision, map[string]any{
				"call_id": strings.TrimSpace(call.CallID),
				"tool":    strings.TrimSpace(call.Name),
			}), errors.New(msg)
		}
		if !cfg.RateLimit.Enabled {
			continue
		}
		limit := cfg.RateLimit.Limit
		if v, ok := cfg.RateLimit.ByToolLimit[namespaceTool]; ok && v > 0 {
			limit = v
		}
		allowed, count := limiterForDomain(e.securityLimiterDomain()).allow(
			e.now(),
			namespaceTool,
			cfg.RateLimit.Window,
			limit,
		)
		if allowed {
			continue
		}
		decision := e.finalizeSecurityDecision(ctx, runID, iteration, securityDecision{
			PolicyKind:    securityPolicyKindRateLimit,
			NamespaceTool: namespaceTool,
			Decision:      string(types.SecurityFilterDecisionDeny),
			ReasonCode:    securityReasonRateLimitDenied,
		})
		e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, decision.ReasonCode)
		msg := fmt.Sprintf("tool call denied by rate limit policy: %s", namespaceTool)
		return &decision, e.securityDeniedError(msg, decision, map[string]any{
			"call_id":         strings.TrimSpace(call.CallID),
			"tool":            strings.TrimSpace(call.Name),
			"rate_limit":      limit,
			"rate_count":      count,
			"rate_window_ms":  cfg.RateLimit.Window.Milliseconds(),
			"rate_scope":      cfg.RateLimit.Scope,
			"rate_exceed_act": cfg.RateLimit.ExceedAction,
		}), errors.New(msg)
	}
	return nil, nil, nil
}

func (e *Engine) applyInputFilters(ctx context.Context, runID string, iteration int, req types.ModelRequest) (types.ModelRequest, *securityDecision, *types.ClassifiedError, error) {
	cfg := e.securityModelIOFilteringConfig()
	if !cfg.Enabled || !cfg.Input.Enabled {
		return req, nil, nil, nil
	}
	if len(e.modelInputFilters) == 0 {
		if cfg.RequireRegisteredFilter {
			decision := e.finalizeSecurityDecision(ctx, runID, iteration, securityDecision{
				PolicyKind:  securityPolicyKindIOFilter,
				FilterStage: runtimeconfig.SecurityModelIOFilterStageInput,
				Decision:    string(types.SecurityFilterDecisionDeny),
				ReasonCode:  securityReasonIOFilterMissing,
			})
			msg := "model input denied because no input security filter is registered"
			return req, &decision, e.securityDeniedError(msg, decision, map[string]any{
				"require_registered_filter": true,
			}), errors.New(msg)
		}
		return req, nil, nil, nil
	}
	current := req
	var latest *securityDecision
	for _, filter := range e.modelInputFilters {
		next, result, err := filter.FilterModelInput(ctx, current)
		if err != nil {
			decision := e.finalizeSecurityDecision(ctx, runID, iteration, securityDecision{
				PolicyKind:  securityPolicyKindIOFilter,
				FilterStage: runtimeconfig.SecurityModelIOFilterStageInput,
				Decision:    string(types.SecurityFilterDecisionDeny),
				ReasonCode:  securityReasonIOFilterError,
			})
			msg := fmt.Sprintf("model input denied because filter execution failed: %v", err)
			return req, &decision, e.securityDeniedError(msg, decision, nil), err
		}
		current = next
		normalized := normalizeFilterDecision(result)
		switch normalized.Decision {
		case types.SecurityFilterDecisionDeny:
			decision := e.finalizeSecurityDecision(ctx, runID, iteration, securityDecision{
				PolicyKind:  securityPolicyKindIOFilter,
				FilterStage: runtimeconfig.SecurityModelIOFilterStageInput,
				Decision:    string(types.SecurityFilterDecisionDeny),
				ReasonCode:  normalizeReasonCode(normalized.ReasonCode, securityReasonIOFilterDenied),
			})
			msg := fmt.Sprintf("model input denied by security filter: %s", decision.ReasonCode)
			return req, &decision, e.securityDeniedError(msg, decision, nil), errors.New(msg)
		case types.SecurityFilterDecisionMatch:
			matchDecision := e.finalizeSecurityDecision(ctx, runID, iteration, securityDecision{
				PolicyKind:  securityPolicyKindIOFilter,
				FilterStage: runtimeconfig.SecurityModelIOFilterStageInput,
				Decision:    string(types.SecurityFilterDecisionMatch),
				ReasonCode:  normalizeReasonCode(normalized.ReasonCode, securityReasonIOFilterMatch),
			})
			latest = &matchDecision
		}
	}
	return current, latest, nil, nil
}

func (e *Engine) applyOutputFilters(ctx context.Context, runID string, iteration int, output string) (string, *securityDecision, *types.ClassifiedError, error) {
	cfg := e.securityModelIOFilteringConfig()
	if !cfg.Enabled || !cfg.Output.Enabled {
		return output, nil, nil, nil
	}
	if len(e.modelOutputFilters) == 0 {
		if cfg.RequireRegisteredFilter {
			decision := e.finalizeSecurityDecision(ctx, runID, iteration, securityDecision{
				PolicyKind:  securityPolicyKindIOFilter,
				FilterStage: runtimeconfig.SecurityModelIOFilterStageOutput,
				Decision:    string(types.SecurityFilterDecisionDeny),
				ReasonCode:  securityReasonIOFilterMissing,
			})
			msg := "model output denied because no output security filter is registered"
			return output, &decision, e.securityDeniedError(msg, decision, map[string]any{
				"require_registered_filter": true,
			}), errors.New(msg)
		}
		return output, nil, nil, nil
	}
	current := output
	var latest *securityDecision
	for _, filter := range e.modelOutputFilters {
		next, result, err := filter.FilterModelOutput(ctx, current)
		if err != nil {
			decision := e.finalizeSecurityDecision(ctx, runID, iteration, securityDecision{
				PolicyKind:  securityPolicyKindIOFilter,
				FilterStage: runtimeconfig.SecurityModelIOFilterStageOutput,
				Decision:    string(types.SecurityFilterDecisionDeny),
				ReasonCode:  securityReasonIOFilterError,
			})
			msg := fmt.Sprintf("model output denied because filter execution failed: %v", err)
			return output, &decision, e.securityDeniedError(msg, decision, nil), err
		}
		current = next
		normalized := normalizeFilterDecision(result)
		switch normalized.Decision {
		case types.SecurityFilterDecisionDeny:
			decision := e.finalizeSecurityDecision(ctx, runID, iteration, securityDecision{
				PolicyKind:  securityPolicyKindIOFilter,
				FilterStage: runtimeconfig.SecurityModelIOFilterStageOutput,
				Decision:    string(types.SecurityFilterDecisionDeny),
				ReasonCode:  normalizeReasonCode(normalized.ReasonCode, securityReasonIOFilterDenied),
			})
			msg := fmt.Sprintf("model output denied by security filter: %s", decision.ReasonCode)
			return output, &decision, e.securityDeniedError(msg, decision, nil), errors.New(msg)
		case types.SecurityFilterDecisionMatch:
			matchDecision := e.finalizeSecurityDecision(ctx, runID, iteration, securityDecision{
				PolicyKind:  securityPolicyKindIOFilter,
				FilterStage: runtimeconfig.SecurityModelIOFilterStageOutput,
				Decision:    string(types.SecurityFilterDecisionMatch),
				ReasonCode:  normalizeReasonCode(normalized.ReasonCode, securityReasonIOFilterMatch),
			})
			latest = &matchDecision
		}
	}
	return current, latest, nil, nil
}

func (e *Engine) finalizeSecurityDecision(ctx context.Context, runID string, iteration int, decision securityDecision) securityDecision {
	decision.PolicyKind = strings.ToLower(strings.TrimSpace(decision.PolicyKind))
	decision.NamespaceTool = strings.ToLower(strings.TrimSpace(decision.NamespaceTool))
	decision.FilterStage = strings.ToLower(strings.TrimSpace(decision.FilterStage))
	decision.Decision = strings.ToLower(strings.TrimSpace(decision.Decision))
	decision.ReasonCode = normalizeReasonCode(decision.ReasonCode, "")
	decision.Severity = e.resolveSecuritySeverity(decision)
	outcome := e.dispatchSecurityAlert(ctx, runID, iteration, decision)
	decision.AlertDispatchStatus = outcome.Status
	decision.AlertFailureReason = outcome.FailureReason
	decision.AlertDeliveryMode = outcome.DeliveryMode
	decision.AlertRetryCount = outcome.RetryCount
	decision.AlertQueueDropped = outcome.QueueDropped
	decision.AlertQueueDropCount = outcome.QueueDropCount
	decision.AlertCircuitState = outcome.CircuitState
	decision.AlertCircuitReason = outcome.CircuitOpenReason
	return decision
}

func (e *Engine) resolveSecuritySeverity(decision securityDecision) string {
	cfg := e.securityEventConfig()
	if !cfg.Enabled {
		return ""
	}
	reason := strings.ToLower(strings.TrimSpace(decision.ReasonCode))
	if reason != "" {
		if mapped, ok := cfg.Severity.ByReasonCode[reason]; ok {
			return normalizeSecuritySeverity(mapped, cfg.Severity.Default)
		}
	}
	policy := strings.ToLower(strings.TrimSpace(decision.PolicyKind))
	if policy != "" {
		if mapped, ok := cfg.Severity.ByPolicyKind[policy]; ok {
			return normalizeSecuritySeverity(mapped, cfg.Severity.Default)
		}
	}
	return normalizeSecuritySeverity(cfg.Severity.Default, runtimeconfig.SecurityEventSeverityHigh)
}

func normalizeSecuritySeverity(raw string, fallback string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case runtimeconfig.SecurityEventSeverityLow, runtimeconfig.SecurityEventSeverityMedium, runtimeconfig.SecurityEventSeverityHigh:
		return value
	}
	fallback = strings.ToLower(strings.TrimSpace(fallback))
	switch fallback {
	case runtimeconfig.SecurityEventSeverityLow, runtimeconfig.SecurityEventSeverityMedium, runtimeconfig.SecurityEventSeverityHigh:
		return fallback
	default:
		return runtimeconfig.SecurityEventSeverityHigh
	}
}

func (e *Engine) dispatchSecurityAlert(ctx context.Context, runID string, iteration int, decision securityDecision) securityAlertDispatchResult {
	cfg := e.securityEventConfig()
	mode := normalizeSecurityAlertDeliveryMode(cfg.Delivery.Mode)
	if !cfg.Enabled {
		return securityAlertDispatchResult{
			Status:       securityAlertDispatchDisabled,
			DeliveryMode: mode,
			CircuitState: runtimeconfig.SecurityEventCircuitStateClosed,
		}
	}
	if decision.Decision != string(types.SecurityFilterDecisionDeny) {
		return securityAlertDispatchResult{
			Status:       securityAlertDispatchNotTriggered,
			DeliveryMode: mode,
			CircuitState: runtimeconfig.SecurityEventCircuitStateClosed,
		}
	}
	if cfg.Alert.TriggerPolicy != runtimeconfig.SecurityEventAlertPolicyDenyOnly {
		return securityAlertDispatchResult{
			Status:       securityAlertDispatchDisabled,
			DeliveryMode: mode,
			CircuitState: runtimeconfig.SecurityEventCircuitStateClosed,
		}
	}
	if cfg.Alert.Sink != runtimeconfig.SecurityEventAlertSinkCallback {
		return securityAlertDispatchResult{
			Status:        securityAlertDispatchFailed,
			FailureReason: securityAlertFailureCallbackMissing,
			DeliveryMode:  mode,
			CircuitState:  runtimeconfig.SecurityEventCircuitStateClosed,
		}
	}
	if e.securityAlert == nil {
		if cfg.Alert.Callback.RequireRegistered {
			return securityAlertDispatchResult{
				Status:        securityAlertDispatchFailed,
				FailureReason: securityAlertFailureCallbackMissing,
				DeliveryMode:  mode,
				CircuitState:  runtimeconfig.SecurityEventCircuitStateClosed,
			}
		}
		return securityAlertDispatchResult{
			Status:       securityAlertDispatchSkipped,
			DeliveryMode: mode,
			CircuitState: runtimeconfig.SecurityEventCircuitStateClosed,
		}
	}
	return e.securityDeliveryExecutor().dispatch(ctx, securityAlertDeliveryRequest{
		Config:   cfg.Delivery,
		Callback: e.securityAlert,
		Event: types.SecurityEvent{
			EventID:       e.securityEventID(runID, iteration),
			RunID:         strings.TrimSpace(runID),
			Iteration:     iteration,
			PolicyKind:    decision.PolicyKind,
			NamespaceTool: decision.NamespaceTool,
			FilterStage:   decision.FilterStage,
			Decision:      decision.Decision,
			ReasonCode:    decision.ReasonCode,
			Severity:      decision.Severity,
			Timestamp:     e.now(),
		},
	})
}

func (e *Engine) securityEventID(runID string, iteration int) string {
	return fmt.Sprintf("%s-%d-%d", strings.TrimSpace(runID), iteration, e.now().UnixNano())
}

func normalizeFilterDecision(in types.SecurityFilterResult) types.SecurityFilterResult {
	out := in
	switch strings.ToLower(strings.TrimSpace(string(in.Decision))) {
	case string(types.SecurityFilterDecisionAllow):
		out.Decision = types.SecurityFilterDecisionAllow
	case string(types.SecurityFilterDecisionMatch):
		out.Decision = types.SecurityFilterDecisionMatch
	case string(types.SecurityFilterDecisionDeny):
		out.Decision = types.SecurityFilterDecisionDeny
	default:
		out.Decision = types.SecurityFilterDecisionAllow
	}
	return out
}

func normalizeReasonCode(raw string, fallback string) string {
	code := strings.ToLower(strings.TrimSpace(raw))
	if code == "" {
		return strings.ToLower(strings.TrimSpace(fallback))
	}
	return code
}

func resolvePermissionPolicy(cfg runtimeconfig.SecurityPermissionConfig, namespaceTool string) string {
	if policy, ok := cfg.ByTool[namespaceTool]; ok {
		return strings.ToLower(strings.TrimSpace(policy))
	}
	policy := strings.ToLower(strings.TrimSpace(cfg.Default))
	if policy == "" {
		return runtimeconfig.SecurityToolPolicyAllow
	}
	return policy
}

func namespaceToolKey(toolName string) (string, bool) {
	normalized := normalizeToolName(toolName)
	if normalized == "" {
		return "", false
	}
	if strings.Contains(normalized, "+") {
		parts := strings.Split(normalized, "+")
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return "", false
		}
		return normalized, true
	}
	namespace := "local"
	tool := normalized
	if idx := strings.Index(normalized, "."); idx >= 0 {
		namespace = strings.TrimSpace(normalized[:idx])
		tool = strings.TrimSpace(normalized[idx+1:])
	}
	if namespace == "" || tool == "" {
		return "", false
	}
	return namespace + "+" + tool, true
}

func (e *Engine) securityDeniedError(message string, decision securityDecision, extra map[string]any) *types.ClassifiedError {
	details := map[string]any{
		"policy_kind":    decision.PolicyKind,
		"namespace_tool": decision.NamespaceTool,
		"filter_stage":   decision.FilterStage,
		"decision":       decision.Decision,
		"reason_code":    decision.ReasonCode,
		"severity":       decision.Severity,
	}
	if decision.AlertDispatchStatus != "" {
		details["alert_dispatch_status"] = decision.AlertDispatchStatus
	}
	if decision.AlertFailureReason != "" {
		details["alert_dispatch_failure_reason"] = decision.AlertFailureReason
	}
	if decision.AlertDeliveryMode != "" {
		details["alert_delivery_mode"] = decision.AlertDeliveryMode
	}
	details["alert_retry_count"] = decision.AlertRetryCount
	if decision.AlertQueueDropped {
		details["alert_queue_dropped"] = true
	}
	if decision.AlertQueueDropCount > 0 {
		details["alert_queue_drop_count"] = decision.AlertQueueDropCount
	}
	if decision.AlertCircuitState != "" {
		details["alert_circuit_state"] = decision.AlertCircuitState
	}
	if decision.AlertCircuitReason != "" {
		details["alert_circuit_open_reason"] = decision.AlertCircuitReason
	}
	stage := runtimePolicyStageForSecurityDecision(decision)
	reasonCode := strings.ToLower(strings.TrimSpace(decision.ReasonCode))
	if trace, ok := e.evaluateRuntimePolicyTrace([]runtimeconfig.RuntimePolicyCandidate{
		{
			Stage:    stage,
			Code:     reasonCode,
			Source:   stage,
			Decision: runtimeconfig.RuntimePolicyDecisionDeny,
		},
	}); ok {
		if strings.TrimSpace(trace.Version) != "" {
			details["policy_precedence_version"] = strings.TrimSpace(trace.Version)
		}
		if strings.TrimSpace(trace.WinnerStage) != "" {
			details["winner_stage"] = strings.TrimSpace(trace.WinnerStage)
		}
		if strings.TrimSpace(trace.DenySource) != "" {
			details["deny_source"] = strings.TrimSpace(trace.DenySource)
		}
		if strings.TrimSpace(trace.TieBreakReason) != "" {
			details["tie_break_reason"] = strings.TrimSpace(trace.TieBreakReason)
		}
		if len(trace.PolicyDecisionPath) > 0 {
			details["policy_decision_path"] = append([]runtimeconfig.RuntimePolicyCandidate(nil), trace.PolicyDecisionPath...)
		}
	}
	for k, v := range extra {
		details[k] = v
	}
	return &types.ClassifiedError{
		Class:     types.ErrSecurity,
		Message:   message,
		Retryable: false,
		Details:   details,
	}
}

func (e *Engine) securityDeliveryExecutor() *securityAlertDeliveryExecutor {
	if e == nil {
		return newSecurityAlertDeliveryExecutor(time.Now)
	}
	e.securityDeliveryMu.Lock()
	defer e.securityDeliveryMu.Unlock()
	if e.securityDelivery == nil {
		e.securityDelivery = newSecurityAlertDeliveryExecutor(e.now)
	}
	return e.securityDelivery
}

func normalizeSecurityAlertDeliveryMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case runtimeconfig.SecurityEventDeliveryModeSync:
		return runtimeconfig.SecurityEventDeliveryModeSync
	case runtimeconfig.SecurityEventDeliveryModeAsync:
		return runtimeconfig.SecurityEventDeliveryModeAsync
	default:
		return runtimeconfig.SecurityEventDeliveryModeAsync
	}
}

func normalizeInputFilters(filters []types.ModelInputSecurityFilter) []types.ModelInputSecurityFilter {
	if len(filters) == 0 {
		return nil
	}
	out := make([]types.ModelInputSecurityFilter, 0, len(filters))
	for _, filter := range filters {
		if filter == nil {
			continue
		}
		out = append(out, filter)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeOutputFilters(filters []types.ModelOutputSecurityFilter) []types.ModelOutputSecurityFilter {
	if len(filters) == 0 {
		return nil
	}
	out := make([]types.ModelOutputSecurityFilter, 0, len(filters))
	for _, filter := range filters {
		if filter == nil {
			continue
		}
		out = append(out, filter)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
