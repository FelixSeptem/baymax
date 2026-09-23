// Package schemaaudit provides a bounded, provider-neutral offline audit of
// admitted tool schema pressure and synthetic selection quality. It never
// invokes tools, providers, registries, files, clocks, or network services.
package schemaaudit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

const (
	AuditVersion     = "tool_schema_pressure_selection_audit.v1"
	EstimatorVersion = "utf8_bytes_div4_v1"

	PressureWithinBudget  = "within_budget"
	PressureElevated      = "elevated"
	PressureExceedsBudget = "exceeds_budget"
	PressureInsufficient  = "insufficient_measurement"

	StrategyFullAdmitted       = "full_admitted_set"
	StrategyCapabilityFiltered = "capability_filtered_set"
	StrategyPriorityTopK       = "priority_top_k"
	StrategySourcePartitioned  = "source_partitioned_set"
	StrategyFixtureDeclared    = "fixture_declared_strategy"

	ConclusionBaselineSufficient   = "baseline_sufficient"
	ConclusionPressureOnly         = "pressure_only"
	ConclusionQualityOnly          = "quality_only"
	ConclusionPressureQualityGap   = "pressure_and_quality_gap"
	ConclusionProjectionCandidate  = "projection_candidate"
	ConclusionInsufficientEvidence = "insufficient_evidence"
)

const (
	CodeInvalidVersion       = "invalid_version"
	CodeOverflow             = "overflow"
	CodeDuplicateIdentity    = "duplicate_identity"
	CodeMissingAdmission     = "missing_admission"
	CodeInvalidSchemaFacts   = "invalid_schema_facts"
	CodePrivacyMaterial      = "privacy_material"
	CodeGoldSetConflict      = "gold_set_conflict"
	CodeUnsupportedStrategy  = "unsupported_strategy"
	CodeEmptyStrategy        = "empty_strategy"
	CodeUnknownTool          = "unknown_tool"
	CodeInsufficientMeasure  = "insufficient_measurement"
	CodeReplayNotIdempotent  = "replay_not_idempotent"
	CodeRunStreamParityDrift = "run_stream_parity_drift"
	CodeUnsupportedSource    = "unsupported_source"
)

const SkipReasonNoMatchingTools = "no_matching_tools"

// Error is a stable, machine-readable audit validation error.
type Error struct{ Code, Message string }

