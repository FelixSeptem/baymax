package diagnosticsreplay

import (
	"encoding/json"
	"reflect"

	"github.com/FelixSeptem/baymax/tool/schemaaudit"
)

const (
	ToolSchemaProjectionReadinessFixtureV1                     = schemaaudit.ReadinessVersion
	ReasonCodeToolSchemaProjectionReadinessSchemaDrift         = "tool_schema_projection_readiness.schema_drift"
	ReasonCodeToolSchemaProjectionReadinessSelectionDrift      = "tool_schema_projection_readiness.selection_drift"
	ReasonCodeToolSchemaProjectionReadinessDigestDrift         = "tool_schema_projection_readiness.digest_drift"
	ReasonCodeToolSchemaProjectionReadinessReplayNotIdempotent = "tool_schema_projection_readiness.replay_not_idempotent"
	ReasonCodeToolSchemaProjectionReadinessParityDrift         = "tool_schema_projection_readiness.run_stream_parity_drift"
)

type ToolSchemaProjectionReadinessFixture struct {
	Version string                                     `json:"version"`
	Cases   []ToolSchemaProjectionReadinessFixtureCase `json:"cases"`
}

type ToolSchemaProjectionReadinessFixtureCase struct {
	CaseID   string                                `json:"case_id"`
	Input    schemaaudit.ReadinessInput            `json:"input"`
	Expected ToolSchemaProjectionReadinessExpected `json:"expected,omitempty"`
}

type ToolSchemaProjectionReadinessExpected struct {
	Conclusion       string `json:"conclusion,omitempty"`
	PressureSignal   string `json:"pressure_signal,omitempty"`
	Opportunity      string `json:"opportunity,omitempty"`
	SelectionQuality string `json:"selection_quality,omitempty"`
}

type ToolSchemaProjectionReadinessReplayResult struct {
	Version string                                    `json:"version"`
	Cases   []ToolSchemaProjectionReadinessReplayCase `json:"cases"`
}

type ToolSchemaProjectionReadinessReplayCase struct {
	CaseID           string `json:"case_id"`
	Conclusion       string `json:"conclusion"`
	PressureSignal   string `json:"pressure_signal"`
	Opportunity      string `json:"opportunity"`
	SelectionQuality string `json:"selection_quality"`
	Digest           string `json:"digest"`
	ReplayDigest     string `json:"replay_digest"`
	Idempotent       bool   `json:"idempotent"`
}

func EvaluateToolSchemaProjectionReadiness(input schemaaudit.ReadinessInput) (schemaaudit.ReadinessResult, error) {
	result, err := schemaaudit.EvaluateReadiness(input)
	if err != nil {
		return schemaaudit.ReadinessResult{}, &ValidationError{Code: schemaaudit.CodeOf(err), Message: err.Error()}
	}
	return result, nil
}

func ReplayToolSchemaProjectionReadinessFixtureJSON(raw []byte) (ToolSchemaProjectionReadinessReplayResult, error) {
	var fixture ToolSchemaProjectionReadinessFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		return ToolSchemaProjectionReadinessReplayResult{}, &ValidationError{Code: ReasonCodeInvalidJSON, Message: err.Error()}
	}
	if fixture.Version != ToolSchemaProjectionReadinessFixtureV1 || len(fixture.Cases) == 0 {
		return ToolSchemaProjectionReadinessReplayResult{}, &ValidationError{Code: ReasonCodeToolSchemaProjectionReadinessSchemaDrift, Message: "unsupported or empty readiness fixture"}
	}
	output := ToolSchemaProjectionReadinessReplayResult{Version: fixture.Version, Cases: make([]ToolSchemaProjectionReadinessReplayCase, 0, len(fixture.Cases))}
	for _, item := range fixture.Cases {
		first, err := replayToolSchemaProjectionReadinessCase(item)
		if err != nil {
			return ToolSchemaProjectionReadinessReplayResult{}, err
		}
		second, err := replayToolSchemaProjectionReadinessCase(item)
		if err != nil {
			return ToolSchemaProjectionReadinessReplayResult{}, err
		}
		if first.Digest != second.Digest {
			return ToolSchemaProjectionReadinessReplayResult{}, &ValidationError{Code: ReasonCodeToolSchemaProjectionReadinessReplayNotIdempotent, Message: item.CaseID}
		}
		first.ReplayDigest = second.Digest
		first.Idempotent = true
		output.Cases = append(output.Cases, first)
	}
	return output, nil
}

func replayToolSchemaProjectionReadinessCase(item ToolSchemaProjectionReadinessFixtureCase) (ToolSchemaProjectionReadinessReplayCase, error) {
	if item.CaseID == "" {
		return ToolSchemaProjectionReadinessReplayCase{}, &ValidationError{Code: ReasonCodeToolSchemaProjectionReadinessSchemaDrift, Message: "case_id is required"}
	}
	result, err := EvaluateToolSchemaProjectionReadiness(item.Input)
	if err != nil {
		return ToolSchemaProjectionReadinessReplayCase{}, err
	}
	if expected := item.Expected; expected.Conclusion != "" && result.Conclusion != expected.Conclusion {
		return ToolSchemaProjectionReadinessReplayCase{}, &ValidationError{Code: ReasonCodeToolSchemaProjectionReadinessSelectionDrift, Message: "conclusion mismatch"}
	}
	if expected := item.Expected; expected.PressureSignal != "" && result.PressureSignal != expected.PressureSignal {
		return ToolSchemaProjectionReadinessReplayCase{}, &ValidationError{Code: ReasonCodeToolSchemaProjectionReadinessSelectionDrift, Message: "pressure signal mismatch"}
	}
	if expected := item.Expected; expected.Opportunity != "" && result.Opportunity != expected.Opportunity {
		return ToolSchemaProjectionReadinessReplayCase{}, &ValidationError{Code: ReasonCodeToolSchemaProjectionReadinessSelectionDrift, Message: "opportunity mismatch"}
	}
	if expected := item.Expected; expected.SelectionQuality != "" && result.SelectionQuality != expected.SelectionQuality {
		return ToolSchemaProjectionReadinessReplayCase{}, &ValidationError{Code: ReasonCodeToolSchemaProjectionReadinessSelectionDrift, Message: "selection quality mismatch"}
	}
	return ToolSchemaProjectionReadinessReplayCase{CaseID: item.CaseID, Conclusion: result.Conclusion, PressureSignal: result.PressureSignal, Opportunity: result.Opportunity, SelectionQuality: result.SelectionQuality, Digest: result.Digest}, nil
}

func CompareToolSchemaProjectionReadinessParity(run, stream schemaaudit.ReadinessResult) error {
	if !reflect.DeepEqual(run, stream) {
		return &ValidationError{Code: ReasonCodeToolSchemaProjectionReadinessParityDrift, Message: "readiness semantics differ between run and stream"}
	}
	return nil
}
