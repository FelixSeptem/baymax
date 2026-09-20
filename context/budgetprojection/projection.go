// Package budgetprojection defines a versioned, bounded, read-only derived
// projection of budget facts that are owned elsewhere.
//
// The package intentionally depends on nothing but the standard library: budget
// facts are produced by `runtime/config` (ReAct iteration/tool-call limits and
// readiness admission), `orchestration/scheduler` (parent remaining budget) and
// `core/runner` (ReAct plan notebook). This package never owns those facts,
// never writes them back, and never establishes a second budget ledger.
package budgetprojection

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// FixtureVersionBudgetV1 is the only accepted fixture/protocol version.
const FixtureVersionBudgetV1 = "budget_projection.v1"

// Bounded limits. Exceeding any of them fails fast instead of truncating.
const (
	MaxProjectionBytes = 4096
	MaxNotes           = 8
	MaxNoteChars       = 120
	MaxCases           = 64
	MaxBenchmarks      = 16
	MaxTraces          = 32
	MaxStepsPerTrace   = 64
	MaxFixtureBytes    = 2 << 20 // 2 MiB
)

// Stable drift / gap classification codes.
const (
	CodeSchemaDrift                = "budget_projection_schema_drift"
	CodeUnknownVersion             = "budget_projection_unknown_version"
	CodeMissingIterationLimit      = "budget_facts_missing_iteration_limit"
	CodeMissingToolCallLimit       = "budget_facts_missing_tool_call_limit"
	CodeMissingTimeBudget          = "budget_facts_missing_time_budget"
	CodeMissingCostThreshold       = "budget_facts_missing_cost_threshold"
	CodeNegativeRemaining          = "budget_projection_negative_remaining"
	CodeRatioOutOfRange            = "budget_projection_ratio_out_of_range"
	CodePressureLevelMismatch      = "budget_projection_pressure_level_mismatch"
	CodeNoteUnbounded              = "budget_projection_note_unbounded"
	CodeDigestMismatch             = "budget_projection_digest_mismatch"
	CodeOverflowDrift              = "budget_projection_overflow_drift"
	CodeWritebackShapeDetected     = "budget_projection_writeback_shape_detected"
	CodeBenchmarkSchemaDrift       = "budget_benchmark_schema_drift"
	CodeBenchmarkMetricMismatch    = "budget_benchmark_metric_mismatch"
	CodeBenchmarkRecoveryRecompute = "budget_benchmark_recovery_recompute_drift"
	CodeBenchmarkOverflowDrift     = "budget_benchmark_overflow_drift"
	CodeReplayNotIdempotent        = "budget_projection_replay_not_idempotent"
)

// BudgetProjectionTaxonomy is the canonical, ordered set of classification codes.
var BudgetProjectionTaxonomy = []string{
	CodeSchemaDrift,
	CodeUnknownVersion,
	CodeMissingIterationLimit,
	CodeMissingToolCallLimit,
	CodeMissingTimeBudget,
	CodeMissingCostThreshold,
	CodeNegativeRemaining,
	CodeRatioOutOfRange,
	CodePressureLevelMismatch,
	CodeNoteUnbounded,
	CodeDigestMismatch,
	CodeOverflowDrift,
	CodeWritebackShapeDetected,
	CodeBenchmarkSchemaDrift,
	CodeBenchmarkMetricMismatch,
	CodeBenchmarkRecoveryRecompute,
	CodeBenchmarkOverflowDrift,
	CodeReplayNotIdempotent,
}

// BudgetPressureLevel is a derived, non-authoritative expression of budget usage.
type BudgetPressureLevel string

const (
	PressureNone      BudgetPressureLevel = "none"
	PressureNormal    BudgetPressureLevel = "normal"
	PressureElevated  BudgetPressureLevel = "elevated"
	PressureCritical  BudgetPressureLevel = "critical"
	PressureSaturated BudgetPressureLevel = "saturated"
)

// BudgetPressureLevels is the canonical ordered set of pressure levels.
var BudgetPressureLevels = []BudgetPressureLevel{
	PressureNone,
	PressureNormal,
	PressureElevated,
	PressureCritical,
	PressureSaturated,
}

