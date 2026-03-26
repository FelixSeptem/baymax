package config

import (
	"fmt"
	"sort"
	"strings"
)

const (
	RuntimePrimarySourceReadiness = "runtime.readiness"
	RuntimePrimarySourceAdapter   = "adapter.health"
	RuntimePrimarySourceTimeout   = "timeout.resolution"
	RuntimePrimarySourceAdmission = "runtime.readiness.admission"
)

const (
	RuntimePrimaryCodeTimeoutRejected  = "runtime.timeout.parent_budget_rejected"
	RuntimePrimaryCodeTimeoutExhausted = "runtime.timeout.exhausted"
	RuntimePrimaryCodeTimeoutClamped   = "runtime.timeout.parent_budget_clamped"
)

type PrimaryReasonArbitrationInput struct {
	TimeoutParentBudgetRejectTotal int
	TimeoutParentBudgetClampTotal  int
	TimeoutExhaustedTotal          int
	TimeoutResolutionSource        string
	ReadinessFindings              []ReadinessFinding
}

type PrimaryReasonArbitrationResult struct {
	Domain        string
	Code          string
	Source        string
	ConflictTotal int
}

type primaryReasonCandidate struct {
	Domain     string
	Code       string
	Source     string
	Precedence int
}

func ArbitratePrimaryReason(in PrimaryReasonArbitrationInput) PrimaryReasonArbitrationResult {
	candidates := buildPrimaryReasonCandidates(in)
	if len(candidates) == 0 {
		return PrimaryReasonArbitrationResult{}
	}

	highest := candidates[0].Precedence
	for i := 1; i < len(candidates); i++ {
		if candidates[i].Precedence < highest {
			highest = candidates[i].Precedence
		}
	}

	top := make([]primaryReasonCandidate, 0, len(candidates))
	for i := range candidates {
		if candidates[i].Precedence == highest {
			top = append(top, candidates[i])
		}
	}
	sort.Slice(top, func(i, j int) bool {
		if top[i].Code != top[j].Code {
			return top[i].Code < top[j].Code
		}
		if top[i].Domain != top[j].Domain {
			return top[i].Domain < top[j].Domain
		}
		return top[i].Source < top[j].Source
	})

	conflict := 0
	if len(top) > 1 {
		conflict = len(top) - 1
	}
	return PrimaryReasonArbitrationResult{
		Domain:        top[0].Domain,
		Code:          top[0].Code,
		Source:        top[0].Source,
		ConflictTotal: conflict,
	}
}

func buildPrimaryReasonCandidates(in PrimaryReasonArbitrationInput) []primaryReasonCandidate {
	byKey := map[string]primaryReasonCandidate{}

	add := func(candidate primaryReasonCandidate) {
		candidate.Domain = strings.ToLower(strings.TrimSpace(candidate.Domain))
		candidate.Code = strings.TrimSpace(candidate.Code)
		candidate.Source = strings.ToLower(strings.TrimSpace(candidate.Source))
		if candidate.Code == "" || candidate.Precedence <= 0 {
			return
		}
		if candidate.Domain == "" {
			candidate.Domain = ReadinessDomainRuntime
		}
		if candidate.Source == "" {
			candidate.Source = RuntimePrimarySourceReadiness
		}
		key := fmt.Sprintf("%d|%s", candidate.Precedence, candidate.Code)
		existing, ok := byKey[key]
		if !ok {
			byKey[key] = candidate
			return
		}
		if candidate.Domain < existing.Domain || (candidate.Domain == existing.Domain && candidate.Source < existing.Source) {
			byKey[key] = candidate
		}
	}

	timeoutSource := normalizeTimeoutPrimarySource(in.TimeoutResolutionSource)
	if in.TimeoutParentBudgetRejectTotal > 0 {
		add(primaryReasonCandidate{
			Domain:     "timeout",
			Code:       RuntimePrimaryCodeTimeoutRejected,
			Source:     timeoutSource,
			Precedence: 1,
		})
	}
	if in.TimeoutExhaustedTotal > 0 {
		add(primaryReasonCandidate{
			Domain:     "timeout",
			Code:       RuntimePrimaryCodeTimeoutExhausted,
			Source:     timeoutSource,
			Precedence: 1,
		})
	}
	if in.TimeoutParentBudgetClampTotal > 0 {
		add(primaryReasonCandidate{
			Domain:     "timeout",
			Code:       RuntimePrimaryCodeTimeoutClamped,
			Source:     timeoutSource,
			Precedence: 5,
		})
	}

	for i := range in.ReadinessFindings {
		finding := in.ReadinessFindings[i]
		code := strings.TrimSpace(finding.Code)
		if code == "" {
			continue
		}
		domain := strings.ToLower(strings.TrimSpace(finding.Domain))
		source := RuntimePrimarySourceReadiness
		if strings.HasPrefix(code, "adapter.health.") || domain == ReadinessDomainAdapter {
			source = RuntimePrimarySourceAdapter
		}
		add(primaryReasonCandidate{
			Domain:     domain,
			Code:       code,
			Source:     source,
			Precedence: readinessPrimaryPrecedence(finding),
		})
	}

	if len(byKey) == 0 {
		return nil
	}
	out := make([]primaryReasonCandidate, 0, len(byKey))
	for _, candidate := range byKey {
		out = append(out, candidate)
	}
	return out
}

func readinessPrimaryPrecedence(finding ReadinessFinding) int {
	code := strings.TrimSpace(finding.Code)
	switch code {
	case ReadinessCodeAdapterRequiredUnavailable, ReadinessCodeAdapterRequiredCircuitOpen:
		return 3
	case ReadinessCodeAdapterOptionalUnavailable, ReadinessCodeAdapterOptionalCircuitOpen, ReadinessCodeAdapterDegraded, ReadinessCodeAdapterHalfOpenDegraded:
		return 4
	}

	switch normalizeReadinessSeverity(finding.Severity) {
	case ReadinessSeverityError:
		return 2
	case ReadinessSeverityWarning:
		return 4
	default:
		return 5
	}
}

func normalizeTimeoutPrimarySource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case TimeoutResolutionSourceProfile:
		return RuntimePrimarySourceTimeout + ".profile"
	case TimeoutResolutionSourceDomain:
		return RuntimePrimarySourceTimeout + ".domain"
	case TimeoutResolutionSourceRequest:
		return RuntimePrimarySourceTimeout + ".request"
	default:
		return RuntimePrimarySourceTimeout
	}
}
