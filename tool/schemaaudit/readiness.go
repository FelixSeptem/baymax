package schemaaudit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

const ReadinessVersion = "tool_schema_projection_readiness.v1"

const (
	ReadinessStable                = "stable"
	ReadinessTransient             = "transient"
	ReadinessWithinBudget          = "within_budget"
	ReadinessInsufficient          = "insufficient_measurement"
	ReadinessPass                  = "pass"
	ReadinessFail                  = "fail"
	ReadinessMissing               = "missing"
	ReadinessOpportunityAvailable  = "available"
	ReadinessOpportunityAbsent     = "not_available"
	ReadinessNotReady              = "not_ready"
	ReadinessReadyForRuntimeDesign = "ready_for_runtime_design"
	ReadinessBlocked               = "blocked"
)

const (
	CodeReadinessInvalidVersion     = "readiness_invalid_version"
	CodeReadinessOverflow           = "readiness_overflow"
	CodeReadinessDuplicateSample    = "readiness_duplicate_sample"
	CodeReadinessMissingAdmission   = "readiness_missing_admission"
	CodeReadinessUnsupportedSource  = "readiness_unsupported_source"
	CodeReadinessInvalidMeasurement = "readiness_invalid_measurement"
	CodeReadinessPrivacyMaterial    = "readiness_privacy_material"
	CodeReadinessReplayDrift        = "readiness_replay_drift"
	CodeReadinessParityDrift        = "readiness_run_stream_parity_drift"
)

type ReadinessPolicy struct {
	MaxSamples        int     `json:"max_samples,omitempty"`
	MaxTools          int     `json:"max_tools,omitempty"`
	MaxIdentityBytes  int     `json:"max_identity_bytes,omitempty"`
	MaxSchemaBytes    int     `json:"max_schema_bytes,omitempty"`
	MinStableSamples  int     `json:"min_stable_samples,omitempty"`
	SchemaBytesBudget int     `json:"schema_bytes_budget,omitempty"`
	TokenBudget       int     `json:"token_budget,omitempty"`
	QualityMinF1      float64 `json:"quality_min_f1,omitempty"`
}

type ReadinessSample struct {
	Ordinal       int     `json:"ordinal"`
	SchemaBytes   int     `json:"schema_bytes"`
	TokenEstimate int     `json:"token_estimate"`
	F1            float64 `json:"f1,omitempty"`
	ForbiddenHit  float64 `json:"forbidden_hit,omitempty"`
}

type ProjectionOpportunity struct {
	Selected             []string `json:"selected,omitempty"`
	Required             []string `json:"required,omitempty"`
	RequiredCapabilities []string `json:"required_capabilities,omitempty"`
}

type ReadinessInput struct {
	Version     string                `json:"version"`
	SnapshotID  string                `json:"snapshot_id"`
	Stream      bool                  `json:"stream,omitempty"`
	Tools       []AdmittedTool        `json:"tools"`
	Policy      ReadinessPolicy       `json:"policy"`
	Cases       []TaskCase            `json:"cases,omitempty"`
	Samples     []ReadinessSample     `json:"samples,omitempty"`
	Opportunity ProjectionOpportunity `json:"opportunity,omitempty"`
}

type ReadinessResult struct {
	Version              string   `json:"version"`
	SnapshotID           string   `json:"snapshot_id"`
	Digest               string   `json:"digest"`
	WindowDigest         string   `json:"window_digest,omitempty"`
	PressureSignal       string   `json:"pressure_signal"`
	Opportunity          string   `json:"opportunity"`
	SelectionQuality     string   `json:"selection_quality"`
	AdmissionIntegrity   string   `json:"admission_integrity"`
	SemanticCompleteness string   `json:"semantic_completeness"`
	RunStreamParity      string   `json:"run_stream_parity"`
	ReplayDeterminism    string   `json:"replay_determinism"`
	Conclusion           string   `json:"conclusion"`
	StableSampleOrdinals []int    `json:"stable_sample_ordinals,omitempty"`
	Selected             []string `json:"selected,omitempty"`
	BaselineQuality      Quality  `json:"baseline_quality"`
	ProjectedQuality     Quality  `json:"projected_quality"`
	Reasons              []string `json:"reasons,omitempty"`
}