// BudgetLimitUsage carries a limit/used pair owned by an upstream budget owner.
type BudgetLimitUsage struct {
	Limit int `json:"limit"`
	Used  int `json:"used"`
}

// BudgetTimeFacts carries the wall-clock budget facts (milliseconds).
type BudgetTimeFacts struct {
	BudgetMS  int64 `json:"budget_ms"`
	ElapsedMS int64 `json:"elapsed_ms"`
}

// BudgetCostFacts carries the admission cost estimate and thresholds.
type BudgetCostFacts struct {
	EstimateTotal    float64 `json:"estimate_total"`
	HardThreshold    float64 `json:"hard_threshold"`
	DegradeThreshold float64 `json:"degrade_threshold"`
}

// BudgetAdmissionFacts carries the readiness admission decision.
type BudgetAdmissionFacts struct {
	Decision      string `json:"decision"`
	DegradeAction string `json:"degrade_action,omitempty"`
}

// BudgetParentFacts carries the scheduler parent remaining budget.
type BudgetParentFacts struct {
	RemainingMS int64 `json:"remaining_ms"`
}

// BudgetPlanFacts carries the ReAct plan notebook lifecycle counters.
type BudgetPlanFacts struct {
	ChangeTotal  int    `json:"change_total"`
	RecoverCount int    `json:"recover_count"`
	Status       string `json:"status,omitempty"`
}

// BudgetFacts is a read-only snapshot of source-owned budget facts. Every
// dimension is a pointer so that "absent" is distinguishable from "zero".
type BudgetFacts struct {
	Iteration *BudgetLimitUsage    `json:"iteration,omitempty"`
	ToolCall  *BudgetLimitUsage    `json:"tool_call,omitempty"`
	Time      *BudgetTimeFacts     `json:"time,omitempty"`
	Cost      *BudgetCostFacts     `json:"cost,omitempty"`
	Admission BudgetAdmissionFacts `json:"admission"`
	Parent    *BudgetParentFacts   `json:"parent,omitempty"`
	Plan      *BudgetPlanFacts     `json:"plan,omitempty"`
}

// BudgetDimension is the derived remaining view of a single dimension. When
// Available is false the remaining value MUST NOT be interpreted.
type BudgetDimension struct {
	Available bool     `json:"available"`
	Remaining *float64 `json:"remaining,omitempty"`
	Limit     *float64 `json:"limit,omitempty"`
	Used      *float64 `json:"used,omitempty"`
	Ratio     *float64 `json:"ratio,omitempty"`
}

// BudgetRemainingProjection is the bounded derived remaining view.
type BudgetRemainingProjection struct {
	Iteration    BudgetDimension `json:"iteration"`
	ToolCall     BudgetDimension `json:"tool_call"`
	TimeMS       BudgetDimension `json:"time_ms"`
	CostHeadroom BudgetDimension `json:"cost_headroom"`
}

// BudgetProjection is the bounded, read-only derived projection.
type BudgetProjection struct {
	Version   string                    `json:"version"`
	Complete  bool                      `json:"complete"`
	Remaining BudgetRemainingProjection `json:"remaining"`
	Pressure  BudgetPressureLevel       `json:"pressure"`
	Notes     []string                  `json:"notes"`
	Digest    string                    `json:"digest"`
}

// Error is a deterministic validation error carrying a stable code.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}

