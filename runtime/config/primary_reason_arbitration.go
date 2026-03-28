package config

import (
	"fmt"
	"sort"
	"strings"
)

const (
	RuntimePrimarySourceReadiness   = "runtime.readiness"
	RuntimePrimarySourceAdapter     = "adapter.health"
	RuntimePrimarySourceTimeout     = "timeout.resolution"
	RuntimePrimarySourceAdmission   = "runtime.readiness.admission"
	RuntimePrimarySourceArbitration = "runtime.arbitration.version"
)

const (
	RuntimePrimaryCodeTimeoutRejected  = "runtime.timeout.parent_budget_rejected"
	RuntimePrimaryCodeTimeoutExhausted = "runtime.timeout.exhausted"
	RuntimePrimaryCodeTimeoutClamped   = "runtime.timeout.parent_budget_clamped"
)

const (
	RuntimeArbitrationRuleVersionA48V1 = "a48.v1"
	RuntimeArbitrationRuleVersionA49V1 = "a49.v1"
	RuntimeArbitrationMaxSecondary     = 3
)

type PrimaryReasonArbitrationInput struct {
	TimeoutParentBudgetRejectTotal int
	TimeoutParentBudgetClampTotal  int
	TimeoutExhaustedTotal          int
	TimeoutResolutionSource        string
	ReadinessFindings              []ReadinessFinding
	RequestedRuleVersion           string
	VersionConfig                  RuntimeArbitrationVersionConfig
}

type PrimaryReasonArbitrationResult struct {
	Domain                string
	Code                  string
	Source                string
	ConflictTotal         int
	SecondaryCodes        []string
	SecondaryCount        int
	RuleVersion           string
	RuleRequestedVersion  string
	RuleEffectiveVersion  string
	RuleVersionSource     string
	RulePolicyAction      string
	RuleUnsupportedTotal  int
	RuleMismatchTotal     int
	RemediationHintCode   string
	RemediationHintDomain string
}

type primaryReasonCandidate struct {
	Domain     string
	Code       string
	Source     string
	Precedence int
}