func readinessDefaults(p ReadinessPolicy) ReadinessPolicy {
	if p.MaxSamples == 0 {
		p.MaxSamples = 32
	}
	if p.MaxTools == 0 {
		p.MaxTools = 128
	}
	if p.MaxIdentityBytes == 0 {
		p.MaxIdentityBytes = 128
	}
	if p.MaxSchemaBytes == 0 {
		p.MaxSchemaBytes = 1 << 20
	}
	if p.MinStableSamples == 0 {
		p.MinStableSamples = 2
	}
	if p.QualityMinF1 == 0 {
		p.QualityMinF1 = 0.8
	}
	return p
}

func EvaluateReadiness(in ReadinessInput) (ReadinessResult, error) {
	var zero ReadinessResult
	if in.Version != ReadinessVersion {
		return zero, &Error{Code: CodeReadinessInvalidVersion, Message: "unsupported readiness version"}
	}
	p := readinessDefaults(in.Policy)
	if len(in.Tools) == 0 || len(in.Tools) > p.MaxTools || len(in.Samples) > p.MaxSamples {
		return zero, &Error{Code: CodeReadinessOverflow, Message: "bounded readiness input limit exceeded"}
	}
	tools := append([]AdmittedTool(nil), in.Tools...)
	sort.Slice(tools, func(i, j int) bool { return tools[i].Identity < tools[j].Identity })
	byID := make(map[string]AdmittedTool, len(tools))
	for _, tool := range tools {
		if tool.Identity == "" || len([]byte(tool.Identity)) > p.MaxIdentityBytes {
			return zero, &Error{Code: CodeReadinessOverflow, Message: "tool identity exceeds readiness bound"}
		}
		if _, ok := byID[tool.Identity]; ok {
			return zero, &Error{Code: CodeReadinessOverflow, Message: "duplicate tool identity"}
		}
		if !tool.Admitted {
			return zero, &Error{Code: CodeReadinessMissingAdmission, Message: tool.Identity}
		}
		if isUnsupportedSource(tool.Source) {
			return zero, &Error{Code: CodeReadinessUnsupportedSource, Message: tool.Source}
		}
		if tool.Schema.Bytes < 0 || tool.Schema.Bytes > p.MaxSchemaBytes {
			return zero, &Error{Code: CodeReadinessOverflow, Message: tool.Identity + " schema exceeds readiness bound"}
		}
		byID[tool.Identity] = tool
	}
	window := append([]ReadinessSample(nil), in.Samples...)
	sort.Slice(window, func(i, j int) bool { return window[i].Ordinal < window[j].Ordinal })
	for i, sample := range window {
		if sample.Ordinal <= 0 || sample.SchemaBytes < 0 || sample.TokenEstimate < 0 || sample.F1 < 0 || sample.ForbiddenHit < 0 {
			return zero, &Error{Code: CodeReadinessInvalidMeasurement, Message: "invalid readiness sample"}
		}
		if i > 0 && window[i-1].Ordinal == sample.Ordinal {
			return zero, &Error{Code: CodeReadinessDuplicateSample, Message: "duplicate sample ordinal"}
		}
	}
	windowDigest := readinessWindowDigest(window)
	pressureSignal, stableOrdinals := classifyReadinessWindow(window, p)
	baseline := aggregateQuality(in.Cases, ids(tools))
	selected := append([]string(nil), in.Opportunity.Selected...)
	sort.Strings(selected)
	projected := aggregateQuality(in.Cases, selected)

	result := ReadinessResult{
		Version: ReadinessVersion, SnapshotID: in.SnapshotID, WindowDigest: windowDigest,
		PressureSignal: pressureSignal, Opportunity: ReadinessOpportunityAbsent,
		SelectionQuality: ReadinessMissing, AdmissionIntegrity: ReadinessPass,
		SemanticCompleteness: ReadinessPass, RunStreamParity: ReadinessPass,
		ReplayDeterminism: ReadinessPass, StableSampleOrdinals: stableOrdinals,
		Selected: selected, BaselineQuality: baseline, ProjectedQuality: projected,
	}

	for _, id := range selected {
		if _, ok := byID[id]; !ok {
			result.AdmissionIntegrity = ReadinessFail
			result.Reasons = append(result.Reasons, "selected_tool_not_admitted")
		}
	}
	for _, id := range in.Opportunity.Required {
		if !contains(selected, id) {
			result.AdmissionIntegrity = ReadinessFail
			result.Reasons = append(result.Reasons, "required_tool_omitted")
		}
	}
	for _, capability := range in.Opportunity.RequiredCapabilities {
		found := false
		for _, id := range selected {
			if tool, ok := byID[id]; ok && contains(tool.Capabilities, capability) {
				found = true
				break
			}
		}
		if !found {
			result.SemanticCompleteness = ReadinessFail
			result.Reasons = append(result.Reasons, "required_capability_omitted")
		}
	}
	for _, id := range selected {
		tool := byID[id]
		if tool.Schema.Digest == "" || tool.Schema.EstimatorVersion == "" || tool.Schema.Bytes < 0 {
			result.SemanticCompleteness = ReadinessFail
			result.Reasons = append(result.Reasons, "schema_semantics_incomplete")
		}
	}
	if len(in.Cases) > 0 {
		if projected.F1 >= p.QualityMinF1 && projected.ForbiddenHit == 0 && projected.F1 >= baseline.F1 {
			result.SelectionQuality = ReadinessPass
		} else {
			result.SelectionQuality = ReadinessFail
			result.Reasons = append(result.Reasons, "selection_quality_insufficient")
		}
	}
	basePressure := makePressure(tools, Policy{SchemaBytesBudget: p.SchemaBytesBudget, TokenBudget: p.TokenBudget})
	projectedPressure := makePressure(toolsForIDs(selected, byID), Policy{SchemaBytesBudget: p.SchemaBytesBudget, TokenBudget: p.TokenBudget})
	if len(selected) > 0 && projectedPressure.CanonicalSchemaBytes < basePressure.CanonicalSchemaBytes && projectedPressure.TokenEstimate <= basePressure.TokenEstimate {
		result.Opportunity = ReadinessOpportunityAvailable
	}
	if result.AdmissionIntegrity == ReadinessFail || result.SemanticCompleteness == ReadinessFail {
		result.Conclusion = ReadinessBlocked
	} else if pressureSignal == ReadinessStable && result.Opportunity == ReadinessOpportunityAvailable && result.SelectionQuality == ReadinessPass {
		result.Conclusion = ReadinessReadyForRuntimeDesign
	} else {
		result.Conclusion = ReadinessNotReady
	}
	result.Digest = readinessResultDigest(result)
	return result, nil
}

func classifyReadinessWindow(samples []ReadinessSample, p ReadinessPolicy) (string, []int) {
	if len(samples) == 0 || p.SchemaBytesBudget <= 0 && p.TokenBudget <= 0 {
		return ReadinessInsufficient, nil
	}
	var exceeded []int
	for _, sample := range samples {
		bytesExceeded := p.SchemaBytesBudget > 0 && sample.SchemaBytes > p.SchemaBytesBudget
		tokensExceeded := p.TokenBudget > 0 && sample.TokenEstimate > p.TokenBudget
		if bytesExceeded || tokensExceeded {
			exceeded = append(exceeded, sample.Ordinal)
		}
	}
	if len(exceeded) >= p.MinStableSamples {
		return ReadinessStable, exceeded
	}
	if len(exceeded) > 0 {
		return ReadinessTransient, exceeded
	}
	return ReadinessWithinBudget, nil
}

func readinessWindowDigest(samples []ReadinessSample) string {
	b, _ := json.Marshal(samples)
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:])
}

func readinessResultDigest(result ReadinessResult) string {
	result.Digest = ""
	b, _ := json.Marshal(result)
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:])
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