func (e *Error) Error() string { return e.Code + ": " + e.Message }
func CodeOf(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

type SchemaFacts struct {
	Digest           string `json:"digest"`
	Bytes            int    `json:"bytes"`
	TokenEstimate    int    `json:"token_estimate"`
	EstimatorVersion string `json:"estimator_version"`
}

type AdmittedTool struct {
	Identity     string      `json:"identity"`
	Source       string      `json:"source"`
	Capabilities []string    `json:"capabilities,omitempty"`
	Priority     int         `json:"priority,omitempty"`
	Admitted     bool        `json:"admitted"`
	Schema       SchemaFacts `json:"schema"`
}

type Policy struct {
	MaxTools              int     `json:"max_tools,omitempty"`
	MaxIdentityBytes      int     `json:"max_identity_bytes,omitempty"`
	MaxSchemaBytes        int     `json:"max_schema_bytes,omitempty"`
	MaxPerToolSchemaBytes int     `json:"max_per_tool_schema_bytes,omitempty"`
	MaxLabelCount         int     `json:"max_label_count,omitempty"`
	MaxStrategies         int     `json:"max_strategies,omitempty"`
	SchemaBytesBudget     int     `json:"schema_bytes_budget,omitempty"`
	TokenBudget           int     `json:"token_budget,omitempty"`
	QualityMinF1          float64 `json:"quality_min_f1,omitempty"`
}

type TaskCase struct {
	ID        string   `json:"id"`
	Intent    string   `json:"intent,omitempty"`
	Expected  []string `json:"expected"`
	Allowed   []string `json:"allowed"`
	Forbidden []string `json:"forbidden"`
	Fallback  []string `json:"fallback,omitempty"`
}

type StrategySpec struct {
	Kind       string   `json:"kind"`
	Capability string   `json:"capability,omitempty"`
	K          int      `json:"k,omitempty"`
	Source     string   `json:"source,omitempty"`
	Declared   []string `json:"declared,omitempty"`
}

type CorpusAdvisory struct {
	Version      string  `json:"version,omitempty"`
	LabeledCases int     `json:"labeled_cases,omitempty"`
	Coverage     float64 `json:"coverage,omitempty"`
	Trend        string  `json:"trend,omitempty"`
}

type Input struct {
	Version    string          `json:"version"`
	SnapshotID string          `json:"snapshot_id"`
	Stream     bool            `json:"stream,omitempty"`
	Tools      []AdmittedTool  `json:"tools"`
	Policy     Policy          `json:"policy"`
	Cases      []TaskCase      `json:"cases"`
	Strategies []StrategySpec  `json:"strategies,omitempty"`
	Corpus     *CorpusAdvisory `json:"corpus,omitempty"`
}

type ToolSize struct {
	Identity      string `json:"identity"`
	Bytes         int    `json:"bytes"`
	TokenEstimate int    `json:"token_estimate"`
}
type Pressure struct {
	AdmittedToolCount    int        `json:"admitted_tool_count"`
	ProjectedToolCount   int        `json:"projected_tool_count"`
	CanonicalSchemaBytes int        `json:"canonical_schema_bytes"`
	TokenEstimate        int        `json:"token_estimate"`
	Level                string     `json:"level"`
	PerTool              []ToolSize `json:"per_tool,omitempty"`
}
type Quality struct {
	Precision        float64 `json:"precision"`
	Recall           float64 `json:"recall"`
	F1               float64 `json:"f1"`
	ExpectedHit      float64 `json:"expected_hit"`
	ForbiddenHit     float64 `json:"forbidden_hit"`
	FallbackCoverage float64 `json:"fallback_coverage"`
}
type StrategyResult struct {
	Kind       string   `json:"kind"`
	Selected   []string `json:"selected"`
	Pressure   Pressure `json:"pressure"`
	Quality    Quality  `json:"quality"`
	SkipReason string   `json:"skip_reason,omitempty"`
}
type CaseResult struct {
	ID         string           `json:"id"`
	Baseline   Quality          `json:"baseline"`
	Strategies []StrategyResult `json:"strategies"`
}
type Result struct {
	Version    string           `json:"version"`
	SnapshotID string           `json:"snapshot_id"`
	Digest     string           `json:"digest"`
	Pressure   Pressure         `json:"pressure"`
	Baseline   Quality          `json:"baseline_quality"`
	Strategies []StrategyResult `json:"strategies"`
	Cases      []CaseResult     `json:"cases"`
	Conclusion string           `json:"conclusion"`
	Reasons    []string         `json:"reasons,omitempty"`
	Corpus     *CorpusAdvisory  `json:"corpus,omitempty"`
}

func defaults(p Policy) Policy {
	if p.MaxTools == 0 {
		p.MaxTools = 128
	}
	if p.MaxIdentityBytes == 0 {
		p.MaxIdentityBytes = 128
	}
	if p.MaxSchemaBytes == 0 {
		p.MaxSchemaBytes = 1 << 20
	}
	if p.MaxPerToolSchemaBytes == 0 {
		p.MaxPerToolSchemaBytes = 256 << 10
	}
	if p.MaxLabelCount == 0 {
		p.MaxLabelCount = 32
	}
	if p.MaxStrategies == 0 {
		p.MaxStrategies = 5
	}
	if p.QualityMinF1 == 0 {
		p.QualityMinF1 = 0.8
	}
	return p
}

// CanonicalSchemaFacts normalizes a JSON-like schema and returns only bounded facts.
func CanonicalSchemaFacts(schema any) (SchemaFacts, error) {
	if containsPrivacy(schema) {
		return SchemaFacts{}, &Error{Code: CodePrivacyMaterial, Message: "schema contains privacy-sensitive material"}
	}
	b, err := json.Marshal(schema)
	if err != nil {
		return SchemaFacts{}, &Error{Code: CodeInvalidSchemaFacts, Message: err.Error()}
	}
	if len(b) == 0 || len(b) > 1<<20 {
		return SchemaFacts{}, &Error{Code: CodeOverflow, Message: "canonical schema exceeds bound"}
	}
	d := sha256.Sum256(b)
	return SchemaFacts{Digest: hex.EncodeToString(d[:]), Bytes: len(b), TokenEstimate: (len(b) + 3) / 4, EstimatorVersion: EstimatorVersion}, nil
}

func containsPrivacy(v any) bool {
	switch x := v.(type) {
	case map[string]any:
		for k, value := range x {
			lk := strings.ToLower(k)
			for _, bad := range []string{"endpoint", "credential", "password", "secret", "token", "prompt", "response", "tool_result"} {
				if strings.Contains(lk, bad) {
					return true
				}
			}
			if containsPrivacy(value) {
				return true
			}
		}
	case []any:
		for _, value := range x {
			if containsPrivacy(value) {
				return true
			}
		}
	}
	return false
}

func Audit(in Input) (Result, error) {
	var zero Result
	if in.Version != AuditVersion {
		return zero, &Error{Code: CodeInvalidVersion, Message: "unsupported audit version"}
	}
	p := defaults(in.Policy)
	if len(in.Tools) == 0 || len(in.Tools) > p.MaxTools || len(in.Cases) == 0 || len(in.Strategies) > p.MaxStrategies {
		return zero, &Error{Code: CodeOverflow, Message: "bounded input limit exceeded"}
	}
	tools := append([]AdmittedTool(nil), in.Tools...)
	sort.Slice(tools, func(i, j int) bool { return tools[i].Identity < tools[j].Identity })
	byID := make(map[string]AdmittedTool, len(tools))
	totalBytes := 0
	insufficientMeasurement := false
	for _, tool := range tools {
		if tool.Identity == "" || len([]byte(tool.Identity)) > p.MaxIdentityBytes {
			return zero, &Error{Code: CodeOverflow, Message: "tool identity exceeds bound"}
		}
		if _, ok := byID[tool.Identity]; ok {
			return zero, &Error{Code: CodeDuplicateIdentity, Message: tool.Identity}
		}
		if !tool.Admitted {
			return zero, &Error{Code: CodeMissingAdmission, Message: tool.Identity}
		}
		if isUnsupportedSource(tool.Source) {
			return zero, &Error{Code: CodeUnsupportedSource, Message: tool.Source}
		}
		if tool.Schema.Bytes < 0 || tool.Schema.Bytes > p.MaxPerToolSchemaBytes {
			return zero, &Error{Code: CodeOverflow, Message: tool.Identity + " schema exceeds bound"}
		}
		if tool.Schema.Digest == "" || len(tool.Schema.Digest) != 64 || tool.Schema.EstimatorVersion != EstimatorVersion || tool.Schema.TokenEstimate != (tool.Schema.Bytes+3)/4 {
			insufficientMeasurement = true
		}
		if len(tool.Capabilities) > p.MaxLabelCount {
			return zero, &Error{Code: CodeOverflow, Message: "capability label bound exceeded"}
		}
		byID[tool.Identity] = tool
		totalBytes += tool.Schema.Bytes
	}
	if totalBytes > p.MaxSchemaBytes {
		return zero, &Error{Code: CodeOverflow, Message: "schema bytes bound exceeded"}
	}
	strategies := in.Strategies
	if len(strategies) == 0 {
		strategies = []StrategySpec{{Kind: StrategyFullAdmitted}}
	}
	for _, c := range in.Cases {
		if err := validateCase(c, byID, p); err != nil {
			return zero, err
		}
	}
	basePressure := makePressure(tools, p)
	if insufficientMeasurement {
		basePressure.Level = PressureInsufficient
	}
	baseQ := aggregateQuality(in.Cases, ids(tools))
	results := make([]StrategyResult, 0, len(strategies))
	improved := false
	for _, spec := range strategies {
		selected, err := selectTools(spec, tools, byID, p)
		if err != nil {
			return zero, err
		}
		q := aggregateQuality(in.Cases, selected)
		pr := makePressure(toolsForIDs(selected, byID), p)
		if insufficientMeasurement {
			pr.Level = PressureInsufficient
		}
		if pr.CanonicalSchemaBytes < basePressure.CanonicalSchemaBytes && q.F1 > baseQ.F1 && q.ForbiddenHit == 0 {
			improved = true
		}
		skipReason := ""
		if len(selected) == 0 {
			skipReason = SkipReasonNoMatchingTools
		}
		results = append(results, StrategyResult{Kind: spec.Kind, Selected: selected, Pressure: pr, Quality: q, SkipReason: skipReason})
	}
	conclusion := ConclusionBaselineSufficient
	pressureGap := basePressure.Level == PressureExceedsBudget
	qualityGap := baseQ.F1 < p.QualityMinF1 || baseQ.ForbiddenHit > 0
	switch {
	case pressureGap && qualityGap:
		if improved {
			conclusion = ConclusionProjectionCandidate
		} else {
			conclusion = ConclusionPressureQualityGap
		}
	case pressureGap:
		conclusion = ConclusionPressureOnly
	case qualityGap:
		conclusion = ConclusionQualityOnly
	}
	result := Result{Version: AuditVersion, SnapshotID: in.SnapshotID, Pressure: basePressure, Baseline: baseQ, Strategies: results, Conclusion: conclusion, Corpus: in.Corpus}
	for _, c := range in.Cases {
		cr := CaseResult{ID: c.ID, Baseline: qualityFor(c, ids(tools))}
		for i, spec := range strategies {
			selected, _ := selectTools(spec, tools, byID, p)
			cr.Strategies = append(cr.Strategies, StrategyResult{Kind: spec.Kind, Selected: selected, Pressure: results[i].Pressure, Quality: qualityFor(c, selected), SkipReason: results[i].SkipReason})
		}
		result.Cases = append(result.Cases, cr)
	}
	result.Digest = resultDigest(result)
	return result, nil
}

func isUnsupportedSource(source string) bool {
	switch strings.ToLower(source) {
	case "remote", "marketplace", "download", "network":
		return true
	default:
		return false
	}
}

func aggregateQuality(cases []TaskCase, selected []string) Quality {
	if len(cases) == 0 {
		return Quality{}
	}
	var out Quality
	for _, c := range cases {
		q := qualityFor(c, selected)
		out.Precision += q.Precision
		out.Recall += q.Recall
		out.F1 += q.F1
		out.ExpectedHit += q.ExpectedHit
		out.ForbiddenHit += q.ForbiddenHit
		out.FallbackCoverage += q.FallbackCoverage
	}
	n := float64(len(cases))
	out.Precision /= n
	out.Recall /= n
	out.F1 /= n
	out.ExpectedHit /= n
	out.ForbiddenHit /= n
	out.FallbackCoverage /= n
	return out
}

func validateCase(c TaskCase, byID map[string]AdmittedTool, p Policy) error {
	if c.ID == "" || len(c.Expected)+len(c.Allowed)+len(c.Forbidden)+len(c.Fallback) > p.MaxLabelCount*4 {
		return &Error{Code: CodeOverflow, Message: "gold label bound exceeded"}
	}
	expected := make(map[string]bool, len(c.Expected))
	for _, id := range c.Expected {
		expected[id] = true
	}
	for _, id := range c.Forbidden {
		if expected[id] {
			return &Error{Code: CodeGoldSetConflict, Message: id}
		}
	}
	for _, values := range [][]string{c.Expected, c.Allowed, c.Forbidden, c.Fallback} {
		for _, id := range values {
			if _, ok := byID[id]; !ok {
				return &Error{Code: CodeUnknownTool, Message: id}
			}
		}
	}
	return nil
}

func ids(tools []AdmittedTool) []string {
	out := make([]string, len(tools))
	for i, t := range tools {
		out[i] = t.Identity
	}
	return out
}

func toolsForIDs(selected []string, byID map[string]AdmittedTool) []AdmittedTool {
	out := make([]AdmittedTool, 0, len(selected))
	for _, id := range selected {
		if tool, ok := byID[id]; ok {
			out = append(out, tool)
		}
	}
	return out
}
func selectTools(s StrategySpec, tools []AdmittedTool, byID map[string]AdmittedTool, p Policy) ([]string, error) {
	selected := make([]AdmittedTool, 0, len(tools))
	switch s.Kind {
	case StrategyFullAdmitted:
		selected = append(selected, tools...)
	case StrategyCapabilityFiltered:
		if s.Capability == "" {
			return nil, &Error{Code: CodeEmptyStrategy, Message: "capability is required"}
		}
		for _, t := range tools {
			for _, c := range t.Capabilities {
				if c == s.Capability {
					selected = append(selected, t)
					break
				}
			}
		}
	case StrategyPriorityTopK:
		if s.K <= 0 || s.K > p.MaxTools {
			return nil, &Error{Code: CodeEmptyStrategy, Message: "k is out of bounds"}
		}
		selected = append(selected, tools...)
		sort.SliceStable(selected, func(i, j int) bool {
			if selected[i].Priority != selected[j].Priority {
				return selected[i].Priority > selected[j].Priority
			}
			return selected[i].Identity < selected[j].Identity
		})
		selected = selected[:min(s.K, len(selected))]
	case StrategySourcePartitioned:
		if s.Source == "" {
			return nil, &Error{Code: CodeEmptyStrategy, Message: "source is required"}
		}
		for _, t := range tools {
			if t.Source == s.Source {
				selected = append(selected, t)
			}
		}
	case StrategyFixtureDeclared:
		if len(s.Declared) == 0 || len(s.Declared) > p.MaxTools {
			return nil, &Error{Code: CodeEmptyStrategy, Message: "declared set is empty or too large"}
		}
		seen := map[string]bool{}
		for _, id := range s.Declared {
			if seen[id] {
				return nil, &Error{Code: CodeDuplicateIdentity, Message: id}
			}
			seen[id] = true
			t, ok := byID[id]
			if !ok {
				return nil, &Error{Code: CodeUnknownTool, Message: id}
			}
			selected = append(selected, t)
		}
	default:
		return nil, &Error{Code: CodeUnsupportedStrategy, Message: s.Kind}
	}
	return ids(selected), nil
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func makePressure(tools []AdmittedTool, p Policy) Pressure {
	out := Pressure{AdmittedToolCount: len(tools), ProjectedToolCount: len(tools), Level: PressureWithinBudget}
	for _, t := range tools {
		out.CanonicalSchemaBytes += t.Schema.Bytes
		out.TokenEstimate += t.Schema.TokenEstimate
		out.PerTool = append(out.PerTool, ToolSize{Identity: t.Identity, Bytes: t.Schema.Bytes, TokenEstimate: t.Schema.TokenEstimate})
	}
	if (p.SchemaBytesBudget > 0 && out.CanonicalSchemaBytes > p.SchemaBytesBudget) || (p.TokenBudget > 0 && out.TokenEstimate > p.TokenBudget) {
		out.Level = PressureExceedsBudget
	} else if (p.SchemaBytesBudget > 0 && out.CanonicalSchemaBytes*10 > p.SchemaBytesBudget*8) || (p.TokenBudget > 0 && out.TokenEstimate*10 > p.TokenBudget*8) {
		out.Level = PressureElevated
	}
	return out
}
func qualityFor(c TaskCase, selected []string) Quality {
	set := map[string]bool{}
	for _, id := range selected {
		set[id] = true
	}
	expected := map[string]bool{}
	for _, id := range c.Expected {
		expected[id] = true
	}
	forbidden := map[string]bool{}
	for _, id := range c.Forbidden {
		forbidden[id] = true
	}
	fallback := map[string]bool{}
	for _, id := range c.Fallback {
		fallback[id] = true
	}
	tp := 0
	for id := range expected {
		if set[id] {
			tp++
		}
	}
	fp := 0
	for id := range set {
		if !expected[id] {
			fp++
		}
	}
	fh := 0
	for id := range forbidden {
		if set[id] {
			fh++
		}
	}
	fc := 0
	for id := range fallback {
		if set[id] {
			fc++
		}
	}
	precision, recall := 0.0, 0.0
	if len(selected) > 0 {
		precision = float64(tp) / float64(len(selected))
	}
	if len(expected) > 0 {
		recall = float64(tp) / float64(len(expected))
	}
	f1 := 0.0
	if precision+recall > 0 {
		f1 = 2 * precision * recall / (precision + recall)
	}
	return Quality{Precision: precision, Recall: recall, F1: f1, ExpectedHit: float64(tp), ForbiddenHit: float64(fh), FallbackCoverage: ratio(fc, len(fallback))}
}
func ratio(a, b int) float64 {
	if b == 0 {
		return 1
	}
	return float64(a) / float64(b)
}
func resultDigest(r Result) string {
	r.Digest = ""
	b, _ := json.Marshal(r)
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:])
}
