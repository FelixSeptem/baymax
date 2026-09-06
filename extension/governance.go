// Package extension contains transport-neutral extension governance primitives.
package extension

import (
	"crypto/sha256"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	adaptercap "github.com/FelixSeptem/baymax/adapter/capability"
	adaptermanifest "github.com/FelixSeptem/baymax/adapter/manifest"
)

const (
	KindHook = "hook"
	KindTool = "tool"

	SourceProjectExplicit = "project-explicit"
	SourceProjectAuto     = "project-auto"
	SourceUserExplicit    = "user-explicit"
	SourceUserAuto        = "user-auto"
	SourceBundle          = "bundle"

	ReasonMissingField          = "extension.missing_field"
	ReasonInvalidField          = "extension.invalid_field"
	ReasonAmbiguousConflict     = "extension.ambiguous_conflict"
	ReasonCompatibilityMismatch = "extension.compatibility_mismatch"
)

type ActivationGeneration uint64

const (
	ReasonDiscovery  = "extension.discovery"
	ReasonAdmission  = "extension.admission"
	ReasonActivation = "extension.activation"
	ReasonReload     = "extension.reload"
	ReasonRollback   = "extension.rollback"
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Descriptor is the validated, side-effect-free identity of an extension.
type Descriptor struct {
	Name     string
	Kind     string
	Version  string
	Compat   string
	Digest   string
	Required []string
	Optional []string
	Source   string
}

// Conflict records a lower-precedence candidate that was not selected.
type Conflict struct {
	Identity string
	Winner   Descriptor
	Loser    Descriptor
}

// Resolution contains the deterministic winner set and bounded conflict data.
type Resolution struct {
	Selected  []Descriptor
	Conflicts []Conflict
}

type CapabilityResult struct {
	Accepted   bool
	Downgraded bool
	Reasons    []string
}

type AdmissionOutcome string

const (
	AdmissionAllow AdmissionOutcome = "allow"
	AdmissionDeny  AdmissionOutcome = "deny"
)

type AdmissionInput struct {
	RuntimeVersion        string
	AvailableCapabilities []string
}

type AdmissionDecision struct {
	Outcome   AdmissionOutcome
	Activated bool
}

// Admit validates and negotiates an extension before any activation side effect.
func Admit(descriptor Descriptor, input AdmissionInput) (AdmissionDecision, error) {
	validated, err := ValidateDescriptor(descriptor)
	if err != nil {
		return AdmissionDecision{Outcome: AdmissionDeny}, nil
	}
	if !compatibilityMatches(validated.Compat, input.RuntimeVersion) {
		return AdmissionDecision{Outcome: AdmissionDeny}, &contractError{code: ReasonCompatibilityMismatch, field: "compat", msg: "runtime version does not satisfy compatibility range"}
	}
	if _, err := NegotiateCapabilities(validated, input.AvailableCapabilities, true); err != nil {
		return AdmissionDecision{Outcome: AdmissionDeny}, nil
	}
	return AdmissionDecision{Outcome: AdmissionAllow, Activated: true}, nil
}

func compatibilityMatches(expr, version string) bool {
	if strings.TrimSpace(expr) == "" || strings.TrimSpace(version) == "" {
		return false
	}
	ok, err := adaptermanifest.EvaluateSemverRange(expr, version)
	return err == nil && ok
}

// NegotiateCapabilities reuses adapter capability semantics for extensions.
func NegotiateCapabilities(descriptor Descriptor, available []string, bestEffort bool) (CapabilityResult, error) {
	strategy := adaptercap.StrategyFailFast
	if bestEffort {
		strategy = adaptercap.StrategyBestEffort
	}
	outcome, err := adaptercap.Negotiate(strategy, adaptercap.Set{
		Required: available,
	}, adaptercap.Request{
		Required: descriptor.Required,
		Optional: descriptor.Optional,
	})
	if err != nil {
		return CapabilityResult{}, err
	}
	if !outcome.Accepted {
		return CapabilityResult{}, &contractError{code: adaptercap.ReasonMissingRequired, field: "capabilities.required", msg: strings.Join(outcome.MissingRequired, ",")}
	}
	return CapabilityResult{Accepted: true, Downgraded: outcome.Downgraded, Reasons: outcome.Reasons}, nil
}

// DiscoverFile reads one local descriptor file and derives a bounded digest.
// File discovery is intentionally local and side-effect free; activation is a
// separate concern owned by the caller after resolution/admission succeeds.
func DiscoverFile(path, source string, descriptor Descriptor) (Descriptor, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Descriptor{}, &contractError{code: ReasonInvalidField, field: "path", msg: err.Error()}
	}
	descriptor.Source = source
	descriptor.Digest = "sha256:" + fmt.Sprintf("%x", sha256.Sum256(data))
	return ValidateDescriptor(descriptor)
}

// DiscoverFiles resolves a set of local resources in path order, independent
// of the caller's or filesystem's enumeration order.
func DiscoverFiles(paths []string, source string, descriptors map[string]Descriptor) ([]Descriptor, error) {
	ordered := append([]string(nil), paths...)
	sort.Strings(ordered)
	result := make([]Descriptor, 0, len(ordered))
	for _, path := range ordered {
		descriptor, ok := descriptors[path]
		if !ok {
			return nil, &contractError{code: ReasonInvalidField, field: "path", msg: "descriptor metadata is missing"}
		}
		resolved, err := DiscoverFile(path, source, descriptor)
		if err != nil {
			return nil, err
		}
		result = append(result, resolved)
	}
	return result, nil
}

