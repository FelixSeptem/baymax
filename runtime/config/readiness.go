package config

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type ReadinessStatus string

const (
	ReadinessStatusReady    ReadinessStatus = "ready"
	ReadinessStatusDegraded ReadinessStatus = "degraded"
	ReadinessStatusBlocked  ReadinessStatus = "blocked"
)

const (
	ReadinessDomainRuntime   = "runtime"
	ReadinessDomainConfig    = "config"
	ReadinessDomainScheduler = "scheduler"
	ReadinessDomainMailbox   = "mailbox"
	ReadinessDomainRecovery  = "recovery"
)

const (
	ReadinessSeverityInfo    = "info"
	ReadinessSeverityWarning = "warning"
	ReadinessSeverityError   = "error"
)

const (
	ReadinessCodeConfigInvalid             = "runtime.config.invalid"
	ReadinessCodeStrictEscalated           = "runtime.readiness.strict_escalated"
	ReadinessCodeSchedulerFallback         = "scheduler.backend.fallback"
	ReadinessCodeSchedulerActivationError  = "scheduler.backend.activation_failed"
	ReadinessCodeMailboxFallback           = "mailbox.backend.fallback"
	ReadinessCodeMailboxActivationError    = "mailbox.backend.activation_failed"
	ReadinessCodeRecoveryFallback          = "recovery.backend.fallback"
	ReadinessCodeRecoveryActivationError   = "recovery.backend.activation_failed"
	ReadinessCodeRuntimeManagerUnavailable = "runtime.manager.unavailable"
)

type ReadinessFinding struct {
	Code     string         `json:"code"`
	Domain   string         `json:"domain"`
	Severity string         `json:"severity"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata"`
}

type ReadinessResult struct {
	Status      ReadinessStatus    `json:"status"`
	Findings    []ReadinessFinding `json:"findings"`
	EvaluatedAt time.Time          `json:"evaluated_at"`
}

type ReadinessSummary struct {
	Status        string `json:"runtime_readiness_status"`
	FindingTotal  int    `json:"runtime_readiness_finding_total"`
	BlockingTotal int    `json:"runtime_readiness_blocking_total"`
	DegradedTotal int    `json:"runtime_readiness_degraded_total"`
	PrimaryCode   string `json:"runtime_readiness_primary_code"`
}

type RuntimeReadinessComponentState struct {
	Enabled           bool   `json:"enabled"`
	ConfiguredBackend string `json:"configured_backend,omitempty"`
	EffectiveBackend  string `json:"effective_backend,omitempty"`
	Fallback          bool   `json:"fallback,omitempty"`
	FallbackReason    string `json:"fallback_reason,omitempty"`
	ActivationError   string `json:"activation_error,omitempty"`
}

type RuntimeReadinessComponentSnapshot struct {
	Scheduler RuntimeReadinessComponentState `json:"scheduler"`
	Mailbox   RuntimeReadinessComponentState `json:"mailbox"`
	Recovery  RuntimeReadinessComponentState `json:"recovery"`
	UpdatedAt time.Time                      `json:"updated_at,omitempty"`
}

func (m *Manager) SetReadinessComponentSnapshot(snapshot RuntimeReadinessComponentSnapshot) {
	if m == nil {
		return
	}
	m.readinessMu.Lock()
	m.readinessComponents = cloneReadinessComponentSnapshot(snapshot)
	m.readinessMu.Unlock()
}

func (m *Manager) ReadinessComponentSnapshot() RuntimeReadinessComponentSnapshot {
	if m == nil {
		return RuntimeReadinessComponentSnapshot{}
	}
	m.readinessMu.RLock()
	defer m.readinessMu.RUnlock()
	return cloneReadinessComponentSnapshot(m.readinessComponents)
}