func newError(code string, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// DeriveBudgetProjection computes the bounded read-only projection from a
// source-owned facts snapshot. It is a pure function: no clock, no globals, no
// side effects, and no write-back path.
func DeriveBudgetProjection(facts BudgetFacts) BudgetProjection {
	projection := BudgetProjection{
		Version:  FixtureVersionBudgetV1,
		Complete: true,
		Notes:    []string{},
	}

	projection.Remaining.Iteration, projection.Complete = deriveLimitDimension(
		facts.Iteration, projection.Complete,
	)
	projection.Remaining.ToolCall, projection.Complete = deriveLimitDimension(
		facts.ToolCall, projection.Complete,
	)
	projection.Remaining.TimeMS = deriveTimeDimension(facts.Time)
	projection.Remaining.CostHeadroom = deriveCostDimension(facts.Cost)
	if !projection.Remaining.TimeMS.Available || !projection.Remaining.CostHeadroom.Available {
		projection.Complete = false
	}

	projection.Pressure = derivePressureLevel(projection.Remaining)
	projection.Notes = deriveNotes(facts, projection)
	projection.Digest = BudgetProjectionDigest(projection)
	return projection
}

func deriveLimitDimension(usage *BudgetLimitUsage, complete bool) (BudgetDimension, bool) {
	if usage == nil || usage.Limit <= 0 {
		return BudgetDimension{}, false
	}
	dimension := BudgetDimension{
		Available: true,
		Limit:     float64Pointer(float64(usage.Limit)),
		Used:      float64Pointer(float64(usage.Used)),
		Remaining: float64Pointer(float64(usage.Limit - usage.Used)),
	}
	if usage.Limit > 0 {
		dimension.Ratio = float64Pointer(float64(usage.Used) / float64(usage.Limit))
	}
	return dimension, complete
}

func deriveTimeDimension(facts *BudgetTimeFacts) BudgetDimension {
	if facts == nil || facts.BudgetMS <= 0 {
		return BudgetDimension{}
	}
	dimension := BudgetDimension{
		Available: true,
		Limit:     float64Pointer(float64(facts.BudgetMS)),
		Used:      float64Pointer(float64(facts.ElapsedMS)),
		Remaining: float64Pointer(float64(facts.BudgetMS - facts.ElapsedMS)),
		Ratio:     float64Pointer(float64(facts.ElapsedMS) / float64(facts.BudgetMS)),
	}
	return dimension
}

func deriveCostDimension(facts *BudgetCostFacts) BudgetDimension {
	if facts == nil || facts.HardThreshold <= 0 {
		return BudgetDimension{}
	}
	dimension := BudgetDimension{
		Available: true,
		Limit:     float64Pointer(facts.HardThreshold),
		Used:      float64Pointer(facts.EstimateTotal),
		Remaining: float64Pointer(facts.HardThreshold - facts.EstimateTotal),
		Ratio:     float64Pointer(facts.EstimateTotal / facts.HardThreshold),
	}
	return dimension
}

// derivePressureLevel derives a non-authoritative pressure expression from the
// highest available usage ratio. Absent dimensions never contribute.
func derivePressureLevel(remaining BudgetRemainingProjection) BudgetPressureLevel {
	ratios := make([]float64, 0, 4)
	for _, dimension := range []BudgetDimension{
		remaining.Iteration,
		remaining.ToolCall,
		remaining.TimeMS,
		remaining.CostHeadroom,
	} {
		if dimension.Available {
			ratios = append(ratios, float64Value(dimension.Ratio))
		}
	}
	if len(ratios) == 0 {
		return PressureNone
	}
	highest := ratios[0]
	for _, ratio := range ratios[1:] {
		if ratio > highest {
			highest = ratio
		}
	}
	switch {
	case highest >= 1:
		return PressureSaturated
	case highest >= 0.8:
		return PressureCritical
	case highest >= 0.5:
		return PressureElevated
	default:
		return PressureNormal
	}
}

// deriveNotes emits bounded, deterministically ordered notes. Only absence or
// anomaly produces a note; healthy dimensions stay silent.
func deriveNotes(facts BudgetFacts, projection BudgetProjection) []string {
	notes := make([]string, 0, MaxNotes)
	if !projection.Remaining.Iteration.Available {
		notes = append(notes, "iteration_limit_unavailable")
	}
	if !projection.Remaining.ToolCall.Available {
		notes = append(notes, "tool_call_limit_unavailable")
	}
	if !projection.Remaining.TimeMS.Available {
		notes = append(notes, "time_budget_unavailable")
	}
	if !projection.Remaining.CostHeadroom.Available {
		notes = append(notes, "cost_threshold_unavailable")
	}
	for _, entry := range []struct {
		name      string
		dimension BudgetDimension
	}{
		{"iteration", projection.Remaining.Iteration},
		{"tool_call", projection.Remaining.ToolCall},
		{"time", projection.Remaining.TimeMS},
		{"cost", projection.Remaining.CostHeadroom},
	} {
		if entry.dimension.Available && float64Value(entry.dimension.Remaining) < 0 {
			notes = append(notes, entry.name+"_remaining_negative")
		}
	}
	if decision := strings.TrimSpace(facts.Admission.Decision); decision != "" && decision != "allow" {
		note := "admission_decision=" + decision
		if len(note) > MaxNoteChars {
			note = "admission_decision=unavailable"
		}
		notes = append(notes, note)
	}
	if facts.Parent != nil && facts.Parent.RemainingMS <= 0 {
		notes = append(notes, "parent_remaining_exhausted")
	}
	if facts.Plan != nil && facts.Plan.RecoverCount > 0 {
		notes = append(notes, "plan_recover_count="+strconv.Itoa(facts.Plan.RecoverCount))
	}
	if projection.Pressure == PressureCritical || projection.Pressure == PressureSaturated {
		notes = append(notes, "pressure="+string(projection.Pressure))
	}
	if len(notes) == 0 {
		notes = append(notes, "none")
	}
	sort.Strings(notes)
	return notes
}

// BudgetProjectionDigest returns the SHA-256 digest of the canonical projection
// serialization. The Digest field itself is excluded.
func BudgetProjectionDigest(projection BudgetProjection) string {
	sum := sha256.Sum256([]byte(canonicalBudgetProjection(projection)))
	return hex.EncodeToString(sum[:])
}

func canonicalBudgetProjection(projection BudgetProjection) string {
	var builder strings.Builder
	builder.WriteString("budget_projection.v1")
	builder.WriteString("|version=")
	builder.WriteString(lengthPrefixed(projection.Version))
	builder.WriteString("|complete=")
	builder.WriteString(strconv.FormatBool(projection.Complete))
	for _, entry := range []struct {
		name      string
		dimension BudgetDimension
	}{
		{"iteration", projection.Remaining.Iteration},
		{"tool_call", projection.Remaining.ToolCall},
		{"time_ms", projection.Remaining.TimeMS},
		{"cost_headroom", projection.Remaining.CostHeadroom},
	} {
		builder.WriteString("|dim=")
		builder.WriteString(entry.name)
		builder.WriteString(":")
		builder.WriteString(strconv.FormatBool(entry.dimension.Available))
		builder.WriteString(":")
		builder.WriteString(formatOptionalFloat(entry.dimension.Limit))
		builder.WriteString(":")
		builder.WriteString(formatOptionalFloat(entry.dimension.Used))
		builder.WriteString(":")
		builder.WriteString(formatOptionalFloat(entry.dimension.Remaining))
		builder.WriteString(":")
		builder.WriteString(formatOptionalFloat(entry.dimension.Ratio))
	}
	builder.WriteString("|pressure=")
	builder.WriteString(string(projection.Pressure))
	builder.WriteString("|notes=")
	builder.WriteString(strconv.Itoa(len(projection.Notes)))
	for _, note := range projection.Notes {
		builder.WriteString(":")
		builder.WriteString(lengthPrefixed(note))
	}
	return builder.String()
}

func lengthPrefixed(value string) string {
	return strconv.Itoa(len(value)) + ":" + value
}

func formatFloat(value float64) string {
	if value == 0 {
		return "0"
	}
	return strconv.FormatFloat(value, 'f', 4, 64)
}

func float64Pointer(value float64) *float64 {
	return &value
}

func float64Value(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func formatOptionalFloat(value *float64) string {
	if value == nil {
		return "null"
	}
	return formatFloat(*value)
}

// ClassifyBudgetProjection validates a projection shape and returns stable codes.
// It never mutates the input and never reports a truncated success.
func ClassifyBudgetProjection(projection BudgetProjection) []string {
	codes := make([]string, 0, 4)
	if len(canonicalBudgetProjection(projection)) > MaxProjectionBytes {
		codes = append(codes, CodeOverflowDrift)
	}
	if projection.Version != FixtureVersionBudgetV1 {
		codes = append(codes, CodeUnknownVersion)
	}
	if !isKnownPressureLevel(projection.Pressure) {
		codes = append(codes, CodeSchemaDrift)
	}
	if len(projection.Notes) > MaxNotes {
		codes = append(codes, CodeNoteUnbounded)
	}
	for _, note := range projection.Notes {
		if len(note) > MaxNoteChars {
			codes = append(codes, CodeNoteUnbounded)
			break
		}
	}
	for _, entry := range []struct {
		code      string
		dimension BudgetDimension
	}{
		{CodeMissingIterationLimit, projection.Remaining.Iteration},
		{CodeMissingToolCallLimit, projection.Remaining.ToolCall},
		{CodeMissingTimeBudget, projection.Remaining.TimeMS},
		{CodeMissingCostThreshold, projection.Remaining.CostHeadroom},
	} {
		if !entry.dimension.Available {
			codes = append(codes, entry.code)
			if entry.dimension.Limit != nil || entry.dimension.Used != nil ||
				entry.dimension.Remaining != nil || entry.dimension.Ratio != nil {
				codes = append(codes, CodeSchemaDrift)
			}
		}
		if entry.dimension.Available {
			if entry.dimension.Limit == nil || entry.dimension.Used == nil ||
				entry.dimension.Remaining == nil || entry.dimension.Ratio == nil {
				codes = append(codes, CodeSchemaDrift)
				continue
			}
			if float64Value(entry.dimension.Remaining) < 0 {
				codes = append(codes, CodeNegativeRemaining)
			}
			if float64Value(entry.dimension.Ratio) < 0 || float64Value(entry.dimension.Ratio) > 1.2 {
				codes = append(codes, CodeRatioOutOfRange)
			}
		}
	}
	if projection.Digest != "" && projection.Digest != BudgetProjectionDigest(projection) {
		codes = append(codes, CodeDigestMismatch)
	}
	return dedupeStable(codes)
}

// ClassifyBudgetFactsGaps returns the absence and anomaly codes implied by a
// facts snapshot. These are the gaps a fixture may declare.
func ClassifyBudgetFactsGaps(facts BudgetFacts) []string {
	return ClassifyBudgetProjection(DeriveBudgetProjection(facts))
}

// ValidateBudgetProjectionCase compares a derived projection against the fixture
// expectation and returns the deterministic drift codes.
func ValidateBudgetProjectionCase(facts BudgetFacts, expected BudgetProjection) []string {
	actual := DeriveBudgetProjection(facts)
	codes := make([]string, 0, 4)
	codes = append(codes, ClassifyBudgetProjection(expected)...)
	if expected.Pressure != actual.Pressure {
		codes = append(codes, CodePressureLevelMismatch)
	}
	if !sameNotes(expected.Notes, actual.Notes) {
		codes = append(codes, CodeSchemaDrift)
	}
	if expected.Digest != "" && expected.Digest != actual.Digest {
		codes = append(codes, CodeDigestMismatch)
	}
	if !sameDimensions(expected.Remaining.Iteration, actual.Remaining.Iteration) ||
		!sameDimensions(expected.Remaining.ToolCall, actual.Remaining.ToolCall) ||
		!sameDimensions(expected.Remaining.TimeMS, actual.Remaining.TimeMS) ||
		!sameDimensions(expected.Remaining.CostHeadroom, actual.Remaining.CostHeadroom) {
		codes = append(codes, CodeSchemaDrift)
	}
	return dedupeStable(codes)
}

func isKnownPressureLevel(level BudgetPressureLevel) bool {
	for _, candidate := range BudgetPressureLevels {
		if candidate == level {
			return true
		}
	}
	return false
}

func sameDimensions(left BudgetDimension, right BudgetDimension) bool {
	return left.Available == right.Available &&
		formatOptionalFloat(left.Limit) == formatOptionalFloat(right.Limit) &&
		formatOptionalFloat(left.Used) == formatOptionalFloat(right.Used) &&
		formatOptionalFloat(left.Remaining) == formatOptionalFloat(right.Remaining) &&
		formatOptionalFloat(left.Ratio) == formatOptionalFloat(right.Ratio)
}

func sameNotes(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func dedupeStable(codes []string) []string {
	seen := map[string]struct{}{}
	output := make([]string, 0, len(codes))
	for _, code := range codes {
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		output = append(output, code)
	}
	sort.Strings(output)
	return output
}
