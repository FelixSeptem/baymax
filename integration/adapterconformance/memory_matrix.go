package adapterconformance

import (
	"fmt"
	"sort"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
	memorypkg "github.com/FelixSeptem/baymax/memory"
)

const (
	MemoryDriftProfileUnknown            = "memory_profile_unknown"
	MemoryDriftRequiredCapabilityMissing = "memory_required_capability_missing"
	MemoryDriftFallback                  = "memory_fallback_drift"
	MemoryDriftRunStreamSemantic         = "memory_run_stream_semantic_drift"
	MemoryDriftErrorTaxonomy             = "memory_error_taxonomy_drift"
)

type MemoryProfileSuite struct {
	ProfileID            string
	Provider             string
	ContractVersion      string
	RequiredOperations   []string
	OptionalCapabilities []string
	OfflineDeterministic bool
}

type MemoryOperationCoverageResult struct {
	Accepted           bool
	DriftClass         string
	MissingRequired    []string
	DowngradedOptional []string
}

type MemoryRunStreamObservation struct {
	Operations     []string
	FallbackPolicy string
	FallbackUsed   bool
	ReasonCode     string
}

func MainstreamMemoryProfileMatrix() []MemoryProfileSuite {
	profiles := []string{
		memorypkg.ProfileMem0,
		memorypkg.ProfileZep,
		memorypkg.ProfileOpenViking,
		memorypkg.ProfileGeneric,
	}
	out := make([]MemoryProfileSuite, 0, len(profiles))
	for _, profileID := range profiles {
		profile, err := memorypkg.ResolveProfile(profileID)
		if err != nil {
			continue
		}
		out = append(out, MemoryProfileSuite{
			ProfileID:            profile.ID,
			Provider:             profile.Provider,
			ContractVersion:      profile.ContractVersion,
			RequiredOperations:   append([]string(nil), profile.RequiredOps...),
			OptionalCapabilities: append([]string(nil), profile.OptionalOps...),
			OfflineDeterministic: true,
		})
	}
	return out
}

func ResolveMemoryProfileSuite(profileID string) (MemoryProfileSuite, error) {
	target := strings.ToLower(strings.TrimSpace(profileID))
	for _, row := range MainstreamMemoryProfileMatrix() {
		if row.ProfileID == target {
			return row, nil
		}
	}
	return MemoryProfileSuite{}, &ConformanceError{
		Class:      types.ErrContext,
		ReasonCode: "memory.profile_unknown",
		Message:    fmt.Sprintf("unsupported memory profile in conformance matrix: %q", profileID),
	}
}

func EvaluateMemoryOperationCoverage(required, optional, supported []string) MemoryOperationCoverageResult {
	requiredNorm := normalizeMemoryOperationList(required)
	optionalNorm := normalizeMemoryOperationList(optional)
	supportedSet := map[string]struct{}{}
	for _, item := range normalizeMemoryOperationList(supported) {
		supportedSet[item] = struct{}{}
	}
	missingRequired := make([]string, 0)
	for _, operation := range requiredNorm {
		if _, ok := supportedSet[operation]; ok {
			continue
		}
		missingRequired = append(missingRequired, operation)
	}
	if len(missingRequired) > 0 {
		return MemoryOperationCoverageResult{
			Accepted:        false,
			DriftClass:      MemoryDriftRequiredCapabilityMissing,
			MissingRequired: missingRequired,
		}
	}
	downgradedOptional := make([]string, 0)
	for _, operation := range optionalNorm {
		if _, ok := supportedSet[operation]; ok {
			continue
		}
		downgradedOptional = append(downgradedOptional, operation)
	}
	return MemoryOperationCoverageResult{
		Accepted:           true,
		DriftClass:         "",
		DowngradedOptional: downgradedOptional,
	}
}

func EvaluateMemoryRunStreamEquivalence(run, stream MemoryRunStreamObservation) (string, error) {
	if err := ValidateMemoryFallbackPolicy(run.FallbackPolicy); err != nil {
		return MemoryDriftFallback, err
	}
	if err := ValidateMemoryFallbackPolicy(stream.FallbackPolicy); err != nil {
		return MemoryDriftFallback, err
	}
	if run.FallbackPolicy != stream.FallbackPolicy || run.FallbackUsed != stream.FallbackUsed {
		return MemoryDriftFallback, fmt.Errorf(
			"memory fallback drift run=%s/%v stream=%s/%v",
			run.FallbackPolicy,
			run.FallbackUsed,
			stream.FallbackPolicy,
			stream.FallbackUsed,
		)
	}
	runOps := normalizeMemoryOperationList(run.Operations)
	streamOps := normalizeMemoryOperationList(stream.Operations)
	if !memoryStringSlicesEqual(runOps, streamOps) {
		return MemoryDriftRunStreamSemantic, fmt.Errorf("memory run/stream operation drift run=%v stream=%v", runOps, streamOps)
	}
	runReason := normalizeMemoryReasonCode(run.ReasonCode)
	streamReason := normalizeMemoryReasonCode(stream.ReasonCode)
	if runReason != "" {
		if err := ValidateMemoryErrorTaxonomy(runReason); err != nil {
			return MemoryDriftErrorTaxonomy, err
		}
	}
	if streamReason != "" {
		if err := ValidateMemoryErrorTaxonomy(streamReason); err != nil {
			return MemoryDriftErrorTaxonomy, err
		}
	}
	if runReason != streamReason {
		return MemoryDriftErrorTaxonomy, fmt.Errorf("memory reason taxonomy drift run=%q stream=%q", runReason, streamReason)
	}
	return "", nil
}

func ValidateMemoryFallbackPolicy(policy string) error {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case memorypkg.FallbackPolicyFailFast,
		memorypkg.FallbackPolicyDegradeToBuiltin,
		memorypkg.FallbackPolicyDegradeWithoutMemory:
		return nil
	default:
		return &ConformanceError{
			Class:      types.ErrContext,
			ReasonCode: "memory.fallback.policy_invalid",
			Message:    "memory fallback policy must be fail_fast|degrade_to_builtin|degrade_without_memory",
		}
	}
}

func ValidateMemoryErrorTaxonomy(reason string) error {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(reason)), "memory.") {
		return &ConformanceError{
			Class:      types.ErrContext,
			ReasonCode: "memory.reason_taxonomy_invalid",
			Message:    "memory reason code must use memory.* taxonomy",
		}
	}
	return nil
}

func CanonicalMemoryDriftClasses() []string {
	return []string{
		MemoryDriftProfileUnknown,
		MemoryDriftRequiredCapabilityMissing,
		MemoryDriftFallback,
		MemoryDriftRunStreamSemantic,
		MemoryDriftErrorTaxonomy,
	}
}

func normalizeMemoryOperationList(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, raw := range items {
		item := strings.ToLower(strings.TrimSpace(raw))
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

func normalizeMemoryReasonCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

func memoryStringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