func (m *Manager) ReadinessPreflight() ReadinessResult {
	evaluatedAt := time.Now().UTC()
	if m == nil {
		return ReadinessResult{
			Status: ReadinessStatusBlocked,
			Findings: []ReadinessFinding{
				{
					Code:     ReadinessCodeRuntimeManagerUnavailable,
					Domain:   ReadinessDomainRuntime,
					Severity: ReadinessSeverityError,
					Message:  "runtime manager is unavailable",
					Metadata: map[string]any{},
				},
			},
			EvaluatedAt: evaluatedAt,
		}
	}

	cfg := m.EffectiveConfig()
	if !cfg.Runtime.Readiness.Enabled {
		return ReadinessResult{
			Status:      ReadinessStatusReady,
			Findings:    nil,
			EvaluatedAt: evaluatedAt,
		}
	}

	findings := make([]ReadinessFinding, 0, 6)
	if err := Validate(cfg); err != nil {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeConfigInvalid,
			Domain:   ReadinessDomainConfig,
			Severity: ReadinessSeverityError,
			Message:  "effective runtime config is invalid",
			Metadata: map[string]any{"error": strings.TrimSpace(err.Error())},
		})
	}

	componentSnapshot := m.ReadinessComponentSnapshot()
	findings = append(findings, componentReadinessFindings("scheduler", componentSnapshot.Scheduler)...)
	findings = append(findings, componentReadinessFindings("mailbox", componentSnapshot.Mailbox)...)
	findings = append(findings, componentReadinessFindings("recovery", componentSnapshot.Recovery)...)
	findings = canonicalizeReadinessFindings(findings)

	status := classifyReadinessStatus(findings)
	if cfg.Runtime.Readiness.Strict && status == ReadinessStatusDegraded {
		findings = append(findings, ReadinessFinding{
			Code:     ReadinessCodeStrictEscalated,
			Domain:   ReadinessDomainRuntime,
			Severity: ReadinessSeverityError,
			Message:  "strict readiness policy escalated degraded findings to blocked",
			Metadata: map[string]any{"strict": true},
		})
		findings = canonicalizeReadinessFindings(findings)
		status = ReadinessStatusBlocked
	}

	return ReadinessResult{
		Status:      status,
		Findings:    findings,
		EvaluatedAt: evaluatedAt,
	}
}

func (r ReadinessResult) Summary() ReadinessSummary {
	summary := ReadinessSummary{
		Status: string(r.Status),
	}
	for i := range r.Findings {
		finding := r.Findings[i]
		summary.FindingTotal++
		switch strings.ToLower(strings.TrimSpace(finding.Severity)) {
		case ReadinessSeverityError:
			summary.BlockingTotal++
			if summary.PrimaryCode == "" {
				summary.PrimaryCode = strings.TrimSpace(finding.Code)
			}
		case ReadinessSeverityWarning:
			summary.DegradedTotal++
			if summary.PrimaryCode == "" {
				summary.PrimaryCode = strings.TrimSpace(finding.Code)
			}
		}
	}
	if strings.TrimSpace(summary.Status) == "" {
		summary.Status = string(ReadinessStatusReady)
	}
	return summary
}

func componentReadinessFindings(component string, state RuntimeReadinessComponentState) []ReadinessFinding {
	component = strings.ToLower(strings.TrimSpace(component))
	if component == "" {
		return nil
	}

	metadata := map[string]any{}
	if configured := strings.ToLower(strings.TrimSpace(state.ConfiguredBackend)); configured != "" {
		metadata["configured_backend"] = configured
	}
	if effective := strings.ToLower(strings.TrimSpace(state.EffectiveBackend)); effective != "" {
		metadata["effective_backend"] = effective
	}

	if activationErr := strings.TrimSpace(state.ActivationError); activationErr != "" {
		metadata["activation_error"] = activationErr
		return []ReadinessFinding{
			{
				Code:     readinessActivationCode(component),
				Domain:   component,
				Severity: ReadinessSeverityError,
				Message:  fmt.Sprintf("%s backend activation failed", component),
				Metadata: metadata,
			},
		}
	}

	if state.Fallback {
		if reason := strings.TrimSpace(state.FallbackReason); reason != "" {
			metadata["fallback_reason"] = reason
		}
		return []ReadinessFinding{
			{
				Code:     readinessFallbackCode(component),
				Domain:   component,
				Severity: ReadinessSeverityWarning,
				Message:  fmt.Sprintf("%s backend fell back to memory", component),
				Metadata: metadata,
			},
		}
	}

	return nil
}

