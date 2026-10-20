package diagnosticsreplay

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestNativeContinuationReplayGoldenAndDrift(t *testing.T) {
	path := os.Getenv("BAYMAX_NATIVE_CONTINUATION_FIXTURE")
	if path == "" {
		path = "testdata/native_continuation.v1.json"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), raw...)
	got, err := ReplayNativeContinuationFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	again, err := ReplayNativeContinuationFixtureJSON(raw)
	if err != nil || !reflect.DeepEqual(got, again) || !reflect.DeepEqual(raw, before) {
		t.Fatal("replay was nondeterministic or mutated input", err)
	}
	if len(got) < 20 {
		t.Fatal("crash-window matrix incomplete")
	}
	for _, result := range got {
		if len(result.Digest) != 64 {
			t.Fatal("missing deterministic replay digest")
		}
	}
}

func TestNativeContinuationReplayDetectsSourceEvidenceDrift(t *testing.T) {
	base := NativeContinuationReplayCase{
		CaseID: "claimed", Form: "direct", PauseReason: "input_required", Window: "claimed_before_start",
		CheckpointID: "checkpoint", RunID: "run", SessionID: "session", SourceNamespace: "runner", Generation: 1,
		SourceState: "claimed", SourceBound: true, ClaimCommitted: true,
		Run:    NativeContinuationProjection{Outcome: "in_progress", RunID: "run", SessionID: "session"},
		Stream: NativeContinuationProjection{Outcome: "in_progress", RunID: "run", SessionID: "session"},
	}
	for _, tc := range []struct {
		reason string
		change func(*NativeContinuationReplayCase)
	}{
		{ReasonNativeContinuationTerminalEvidenceDrift, func(c *NativeContinuationReplayCase) { c.Run.Outcome = "completed"; c.Stream.Outcome = "completed" }},
		{ReasonNativeContinuationMissingBinding, func(c *NativeContinuationReplayCase) { c.SourceBound = false }},
		{ReasonNativeContinuationDuplicateExecution, func(c *NativeContinuationReplayCase) { c.ExecutionCount = 2 }},
		{ReasonNativeContinuationUnsupportedComposite, func(c *NativeContinuationReplayCase) { c.Form = "workflow"; c.ExecutionCount = 1 }},
		{ReasonNativeContinuationParityDrift, func(c *NativeContinuationReplayCase) { c.Stream.Outcome = "completed" }},
		{ReasonNativeContinuationIdentityDrift, func(c *NativeContinuationReplayCase) { c.Run.RunID = "new-run"; c.Stream.RunID = "new-run" }},
		{ReasonNativeContinuationSideEffect, func(c *NativeContinuationReplayCase) { c.ClaimCommitted = false; c.ExecutionCount = 1 }},
	} {
		t.Run(tc.reason, func(t *testing.T) {
			item := base
			tc.change(&item)
			item.ExpectedReason = tc.reason
			raw, err := json.Marshal(NativeContinuationFixture{Version: NativeContinuationReplayVersion, Cases: []NativeContinuationReplayCase{item}})
			if err != nil {
				t.Fatal(err)
			}
			got, err := ReplayNativeContinuationFixtureJSON(raw)
			if err != nil || len(got) != 1 || got[0].Reason != tc.reason {
				t.Fatalf("drift not classified: %#v %v", got, err)
			}
			item.ExpectedReason = ""
			raw, _ = json.Marshal(NativeContinuationFixture{Version: NativeContinuationReplayVersion, Cases: []NativeContinuationReplayCase{item}})
			if _, err = ReplayNativeContinuationFixtureJSON(raw); err == nil {
				t.Fatal("invented success was accepted")
			}
		})
	}
}

func TestNativeContinuationReplayIsReferenceOnlyAndStrict(t *testing.T) {
	for _, raw := range []string{
		`{"version":"native_continuation.v1","cases":[],"callback":"execute"}`,
		`{"version":"native_continuation.v1","cases":[{"case_id":"payload","material":{"credential":"secret"}}]}`,
		`{"version":"future","cases":[]}`,
		`{"version":"native_continuation.v1","cases":[]} {}`,
	} {
		_, err := ReplayNativeContinuationFixtureJSON([]byte(raw))
		var validation *ValidationError
		if !errors.As(err, &validation) || validation.Code != ReasonNativeContinuationReplayDrift {
			t.Fatal("unsafe schema accepted", err)
		}
	}
}
