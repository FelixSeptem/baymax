package diagnosticsreplay

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const ToolLifecycleReplayVersion = "tool_lifecycle_failure_isolation.v1"

type ToolLifecycleReplayStage struct {
	Stage      string `json:"stage"`
	Outcome    string `json:"outcome"`
	ReasonCode string `json:"reason_code,omitempty"`
	Started    bool   `json:"started,omitempty"`
}

type ToolLifecycleReplayCall struct {
	CallID           string                     `json:"call_id"`
	ToolName         string                     `json:"tool_name"`
	InputIndex       int                        `json:"input_index"`
	FailureOrigin    string                     `json:"failure_origin,omitempty"`
	ExecutionStarted bool                       `json:"execution_started,omitempty"`
	Finalized        bool                       `json:"finalized,omitempty"`
	AttemptCount     int                        `json:"attempt_count,omitempty"`
	FinalizeCount    int                        `json:"finalize_count,omitempty"`
	Owner            string                     `json:"owner,omitempty"`
	Stages           []ToolLifecycleReplayStage `json:"stages"`
}

type ToolLifecycleReplayOutput struct {
	Version string                    `json:"version"`
	Calls   []ToolLifecycleReplayCall `json:"calls"`
}

func ParseToolLifecycleReplayJSON(raw []byte) (ToolLifecycleReplayOutput, error) {
	var output ToolLifecycleReplayOutput
	if err := json.Unmarshal(raw, &output); err != nil {
		return ToolLifecycleReplayOutput{}, &ValidationError{Code: ReasonCodeInvalidJSON, Message: err.Error()}
	}
	if strings.TrimSpace(output.Version) != ToolLifecycleReplayVersion {
		return ToolLifecycleReplayOutput{}, &ValidationError{Code: "unsupported_version", Message: fmt.Sprintf("version %q", output.Version)}
	}
	if len(output.Calls) == 0 {
		return ToolLifecycleReplayOutput{}, &ValidationError{Code: ReasonCodeMissingRequiredField, Message: "calls must not be empty"}
	}
	seenCallIDs := make(map[string]struct{}, len(output.Calls))
	for i := range output.Calls {
		call := &output.Calls[i]
		if strings.TrimSpace(call.CallID) == "" || strings.TrimSpace(call.ToolName) == "" {
			return ToolLifecycleReplayOutput{}, &ValidationError{Code: ReasonCodeMissingRequiredField, Message: fmt.Sprintf("calls[%d] correlation is required", i)}
		}
		if _, exists := seenCallIDs[call.CallID]; exists {
			return ToolLifecycleReplayOutput{}, &ValidationError{Code: "duplicate_call_id", Message: fmt.Sprintf("calls[%d].call_id = %q", i, call.CallID)}
		}
		if owner := strings.TrimSpace(call.Owner); owner != "" && owner != "source_owned" {
			return ToolLifecycleReplayOutput{}, &ValidationError{Code: "hosted_ownership", Message: fmt.Sprintf("calls[%d].owner = %q", i, owner)}
		}
		if !validToolLifecycleReplayOrigin(call.FailureOrigin) {
			return ToolLifecycleReplayOutput{}, &ValidationError{Code: "failure_origin_drift", Message: fmt.Sprintf("calls[%d].failure_origin = %q", i, call.FailureOrigin)}
		}
		if call.FinalizeCount > 1 {
			return ToolLifecycleReplayOutput{}, &ValidationError{Code: "duplicate_finalize", Message: fmt.Sprintf("calls[%d].finalize_count = %d", i, call.FinalizeCount)}
		}
		seenCallIDs[call.CallID] = struct{}{}
		if len(call.Stages) != 5 {
			return ToolLifecycleReplayOutput{}, &ValidationError{Code: "lifecycle_stage_order_drift", Message: fmt.Sprintf("calls[%d] must contain five stages", i)}
		}
		want := []string{"prepare", "validate", "authorize", "execute", "finalize"}
		for j, stage := range call.Stages {
			if stage.Stage != want[j] {
				return ToolLifecycleReplayOutput{}, &ValidationError{Code: "lifecycle_stage_order_drift", Message: fmt.Sprintf("calls[%d].stages[%d] = %q", i, j, stage.Stage)}
			}
			if strings.TrimSpace(stage.Outcome) == "" {
				return ToolLifecycleReplayOutput{}, &ValidationError{Code: "invalid_stage_outcome", Message: fmt.Sprintf("calls[%d].stages[%d] outcome is required", i, j)}
			}
		}
		if call.Finalized && call.Stages[4].Outcome != "succeeded" {
			return ToolLifecycleReplayOutput{}, &ValidationError{Code: "duplicate_finalize", Message: fmt.Sprintf("calls[%d] finalized without successful finalize stage", i)}
		}
	}
	sort.SliceStable(output.Calls, func(i, j int) bool {
		if output.Calls[i].InputIndex != output.Calls[j].InputIndex {
			return output.Calls[i].InputIndex < output.Calls[j].InputIndex
		}
		return output.Calls[i].CallID < output.Calls[j].CallID
	})
	return output, nil
}

func validToolLifecycleReplayOrigin(origin string) bool {
	switch strings.TrimSpace(origin) {
	case "", "unknown", "lookup", "validation", "policy_denied", "sandbox_denied", "sandbox_failure", "middleware_short_circuit", "middleware_failure", "panic", "timeout", "canceled", "retry_exhausted", "execution":
		return true
	default:
		return false
	}
}