func readinessFallbackCode(component string) string {
	switch component {
	case "scheduler":
		return ReadinessCodeSchedulerFallback
	case "mailbox":
		return ReadinessCodeMailboxFallback
	case "recovery":
		return ReadinessCodeRecoveryFallback
	default:
		return component + ".backend.fallback"
	}
}

func readinessActivationCode(component string) string {
	switch component {
	case "scheduler":
		return ReadinessCodeSchedulerActivationError
	case "mailbox":
		return ReadinessCodeMailboxActivationError
	case "recovery":
		return ReadinessCodeRecoveryActivationError
	default:
		return component + ".backend.activation_failed"
	}
}

func classifyReadinessStatus(findings []ReadinessFinding) ReadinessStatus {
	status := ReadinessStatusReady
	for i := range findings {
		switch strings.ToLower(strings.TrimSpace(findings[i].Severity)) {
		case ReadinessSeverityError:
			return ReadinessStatusBlocked
		case ReadinessSeverityWarning:
			status = ReadinessStatusDegraded
		}
	}
	return status
}

func canonicalizeReadinessFindings(findings []ReadinessFinding) []ReadinessFinding {
	if len(findings) == 0 {
		return nil
	}
	out := make([]ReadinessFinding, 0, len(findings))
	for i := range findings {
		item := findings[i]
		item.Code = strings.TrimSpace(item.Code)
		item.Domain = strings.ToLower(strings.TrimSpace(item.Domain))
		item.Severity = normalizeReadinessSeverity(item.Severity)
		item.Message = strings.TrimSpace(item.Message)
		item.Metadata = cloneAnyMap(item.Metadata)
		if item.Metadata == nil {
			item.Metadata = map[string]any{}
		}
		if item.Code == "" {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		li := readinessSeverityRank(out[i].Severity)
		lj := readinessSeverityRank(out[j].Severity)
		if li != lj {
			return li > lj
		}
		if out[i].Domain != out[j].Domain {
			return out[i].Domain < out[j].Domain
		}
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].Message < out[j].Message
	})
	return out
}

func normalizeReadinessSeverity(in string) string {
	switch strings.ToLower(strings.TrimSpace(in)) {
	case ReadinessSeverityInfo:
		return ReadinessSeverityInfo
	case ReadinessSeverityWarning:
		return ReadinessSeverityWarning
	case ReadinessSeverityError:
		return ReadinessSeverityError
	default:
		return ReadinessSeverityInfo
	}
}

func readinessSeverityRank(severity string) int {
	switch normalizeReadinessSeverity(severity) {
	case ReadinessSeverityError:
		return 3
	case ReadinessSeverityWarning:
		return 2
	default:
		return 1
	}
}

func cloneReadinessComponentSnapshot(in RuntimeReadinessComponentSnapshot) RuntimeReadinessComponentSnapshot {
	out := in
	out.Scheduler = cloneReadinessComponentState(in.Scheduler)
	out.Mailbox = cloneReadinessComponentState(in.Mailbox)
	out.Recovery = cloneReadinessComponentState(in.Recovery)
	return out
}

func cloneReadinessComponentState(in RuntimeReadinessComponentState) RuntimeReadinessComponentState {
	return RuntimeReadinessComponentState{
		Enabled:           in.Enabled,
		ConfiguredBackend: strings.TrimSpace(in.ConfiguredBackend),
		EffectiveBackend:  strings.TrimSpace(in.EffectiveBackend),
		Fallback:          in.Fallback,
		FallbackReason:    strings.TrimSpace(in.FallbackReason),
		ActivationError:   strings.TrimSpace(in.ActivationError),
	}
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
