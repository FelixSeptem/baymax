package diagnosticsreplay

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestEventStreamTerminalRecoveryFixtureFile(t *testing.T) {
	raw, err := os.ReadFile("testdata/event_stream_terminal_recovery.json")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if _, err := EvaluateEventStreamTerminalRecoveryFixtureJSON(raw); err != nil {
		t.Fatalf("EvaluateEventStreamTerminalRecoveryFixtureJSON() error = %v", err)
	}
}

func TestEventStreamTerminalRecoveryFixtureCoversCanonicalRecoveryScenarios(t *testing.T) {
	raw, err := os.ReadFile("testdata/event_stream_terminal_recovery.json")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	cases, err := EvaluateEventStreamTerminalRecoveryFixtureJSON(raw)
	if err != nil {
		t.Fatalf("EvaluateEventStreamTerminalRecoveryFixtureJSON() error = %v", err)
	}
	want := map[string]bool{
		"success_terminal_snapshot":              false,
		"observer_disconnect":                    false,
		"observer_stop":                          false,
		"catch_up_live_handoff":                  false,
		"overlap_deduplicated":                   false,
		"sequence_gap":                           false,
		"retention_expired":                      false,
		"backpressure":                           false,
		"canceled_terminal":                      false,
		"timed_out_terminal":                     false,
		"provider_failure_after_partial_output":  false,
		"late_terminal_conflict_preserves_first": false,
	}
	for _, tc := range cases {
		if _, ok := want[tc.Name]; ok {
			want[tc.Name] = true
		}
	}
	for name, covered := range want {
		if !covered {
			t.Errorf("canonical fixture missing scenario %q", name)
		}
	}
}

func TestEvaluateEventStreamTerminalRecoveryFixture(t *testing.T) {
	raw := []byte(`{"version":"runtime_event_stream_terminal_recovery.v1","cases":[{"name":"terminal","run":{"observer_state":"terminal_available","terminal_state":"completed","terminal_failure_family":"none","retained_event_total":2,"retained_tool_call_total":1,"terminal_conflict_recorded":false},"stream":{"observer_state":"terminal_available","terminal_state":"completed","terminal_failure_family":"none","retained_event_total":2,"retained_tool_call_total":1,"terminal_conflict_recorded":false},"expected":{"observer_state":"terminal_available","terminal_state":"completed","terminal_failure_family":"none","retained_event_total":2,"retained_tool_call_total":1,"terminal_conflict_recorded":false}}]}`)
	if _, err := EvaluateEventStreamTerminalRecoveryFixtureJSON(raw); err != nil {
		t.Fatalf("EvaluateEventStreamTerminalRecoveryFixtureJSON() error = %v", err)
	}
}

func TestEvaluateEventStreamTerminalRecoveryFixtureClassifiesParityDrift(t *testing.T) {
	raw := []byte(`{"version":"runtime_event_stream_terminal_recovery.v1","cases":[{"name":"parity","run":{"observer_state":"live"},"stream":{"observer_state":"gap"},"expected":{"observer_state":"live"}}]}`)
	_, err := EvaluateEventStreamTerminalRecoveryFixtureJSON(raw)
	if err == nil || !strings.Contains(err.Error(), "stream_recovery_run_stream_parity_drift") {
		t.Fatalf("error = %v, want parity drift", err)
	}
}

func TestEvaluateEventStreamTerminalRecoveryFixtureClassifiesRecoveryNegativeFixtures(t *testing.T) {
	tests := []struct {
		name string
		file string
		code string
	}{
		{name: "malformed", file: "event_stream_terminal_recovery_malformed.json", code: ReasonCodeStreamRecoverySchema},
		{name: "unsupported version", file: "event_stream_terminal_recovery_unsupported_version.json", code: ReasonCodeStreamRecoverySchema},
		{name: "duplicate overlap", file: "event_stream_terminal_recovery_duplicate_overlap_drift.json", code: ReasonCodeStreamRecoveryDedupeDrift},
		{name: "hosted ownership", file: "event_stream_terminal_recovery_hosted_ownership.json", code: ReasonCodeStreamRecoveryLibraryFirstBoundary},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := os.ReadFile("testdata/" + tc.file)
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			_, err = EvaluateEventStreamTerminalRecoveryFixtureJSON(raw)
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("error = %v, want %q", err, tc.code)
			}
		})
	}
}

func TestEventStreamTerminalRecoveryFixtureNegativeOutcomeScenariosRemainReplayable(t *testing.T) {
	for _, file := range []string{
		"event_stream_terminal_recovery_retention_expired.json",
		"event_stream_terminal_recovery_sequence_gap.json",
	} {
		t.Run(file, func(t *testing.T) {
			raw, err := os.ReadFile("testdata/" + file)
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			if _, err := EvaluateEventStreamTerminalRecoveryFixtureJSON(raw); err != nil {
				t.Fatalf("EvaluateEventStreamTerminalRecoveryFixtureJSON() error = %v", err)
			}
		})
	}
}

func TestEvaluateEventStreamTerminalRecoveryFixtureClassifiesHandoffDrift(t *testing.T) {
	raw := []byte(`{"version":"runtime_event_stream_terminal_recovery.v1","cases":[{"name":"handoff","run":{"observer_state":"live","recovery_reason":"realtime.binding.live"},"stream":{"observer_state":"live","recovery_reason":"realtime.binding.live"},"expected":{"observer_state":"catching_up","recovery_reason":"realtime.binding.catching_up"}}]}`)
	_, err := EvaluateEventStreamTerminalRecoveryFixtureJSON(raw)
	if err == nil || !strings.Contains(err.Error(), ReasonCodeStreamRecoveryCursorHandoffDrift) {
		t.Fatalf("error = %v, want handoff drift", err)
	}
}

func TestEventStreamTerminalRecoveryMixedVersionReplayRemainsDeterministic(t *testing.T) {
	historicalRaw, err := os.ReadFile("testdata/terminal_outcomes.json")
	if err != nil {
		t.Fatalf("ReadFile(historical terminal fixture) error = %v", err)
	}
	historical, err := ParseMinimalReplayJSON(historicalRaw)
	if err != nil {
		t.Fatalf("ParseMinimalReplayJSON(historical terminal fixture) error = %v", err)
	}
	protocolRaw, err := os.ReadFile("testdata/success_input.json")
	if err != nil {
		t.Fatalf("ReadFile(historical arbitration fixture) error = %v", err)
	}
	if _, err := ParseMinimalReplayJSON(protocolRaw); err != nil {
		t.Fatalf("ParseMinimalReplayJSON(historical minimal fixture) error = %v", err)
	}
	recoveryRaw, err := os.ReadFile("testdata/event_stream_terminal_recovery.json")
	if err != nil {
		t.Fatalf("ReadFile(recovery fixture) error = %v", err)
	}
	first, err := EvaluateEventStreamTerminalRecoveryFixtureJSON(recoveryRaw)
	if err != nil {
		t.Fatalf("first recovery replay error = %v", err)
	}
	second, err := EvaluateEventStreamTerminalRecoveryFixtureJSON(recoveryRaw)
	if err != nil {
		t.Fatalf("second recovery replay error = %v", err)
	}
	if len(historical.Events) == 0 || !reflect.DeepEqual(first, second) {
		t.Fatalf("mixed-version replay is not deterministic: historical=%#v first=%#v second=%#v", historical, first, second)
	}
}
