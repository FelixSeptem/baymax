package capability

import (
	"fmt"
	"sort"
	"strings"
)

const (
	StrategyFailFast   = "fail_fast"
	StrategyBestEffort = "best_effort"
)

const (
	ReasonMissingRequired       = "adapter.capability.missing_required"
	ReasonOptionalDowngraded    = "adapter.capability.optional_downgraded"
	ReasonStrategyOverrideApply = "adapter.capability.strategy_override_applied"
)

const (
	CodeInvalidStrategy = "adapter.capability.invalid_strategy"
)

type Set struct {
	Required []string
	Optional []string
}

type Request struct {
	Required         []string
	Optional         []string
	StrategyOverride string
}

type Diagnostics struct {
	StrategyApplied         string   `json:"adapter_capability_strategy_applied,omitempty"`
	StrategyOverrideApplied bool     `json:"adapter_capability_strategy_override_applied,omitempty"`
	MissingRequired         []string `json:"adapter_capability_missing_required,omitempty"`
	MissingOptional         []string `json:"adapter_capability_missing_optional,omitempty"`
	DowngradedOptional      []string `json:"adapter_capability_downgraded_optional,omitempty"`
	ReasonCodes             []string `json:"adapter_capability_reason_codes,omitempty"`
}

type Outcome struct {
	Accepted                bool
	Downgraded              bool
	AppliedStrategy         string
	StrategyOverrideApplied bool
	MissingRequired         []string
	MissingOptional         []string
	DowngradedOptional      []string
	Reasons                 []string
	Diagnostics             Diagnostics
}

type NegotiationError struct {
	Code    string
	Message string
}

func (e *NegotiationError) Error() string {
	if e == nil {
		return ""
	}
	return "[" + e.Code + "] " + e.Message
}

func Negotiate(defaultStrategy string, declared Set, request Request) (Outcome, error) {
	declaredNorm := normalizeSet(declared)
	requestNorm := normalizeRequest(request)

	applied, overrideApplied, err := resolveStrategy(defaultStrategy, requestNorm.StrategyOverride)
	if err != nil {
		return Outcome{}, err
	}

	available := union(declaredNorm.Required, declaredNorm.Optional)
	missingRequired := missingCapabilities(requestNorm.Required, available)
	missingOptional := missingCapabilities(requestNorm.Optional, available)

	outcome := Outcome{
		Accepted:                true,
		AppliedStrategy:         applied,
		StrategyOverrideApplied: overrideApplied,
		MissingRequired:         missingRequired,
		MissingOptional:         missingOptional,
	}

	if len(missingRequired) > 0 {
		outcome.Accepted = false
	}

	// fail_fast strategy treats missing optional requests as strict requirements.
	if applied == StrategyFailFast && len(missingOptional) > 0 {
		outcome.Accepted = false
		outcome.MissingRequired = union(outcome.MissingRequired, missingOptional)
	}

	if applied == StrategyBestEffort && len(missingOptional) > 0 {
		outcome.Downgraded = true
		outcome.DowngradedOptional = append([]string(nil), missingOptional...)
	}

	outcome.Reasons = buildReasons(outcome)
	outcome.Diagnostics = Diagnostics{
		StrategyApplied:         outcome.AppliedStrategy,
		StrategyOverrideApplied: outcome.StrategyOverrideApplied,
		MissingRequired:         append([]string(nil), outcome.MissingRequired...),
		MissingOptional:         append([]string(nil), outcome.MissingOptional...),
		DowngradedOptional:      append([]string(nil), outcome.DowngradedOptional...),
		ReasonCodes:             append([]string(nil), outcome.Reasons...),
	}
	return outcome, nil
}

func IsStrategy(in string) bool {
	switch strings.ToLower(strings.TrimSpace(in)) {
	case StrategyFailFast, StrategyBestEffort:
		return true
	default:
		return false
	}
}

func normalizeSet(in Set) Set {
	return Set{
		Required: normalizeCapabilities(in.Required),
		Optional: normalizeCapabilities(in.Optional),
	}
}

func normalizeRequest(in Request) Request {
	return Request{
		Required:         normalizeCapabilities(in.Required),
		Optional:         normalizeCapabilities(in.Optional),
		StrategyOverride: strings.ToLower(strings.TrimSpace(in.StrategyOverride)),
	}
}

func normalizeCapabilities(in []string) []string {
	if in == nil {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, raw := range in {
		key := strings.ToLower(strings.TrimSpace(raw))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func resolveStrategy(defaultStrategy, override string) (string, bool, error) {
	applied := strings.ToLower(strings.TrimSpace(defaultStrategy))
	if applied == "" {
		applied = StrategyFailFast
	}
	if !IsStrategy(applied) {
		return "", false, &NegotiationError{
			Code:    CodeInvalidStrategy,
			Message: fmt.Sprintf("default strategy must be one of [%s,%s]", StrategyFailFast, StrategyBestEffort),
		}
	}
	overrideNorm := strings.ToLower(strings.TrimSpace(override))
	if overrideNorm == "" {
		return applied, false, nil
	}
	if !IsStrategy(overrideNorm) {
		return "", false, &NegotiationError{
			Code:    CodeInvalidStrategy,
			Message: fmt.Sprintf("strategy override must be one of [%s,%s]", StrategyFailFast, StrategyBestEffort),
		}
	}
	if overrideNorm == applied {
		return applied, false, nil
	}
	return overrideNorm, true, nil
}

func union(a, b []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(a)+len(b))
	for _, item := range append(append([]string(nil), a...), b...) {
		key := strings.ToLower(strings.TrimSpace(item))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func missingCapabilities(requested, available []string) []string {
	availableSet := map[string]struct{}{}
	for _, item := range available {
		availableSet[strings.ToLower(strings.TrimSpace(item))] = struct{}{}
	}
	missing := make([]string, 0)
	for _, item := range requested {
		key := strings.ToLower(strings.TrimSpace(item))
		if key == "" {
			continue
		}
		if _, ok := availableSet[key]; ok {
			continue
		}
		missing = append(missing, key)
	}
	sort.Strings(missing)
	return missing
}

func buildReasons(outcome Outcome) []string {
	reasons := make([]string, 0, 3)
	if len(outcome.MissingRequired) > 0 {
		reasons = append(reasons, ReasonMissingRequired)
	}
	if outcome.Downgraded && len(outcome.DowngradedOptional) > 0 {
		reasons = append(reasons, ReasonOptionalDowngraded)
	}
	if outcome.StrategyOverrideApplied {
		reasons = append(reasons, ReasonStrategyOverrideApply)
	}
	return reasons
}