func ArbitratePrimaryReason(in PrimaryReasonArbitrationInput) PrimaryReasonArbitrationResult {
	resolvedVersion, versionErr := ResolveArbitrationRuleVersion(in.VersionConfig, in.RequestedRuleVersion)
	if versionErr != nil {
		primaryCode := ReadinessCodeArbitrationVersionUnsupported
		if typedErr, ok := versionErr.(*ArbitrationRuleVersionError); ok && typedErr.Code == ArbitrationRuleVersionErrorMismatch {
			primaryCode = ReadinessCodeArbitrationVersionMismatch
		}
		hintCode, hintDomain := mustRemediationHintForPrimaryCode(primaryCode)
		return PrimaryReasonArbitrationResult{
			Domain:                ReadinessDomainRuntime,
			Code:                  primaryCode,
			Source:                RuntimePrimarySourceArbitration,
			RuleVersion:           strings.TrimSpace(resolvedVersion.EffectiveVersion),
			RuleRequestedVersion:  strings.TrimSpace(resolvedVersion.RequestedVersion),
			RuleEffectiveVersion:  strings.TrimSpace(resolvedVersion.EffectiveVersion),
			RuleVersionSource:     strings.TrimSpace(resolvedVersion.VersionSource),
			RulePolicyAction:      strings.TrimSpace(resolvedVersion.PolicyAction),
			RuleUnsupportedTotal:  resolvedVersion.UnsupportedTotal,
			RuleMismatchTotal:     resolvedVersion.MismatchTotal,
			RemediationHintCode:   hintCode,
			RemediationHintDomain: hintDomain,
		}
	}

	candidates := buildPrimaryReasonCandidates(in)
	if len(candidates) == 0 {
		return PrimaryReasonArbitrationResult{
			RuleVersion:          strings.TrimSpace(resolvedVersion.EffectiveVersion),
			RuleRequestedVersion: strings.TrimSpace(resolvedVersion.RequestedVersion),
			RuleEffectiveVersion: strings.TrimSpace(resolvedVersion.EffectiveVersion),
			RuleVersionSource:    strings.TrimSpace(resolvedVersion.VersionSource),
			RulePolicyAction:     strings.TrimSpace(resolvedVersion.PolicyAction),
			RuleUnsupportedTotal: resolvedVersion.UnsupportedTotal,
			RuleMismatchTotal:    resolvedVersion.MismatchTotal,
		}
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
	secondary, secondaryTotal := buildSecondaryReasons(candidates, top[0])
	hintCode, hintDomain := mustRemediationHintForPrimaryCode(top[0].Code)
	return PrimaryReasonArbitrationResult{
		Domain:                top[0].Domain,
		Code:                  top[0].Code,
		Source:                top[0].Source,
		ConflictTotal:         conflict,
		SecondaryCodes:        secondary,
		SecondaryCount:        secondaryTotal,
		RuleVersion:           strings.TrimSpace(resolvedVersion.EffectiveVersion),
		RuleRequestedVersion:  strings.TrimSpace(resolvedVersion.RequestedVersion),
		RuleEffectiveVersion:  strings.TrimSpace(resolvedVersion.EffectiveVersion),
		RuleVersionSource:     strings.TrimSpace(resolvedVersion.VersionSource),
		RulePolicyAction:      strings.TrimSpace(resolvedVersion.PolicyAction),
		RuleUnsupportedTotal:  resolvedVersion.UnsupportedTotal,
		RuleMismatchTotal:     resolvedVersion.MismatchTotal,
		RemediationHintCode:   hintCode,
		RemediationHintDomain: hintDomain,
	}
}

func buildSecondaryReasons(candidates []primaryReasonCandidate, primary primaryReasonCandidate) ([]string, int) {
	if len(candidates) == 0 {
		return nil, 0
	}
	ordered := append([]primaryReasonCandidate(nil), candidates...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Precedence != ordered[j].Precedence {
			return ordered[i].Precedence < ordered[j].Precedence
		}
		if ordered[i].Code != ordered[j].Code {
			return ordered[i].Code < ordered[j].Code
		}
		if ordered[i].Domain != ordered[j].Domain {
			return ordered[i].Domain < ordered[j].Domain
		}
		return ordered[i].Source < ordered[j].Source
	})

	seen := map[string]struct{}{}
	all := make([]string, 0, len(ordered))
	for i := range ordered {
		code := strings.TrimSpace(ordered[i].Code)
		if code == "" || code == primary.Code {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		all = append(all, code)
	}
	if len(all) == 0 {
		return nil, 0
	}
	total := len(all)
	if len(all) > RuntimeArbitrationMaxSecondary {
		all = all[:RuntimeArbitrationMaxSecondary]
	}
	return all, total
}

type remediationHint struct {
	Code   string
	Domain string
}

var remediationHintByPrimaryCode = map[string]remediationHint{
	RuntimePrimaryCodeTimeoutRejected:  {Code: "timeout.adjust_parent_budget", Domain: "timeout"},
	RuntimePrimaryCodeTimeoutExhausted: {Code: "timeout.increase_effective_budget", Domain: "timeout"},
	RuntimePrimaryCodeTimeoutClamped:   {Code: "timeout.adjust_parent_budget", Domain: "timeout"},

	ReadinessCodeConfigInvalid:                 {Code: "runtime.validate_config", Domain: "config"},
	ReadinessCodeStrictEscalated:               {Code: "runtime.relax_strict_mode", Domain: "runtime"},
	ReadinessCodeSchedulerFallback:             {Code: "scheduler.recover_backend", Domain: ReadinessDomainScheduler},
	ReadinessCodeSchedulerActivationError:      {Code: "scheduler.fix_activation", Domain: ReadinessDomainScheduler},
	ReadinessCodeMailboxFallback:               {Code: "mailbox.recover_backend", Domain: ReadinessDomainMailbox},
	ReadinessCodeMailboxActivationError:        {Code: "mailbox.fix_activation", Domain: ReadinessDomainMailbox},
	ReadinessCodeRecoveryFallback:              {Code: "recovery.recover_backend", Domain: ReadinessDomainRecovery},
	ReadinessCodeRecoveryActivationError:       {Code: "recovery.fix_activation", Domain: ReadinessDomainRecovery},
	ReadinessCodeRuntimeManagerUnavailable:     {Code: "runtime.restart_manager", Domain: "runtime"},
	ReadinessCodeArbitrationVersionUnsupported: {Code: "runtime.select_supported_arbitration_version", Domain: "runtime"},
	ReadinessCodeArbitrationVersionMismatch:    {Code: "runtime.align_arbitration_compat_window", Domain: "runtime"},

	ReadinessCodeAdapterRequiredUnavailable: {Code: "adapter.restore_required", Domain: ReadinessDomainAdapter},
	ReadinessCodeAdapterOptionalUnavailable: {Code: "adapter.restore_optional", Domain: ReadinessDomainAdapter},
	ReadinessCodeAdapterDegraded:            {Code: "adapter.investigate_degraded", Domain: ReadinessDomainAdapter},
	ReadinessCodeAdapterRequiredCircuitOpen: {Code: "adapter.reset_required_circuit", Domain: ReadinessDomainAdapter},
	ReadinessCodeAdapterOptionalCircuitOpen: {Code: "adapter.reset_optional_circuit", Domain: ReadinessDomainAdapter},
	ReadinessCodeAdapterHalfOpenDegraded:    {Code: "adapter.investigate_half_open", Domain: ReadinessDomainAdapter},
	ReadinessCodeAdapterGovernanceRecovered: {Code: "adapter.monitor_recovery", Domain: ReadinessDomainAdapter},

	ReadinessAdmissionCodeBypassDisabled:  {Code: "readiness.admission_enable_if_required", Domain: "runtime"},
	ReadinessAdmissionCodeReady:           {Code: "readiness.no_action", Domain: "runtime"},
	ReadinessAdmissionCodeBlocked:         {Code: "readiness.resolve_blocking_findings", Domain: "runtime"},
	ReadinessAdmissionCodeDegradedAllow:   {Code: "readiness.monitor_degraded", Domain: "runtime"},
	ReadinessAdmissionCodeDegradedDeny:    {Code: "readiness.resolve_degraded_findings", Domain: "runtime"},
	ReadinessAdmissionCodeUnknownStatus:   {Code: "readiness.check_status_mapping", Domain: "runtime"},
	ReadinessAdmissionCodeManagerNotReady: {Code: "runtime.restart_manager", Domain: "runtime"},
}

func RemediationHintForPrimaryCode(primaryCode string) (string, string, bool) {
	code := strings.TrimSpace(primaryCode)
	if code == "" {
		return "", "", false
	}
	hint, ok := remediationHintByPrimaryCode[code]
	if !ok {
		return "", "", false
	}
	return hint.Code, hint.Domain, true
}

func mustRemediationHintForPrimaryCode(primaryCode string) (string, string) {
	hintCode, hintDomain, ok := RemediationHintForPrimaryCode(primaryCode)
	if ok {
		return hintCode, hintDomain
	}
	panic(fmt.Sprintf("unsupported primary reason code for remediation hint taxonomy: %q", strings.TrimSpace(primaryCode)))
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
	case ReadinessCodeArbitrationVersionUnsupported, ReadinessCodeArbitrationVersionMismatch:
		return 1
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