type contractError struct {
	code  string
	field string
	msg   string
}

func (e *contractError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.code, e.field, e.msg)
}

// ErrorCode extracts a stable extension governance error code.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	}
	if e, ok := err.(*contractError); ok {
		return e.code
	}
	return "extension.error"
}

// ValidateDescriptor validates and normalizes an extension descriptor.
func ValidateDescriptor(in Descriptor) (Descriptor, error) {
	d := normalizeDescriptor(in)
	checks := []struct {
		field string
		value string
	}{
		{"name", d.Name}, {"kind", d.Kind}, {"version", d.Version},
		{"compat", d.Compat}, {"digest", d.Digest}, {"source", d.Source},
	}
	for _, check := range checks {
		if check.value == "" {
			return Descriptor{}, &contractError{code: ReasonMissingField, field: check.field, msg: "required field is missing"}
		}
	}
	if !namePattern.MatchString(d.Name) {
		return Descriptor{}, &contractError{code: ReasonInvalidField, field: "name", msg: "name must match ^[a-z][a-z0-9-]*$"}
	}
	if d.Kind != KindHook && d.Kind != KindTool {
		return Descriptor{}, &contractError{code: ReasonInvalidField, field: "kind", msg: "kind must be hook or tool"}
	}
	if ok, err := adaptermanifest.EvaluateSemverRange("="+d.Version, d.Version); err != nil || !ok {
		return Descriptor{}, &contractError{code: ReasonInvalidField, field: "version", msg: "version must be valid semver"}
	}
	if !strings.HasPrefix(d.Digest, "sha256:") || len(strings.TrimPrefix(d.Digest, "sha256:")) == 0 {
		return Descriptor{}, &contractError{code: ReasonInvalidField, field: "digest", msg: "digest must use sha256:<value> format"}
	}
	if !validSource(d.Source) {
		return Descriptor{}, &contractError{code: ReasonInvalidField, field: "source", msg: "source is not recognized"}
	}
	return d, nil
}

// ResolveResources selects one descriptor per identity using fixed precedence.
func ResolveResources(candidates []Descriptor) (Resolution, error) {
	type item struct {
		d Descriptor
		i int
	}
	items := make([]item, 0, len(candidates))
	for i, candidate := range candidates {
		d, err := ValidateDescriptor(candidate)
		if err != nil {
			return Resolution{}, err
		}
		items = append(items, item{d: d, i: i})
	}
	sort.SliceStable(items, func(i, j int) bool {
		pi, pj := sourceRank(items[i].d.Source), sourceRank(items[j].d.Source)
		if pi != pj {
			return pi < pj
		}
		ki, kj := identity(items[i].d), identity(items[j].d)
		if ki != kj {
			return ki < kj
		}
		if items[i].d.Digest != items[j].d.Digest {
			return items[i].d.Digest < items[j].d.Digest
		}
		return items[i].i < items[j].i
	})
	result := Resolution{Selected: make([]Descriptor, 0, len(items))}
	byIdentity := map[string]Descriptor{}
	for _, current := range items {
		key := identity(current.d)
		winner, exists := byIdentity[key]
		if !exists {
			byIdentity[key] = current.d
			result.Selected = append(result.Selected, current.d)
			continue
		}
		if sourceRank(winner.Source) == sourceRank(current.d.Source) && winner.Digest != current.d.Digest {
			return Resolution{}, &contractError{code: ReasonAmbiguousConflict, field: key, msg: "equal-precedence resources have different digests"}
		}
		result.Conflicts = append(result.Conflicts, Conflict{Identity: key, Winner: winner, Loser: current.d})
	}
	sort.Slice(result.Selected, func(i, j int) bool { return identity(result.Selected[i]) < identity(result.Selected[j]) })
	sort.Slice(result.Conflicts, func(i, j int) bool { return result.Conflicts[i].Identity < result.Conflicts[j].Identity })
	return result, nil
}

func normalizeDescriptor(in Descriptor) Descriptor {
	in.Name = strings.ToLower(strings.TrimSpace(in.Name))
	in.Kind = strings.ToLower(strings.TrimSpace(in.Kind))
	in.Version = strings.TrimSpace(in.Version)
	in.Compat = strings.TrimSpace(in.Compat)
	in.Digest = strings.ToLower(strings.TrimSpace(in.Digest))
	in.Source = strings.ToLower(strings.TrimSpace(in.Source))
	in.Required = normalizeList(in.Required)
	in.Optional = normalizeList(in.Optional)
	return in
}

func normalizeList(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func identity(d Descriptor) string { return d.Kind + ":" + d.Name }

func validSource(source string) bool {
	return source == SourceProjectExplicit || source == SourceProjectAuto || source == SourceUserExplicit || source == SourceUserAuto || source == SourceBundle
}

func sourceRank(source string) int {
	switch source {
	case SourceProjectExplicit:
		return 0
	case SourceProjectAuto:
		return 1
	case SourceUserExplicit:
		return 2
	case SourceUserAuto:
		return 3
	case SourceBundle:
		return 4
	default:
		return 99
	}
}
