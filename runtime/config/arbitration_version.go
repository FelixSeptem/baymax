package config

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	RuntimeArbitrationVersionSourceDefault   = "default"
	RuntimeArbitrationVersionSourceRequested = "requested"

	RuntimeArbitrationVersionPolicyFailFast = "fail_fast"

	RuntimeArbitrationPolicyActionNone                = "none"
	RuntimeArbitrationPolicyActionDisabled            = "disabled"
	RuntimeArbitrationPolicyActionFailFastUnsupported = "fail_fast_unsupported_version"
	RuntimeArbitrationPolicyActionFailFastMismatch    = "fail_fast_version_mismatch"

	ArbitrationRuleVersionErrorUnsupported = "unsupported_version"
	ArbitrationRuleVersionErrorMismatch    = "version_mismatch"
)

type RuntimeArbitrationConfig struct {
	Version RuntimeArbitrationVersionConfig `json:"version"`
}

type RuntimeArbitrationVersionConfig struct {
	Enabled       bool   `json:"enabled"`
	Default       string `json:"default"`
	CompatWindow  int    `json:"compat_window"`
	OnUnsupported string `json:"on_unsupported"`
	OnMismatch    string `json:"on_mismatch"`
}

type ArbitrationRuleVersionResolution struct {
	RequestedVersion string
	EffectiveVersion string
	VersionSource    string
	PolicyAction     string
	UnsupportedTotal int
	MismatchTotal    int
}

type ArbitrationRuleVersionError struct {
	Code             string
	RequestedVersion string
	DefaultVersion   string
	EffectiveVersion string
	VersionSource    string
	PolicyAction     string
	Message          string
}

func (e *ArbitrationRuleVersionError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Message) != "" {
		return strings.TrimSpace(e.Message)
	}
	return fmt.Sprintf(
		"arbitration version resolution failed code=%s requested=%q default=%q effective=%q source=%q action=%q",
		strings.TrimSpace(e.Code),
		strings.TrimSpace(e.RequestedVersion),
		strings.TrimSpace(e.DefaultVersion),
		strings.TrimSpace(e.EffectiveVersion),
		strings.TrimSpace(e.VersionSource),
		strings.TrimSpace(e.PolicyAction),
	)
}

var arbitrationVersionPattern = regexp.MustCompile(`^a(\d+)\.v(\d+)$`)

var runtimeArbitrationRuleRegistry = map[string]struct{}{
	RuntimeArbitrationRuleVersionPrimaryReasonV1:  {},
	RuntimeArbitrationRuleVersionExplainabilityV1: {},
}

func RegisteredRuntimeArbitrationRuleVersions() []string {
	out := make([]string, 0, len(runtimeArbitrationRuleRegistry))
	for version := range runtimeArbitrationRuleRegistry {
		out = append(out, version)
	}
	sort.Slice(out, func(i, j int) bool {
		return compareArbitrationVersion(out[i], out[j]) < 0
	})
	return out
}

func normalizeRuntimeArbitrationVersionConfig(in RuntimeArbitrationVersionConfig) RuntimeArbitrationVersionConfig {
	base := DefaultConfig().Runtime.Arbitration.Version
	out := in
	out.Default = strings.ToLower(strings.TrimSpace(out.Default))
	out.OnUnsupported = strings.ToLower(strings.TrimSpace(out.OnUnsupported))
	out.OnMismatch = strings.ToLower(strings.TrimSpace(out.OnMismatch))
	if strings.TrimSpace(out.Default) == "" {
		out.Default = strings.ToLower(strings.TrimSpace(base.Default))
	}
	if out.CompatWindow < 0 {
		out.CompatWindow = base.CompatWindow
	}
	if out.OnUnsupported == "" {
		out.OnUnsupported = strings.ToLower(strings.TrimSpace(base.OnUnsupported))
	}
	if out.OnMismatch == "" {
		out.OnMismatch = strings.ToLower(strings.TrimSpace(base.OnMismatch))
	}
	return out
}

func ValidateRuntimeArbitrationVersionConfig(cfg RuntimeArbitrationVersionConfig) error {
	if cfg.CompatWindow < 0 {
		return fmt.Errorf("runtime.arbitration.version.compat_window must be >= 0, got %d", cfg.CompatWindow)
	}
	normalized := normalizeRuntimeArbitrationVersionConfig(cfg)
	if !isSupportedArbitrationRuleVersion(normalized.Default) {
		return fmt.Errorf(
			"runtime.arbitration.version.default must be a registered version, got %q",
			cfg.Default,
		)
	}
	switch normalized.OnUnsupported {
	case RuntimeArbitrationVersionPolicyFailFast:
	default:
		return fmt.Errorf(
			"runtime.arbitration.version.on_unsupported must be one of [%s], got %q",
			RuntimeArbitrationVersionPolicyFailFast,
			cfg.OnUnsupported,
		)
	}
	switch normalized.OnMismatch {
	case RuntimeArbitrationVersionPolicyFailFast:
	default:
		return fmt.Errorf(
			"runtime.arbitration.version.on_mismatch must be one of [%s], got %q",
			RuntimeArbitrationVersionPolicyFailFast,
			cfg.OnMismatch,
		)
	}
	return nil
}

