package diagnosticsreplay

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestWorkflowContinuationReplayGoldenAndDrift(t *testing.T) {
	path := os.Getenv("BAYMAX_WORKFLOW_CONTINUATION_FIXTURE")
	if path == "" {
		path = "testdata/workflow_continuation.v1.json"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), raw...)
	got, err := ReplayWorkflowContinuationFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	again, err := ReplayWorkflowContinuationFixtureJSON(raw)
	if err != nil || !reflect.DeepEqual(got, again) || !reflect.DeepEqual(raw, before) || len(got) < 12 {
		t.Fatal("replay was nondeterministic, mutated input, or lacks lifecycle coverage", err)
	}
}

func TestWorkflowContinuationReplayClassifiesDrift(t *testing.T) {
	base := WorkflowContinuationReplayCase{CaseID: "workflow", Form: "workflow", PauseReason: "input_required", Window: "claimed_before_start", CheckpointID: "checkpoint", GraphID: "graph", GraphDigest: "digest", RunID: "run", SessionID: "session", SourceNamespace: "workflow", Generation: 1, SourceState: "claimed", ClaimCommitted: true, Run: WorkflowContinuationProjection{Outcome: "in_progress", RunID: "run", SessionID: "session"}, Stream: WorkflowContinuationProjection{Outcome: "in_progress", RunID: "run", SessionID: "session"}}
	for _, tc := range []struct {
		reason string
		change func(*WorkflowContinuationReplayCase)
	}{
		{ReasonWorkflowContinuationGraphDrift, func(c *WorkflowContinuationReplayCase) { c.GraphDigest = "" }},
		{ReasonWorkflowContinuationDuplicateExecution, func(c *WorkflowContinuationReplayCase) { c.ExecutionCount = 2 }},
		{ReasonWorkflowContinuationIdentityDrift, func(c *WorkflowContinuationReplayCase) { c.Stream.RunID = "other" }},
		{ReasonWorkflowContinuationClaimEvidence, func(c *WorkflowContinuationReplayCase) { c.ClaimCommitted = false }},
		{ReasonWorkflowContinuationUnsupported, func(c *WorkflowContinuationReplayCase) {
			c.Form = "team"
			c.ExecutionCount = 1
			c.Run = WorkflowContinuationProjection{Outcome: "unsupported_capability"}
			c.Stream = c.Run
		}},
	} {
		t.Run(tc.reason, func(t *testing.T) {
			item := base
			tc.change(&item)
			item.ExpectedReason = tc.reason
			raw, _ := json.Marshal(WorkflowContinuationReplayFixture{Version: WorkflowContinuationReplayVersion, Cases: []WorkflowContinuationReplayCase{item}})
			got, err := ReplayWorkflowContinuationFixtureJSON(raw)
			if err != nil || len(got) != 1 || got[0].Reason != tc.reason {
				t.Fatalf("drift not classified: %#v %v", got, err)
			}
			item.ExpectedReason = ""
			raw, _ = json.Marshal(WorkflowContinuationReplayFixture{Version: WorkflowContinuationReplayVersion, Cases: []WorkflowContinuationReplayCase{item}})
			if _, err := ReplayWorkflowContinuationFixtureJSON(raw); err == nil {
				t.Fatal("invented success accepted")
			}
		})
	}
}

func TestWorkflowContinuationReplayIsReferenceOnlyAndStrict(t *testing.T) {
	for _, raw := range []string{
		`{"version":"workflow_continuation.v1","cases":[],"execute":true}`,
		`{"version":"workflow_continuation.v1","cases":[{"case_id":"x","material":{"secret":"x"}}]}`,
	} {
		_, err := ReplayWorkflowContinuationFixtureJSON([]byte(raw))
		var validation *ValidationError
		if !errors.As(err, &validation) || validation.Code != ReasonWorkflowContinuationReplayDrift {
			t.Fatal("unsafe fixture accepted", err)
		}
	}
}
