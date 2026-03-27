package composer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

// ReadinessPreflight returns runtime readiness result for managed runtime path.
// The query is read-only and does not mutate scheduler/task state.
func (c *Composer) ReadinessPreflight() (runtimeconfig.ReadinessResult, error) {
	if c == nil {
		return runtimeconfig.ReadinessResult{}, errors.New("composer is nil")
	}
	if c.runtimeMgr == nil {
		return runtimeconfig.ReadinessResult{}, errors.New("runtime manager is not initialized")
	}
	return c.runtimeMgr.ReadinessPreflight(), nil
}

func (c *Composer) guardReadinessAdmission(
	ctx context.Context,
	req types.RunRequest,
	h types.EventHandler,
) (types.RunRequest, *types.RunResult, error) {
	if c == nil || c.runtimeMgr == nil {
		return req, nil, nil
	}
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		runID = fmt.Sprintf("run-%d", resolveReadinessSnapshotTime(c.now).UnixNano())
		req.RunID = runID
	}
	decision := c.runtimeMgr.EvaluateReadinessAdmission()
	c.recordReadinessAdmission(runID, decision)
	if decision.Outcome != runtimeconfig.ReadinessAdmissionOutcomeDeny {
		return req, nil, nil
	}

	msg := fmt.Sprintf("runtime readiness admission denied: %s", strings.TrimSpace(decision.ReasonCode))
	result := &types.RunResult{
		RunID:       runID,
		Iterations:  0,
		ToolCalls:   nil,
		LatencyMs:   0,
		FinalAnswer: "",
		Warnings:    nil,
		Error: &types.ClassifiedError{
			Class:     types.ErrContext,
			Message:   msg,
			Retryable: false,
			Details: map[string]any{
				"reason_code":                        strings.TrimSpace(decision.ReasonCode),
				"runtime_readiness":                  string(decision.ReadinessStatus),
				"readiness_primary_domain":           strings.TrimSpace(decision.ReadinessPrimaryDomain),
				"readiness_primary_code":             strings.TrimSpace(decision.ReadinessPrimaryCode),
				"readiness_primary_source":           strings.TrimSpace(decision.ReadinessPrimarySource),
				"readiness_secondary_reason_codes":   append([]string(nil), decision.ReadinessSecondaryReasonCodes...),
				"readiness_secondary_reason_count":   decision.ReadinessSecondaryReasonCount,
				"readiness_arbitration_rule_version": strings.TrimSpace(decision.ReadinessArbitrationRuleVersion),
				"readiness_remediation_hint_code":    strings.TrimSpace(decision.ReadinessRemediationHintCode),
				"readiness_remediation_hint_domain":  strings.TrimSpace(decision.ReadinessRemediationHintDomain),
				"admission_mode":                     strings.TrimSpace(decision.Mode),
			},
		},
	}
	c.emitAdmissionDeniedEvent(ctx, runID, h, result)
	return req, result, errors.New(msg)
}

func (c *Composer) recordReadinessAdmission(runID string, decision runtimeconfig.ReadinessAdmissionDecision) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return
	}
	c.runMu.Lock()
	defer c.runMu.Unlock()
	stat := c.ensureRunStat(runID)
	stat.ReadinessAdmissionMode = strings.TrimSpace(decision.Mode)
	stat.ReadinessAdmissionPrimaryCode = strings.TrimSpace(decision.ReadinessPrimaryCode)
	if decision.Bypass {
		stat.ReadinessAdmissionBypassTotal++
		return
	}
	stat.ReadinessAdmissionTotal++
	switch decision.ReadinessStatus {
	case runtimeconfig.ReadinessStatusBlocked:
		stat.ReadinessAdmissionBlockedTotal++
	case runtimeconfig.ReadinessStatusDegraded:
		if decision.Outcome == runtimeconfig.ReadinessAdmissionOutcomeAllow {
			stat.ReadinessAdmissionDegradedAllowTotal++
		}
	}
}

func (c *Composer) emitAdmissionDeniedEvent(ctx context.Context, runID string, h types.EventHandler, result *types.RunResult) {
	if c == nil {
		return
	}
	handler := c.bridgeHandler(h)
	if handler == nil || result == nil {
		return
	}
	payload := map[string]any{
		"status":      "failed",
		"iterations":  result.Iterations,
		"tool_calls":  0,
		"latency_ms":  result.LatencyMs,
		"error_class": string(types.ErrContext),
		"error":       result.Error.Message,
	}
	handler.OnEvent(ctx, types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    "run.finished",
		RunID:   runID,
		Time:    resolveReadinessSnapshotTime(c.now),
		Payload: payload,
	})
}

func (c *Composer) publishRuntimeReadinessSnapshot() {
	if c == nil || c.runtimeMgr == nil {
		return
	}
	c.schedulerMu.RLock()
	snapshot := runtimeconfig.RuntimeReadinessComponentSnapshot{
		Scheduler: runtimeconfig.RuntimeReadinessComponentState{
			Enabled:           true,
			ConfiguredBackend: strings.TrimSpace(c.schedulerConfiguredBackend),
			EffectiveBackend:  strings.TrimSpace(c.schedulerBackend),
			Fallback:          c.schedulerFallback,
			FallbackReason:    strings.TrimSpace(c.schedulerFallbackReason),
		},
		Mailbox: runtimeconfig.RuntimeReadinessComponentState{
			Enabled:           c.mailboxEnabled,
			ConfiguredBackend: strings.TrimSpace(c.mailboxConfiguredBackend),
			EffectiveBackend:  strings.TrimSpace(c.mailboxBackend),
			Fallback:          c.mailboxFallback,
			FallbackReason:    strings.TrimSpace(c.mailboxFallbackReason),
		},
		Recovery: runtimeconfig.RuntimeReadinessComponentState{
			Enabled:           c.recoveryEnabled,
			ConfiguredBackend: strings.TrimSpace(c.recoveryConfiguredBackend),
			EffectiveBackend:  strings.TrimSpace(c.recoveryBackend),
			Fallback:          c.recoveryFallback,
			FallbackReason:    strings.TrimSpace(c.recoveryFallbackReason),
		},
		UpdatedAt: resolveReadinessSnapshotTime(c.now),
	}
	c.schedulerMu.RUnlock()
	c.runtimeMgr.SetReadinessComponentSnapshot(snapshot)
}

func resolveReadinessSnapshotTime(now func() time.Time) time.Time {
	if now == nil {
		return time.Now().UTC()
	}
	return now().UTC()
}