func ResolveArbitrationRuleVersion(cfg RuntimeArbitrationVersionConfig, requested string) (ArbitrationRuleVersionResolution, error) {
	normalized := normalizeRuntimeArbitrationVersionConfig(cfg)
	resolution := ArbitrationRuleVersionResolution{
		RequestedVersion: strings.ToLower(strings.TrimSpace(requested)),
		EffectiveVersion: strings.ToLower(strings.TrimSpace(normalized.Default)),
		VersionSource:    RuntimeArbitrationVersionSourceDefault,
		PolicyAction:     RuntimeArbitrationPolicyActionNone,
	}
	if !normalized.Enabled {
		resolution.PolicyAction = RuntimeArbitrationPolicyActionDisabled
		return resolution, nil
	}
	if resolution.RequestedVersion == "" {
		return resolution, nil
	}
	resolution.VersionSource = RuntimeArbitrationVersionSourceRequested
	if !isSupportedArbitrationRuleVersion(resolution.RequestedVersion) {
		resolution.EffectiveVersion = ""
		resolution.PolicyAction = RuntimeArbitrationPolicyActionFailFastUnsupported
		resolution.UnsupportedTotal = 1
		return resolution, &ArbitrationRuleVersionError{
			Code:             ArbitrationRuleVersionErrorUnsupported,
			RequestedVersion: resolution.RequestedVersion,
			DefaultVersion:   normalized.Default,
			EffectiveVersion: resolution.EffectiveVersion,
			VersionSource:    resolution.VersionSource,
			PolicyAction:     resolution.PolicyAction,
			Message: fmt.Sprintf(
				"requested arbitration version %q is unsupported",
				resolution.RequestedVersion,
			),
		}
	}
	if !isWithinArbitrationCompatibilityWindow(normalized.Default, resolution.RequestedVersion, normalized.CompatWindow) {
		resolution.EffectiveVersion = ""
		resolution.PolicyAction = RuntimeArbitrationPolicyActionFailFastMismatch
		resolution.MismatchTotal = 1
		return resolution, &ArbitrationRuleVersionError{
			Code:             ArbitrationRuleVersionErrorMismatch,
			RequestedVersion: resolution.RequestedVersion,
			DefaultVersion:   normalized.Default,
			EffectiveVersion: resolution.EffectiveVersion,
			VersionSource:    resolution.VersionSource,
			PolicyAction:     resolution.PolicyAction,
			Message: fmt.Sprintf(
				"requested arbitration version %q is outside compatibility window=%d from default=%q",
				resolution.RequestedVersion,
				normalized.CompatWindow,
				normalized.Default,
			),
		}
	}
	resolution.EffectiveVersion = resolution.RequestedVersion
	return resolution, nil
}

func isSupportedArbitrationRuleVersion(version string) bool {
	_, ok := runtimeArbitrationRuleRegistry[strings.ToLower(strings.TrimSpace(version))]
	return ok
}

func isWithinArbitrationCompatibilityWindow(defaultVersion, requestedVersion string, window int) bool {
	if window < 0 {
		return false
	}
	ordered := RegisteredRuntimeArbitrationRuleVersions()
	byVersion := make(map[string]int, len(ordered))
	for i := range ordered {
		byVersion[ordered[i]] = i
	}
	defaultIdx, ok := byVersion[strings.ToLower(strings.TrimSpace(defaultVersion))]
	if !ok {
		return false
	}
	requestedIdx, ok := byVersion[strings.ToLower(strings.TrimSpace(requestedVersion))]
	if !ok {
		return false
	}
	delta := defaultIdx - requestedIdx
	if delta < 0 {
		delta = -delta
	}
	return delta <= window
}

func compareArbitrationVersion(left, right string) int {
	left = strings.ToLower(strings.TrimSpace(left))
	right = strings.ToLower(strings.TrimSpace(right))
	if left == right {
		return 0
	}
	la, lv, lok := parseArbitrationVersion(left)
	ra, rv, rok := parseArbitrationVersion(right)
	if lok && rok {
		if la != ra {
			if la < ra {
				return -1
			}
			return 1
		}
		if lv != rv {
			if lv < rv {
				return -1
			}
			return 1
		}
	}
	if left < right {
		return -1
	}
	return 1
}

func parseArbitrationVersion(version string) (int, int, bool) {
	matches := arbitrationVersionPattern.FindStringSubmatch(strings.ToLower(strings.TrimSpace(version)))
	if len(matches) != 3 {
		return 0, 0, false
	}
	a, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, false
	}
	v, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, false
	}
	return a, v, true
}
