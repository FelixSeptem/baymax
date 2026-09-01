package diagnosticsreplay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const EventStreamTerminalRecoveryFixtureV1 = "runtime_event_stream_terminal_recovery.v1"

const (
	ReasonCodeStreamRecoverySchema               = "stream_recovery_schema_mismatch"
	ReasonCodeStreamRecoveryCursorHandoffDrift   = "stream_recovery_cursor_handoff_drift"
	ReasonCodeStreamRecoveryDedupeDrift          = "stream_recovery_dedupe_drift"
	ReasonCodeStreamRecoveryTerminalDrift        = "stream_recovery_terminal_drift"
	ReasonCodeStreamRecoveryRetainedDrift        = "stream_recovery_retained_facts_drift"
	ReasonCodeStreamRecoveryRunStreamParity      = "stream_recovery_run_stream_parity_drift"
	ReasonCodeStreamRecoveryLibraryFirstBoundary = "stream_recovery_library_first_boundary"
)

type EventStreamTerminalRecoveryFixture struct {
	Version   string                                   `json:"version"`
	Ownership string                                   `json:"ownership,omitempty"`
	Cases     []EventStreamTerminalRecoveryFixtureCase `json:"cases"`
}

type EventStreamTerminalRecoveryFixtureCase struct {
	Name     string                            `json:"name"`
	Run      EventStreamTerminalRecoveryReplay `json:"run"`
	Stream   EventStreamTerminalRecoveryReplay `json:"stream"`
	Expected EventStreamTerminalRecoveryReplay `json:"expected"`
}

type EventStreamTerminalRecoveryReplay struct {
	ObserverState            string `json:"observer_state"`
	RecoveryReason           string `json:"recovery_reason,omitempty"`
	TerminalState            string `json:"terminal_state,omitempty"`
	TerminalFailureFamily    string `json:"terminal_failure_family,omitempty"`
	RetainedEventTotal       int    `json:"retained_event_total,omitempty"`
	RetainedToolCallTotal    int    `json:"retained_tool_call_total,omitempty"`
	DeduplicatedEventTotal   int    `json:"deduplicated_event_total,omitempty"`
	TerminalConflictRecorded bool   `json:"terminal_conflict_recorded,omitempty"`
}

func EvaluateEventStreamTerminalRecoveryFixtureJSON(raw []byte) ([]EventStreamTerminalRecoveryFixtureCase, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var fixture EventStreamTerminalRecoveryFixture
	if err := dec.Decode(&fixture); err != nil {
		return nil, &ValidationError{Code: ReasonCodeStreamRecoverySchema, Message: err.Error()}
	}
	if strings.TrimSpace(fixture.Version) != EventStreamTerminalRecoveryFixtureV1 || len(fixture.Cases) == 0 {
		return nil, &ValidationError{Code: ReasonCodeStreamRecoverySchema, Message: "unsupported version or empty cases"}
	}
	if ownership := strings.TrimSpace(fixture.Ownership); ownership != "" && ownership != "source_owned" {
		return nil, &ValidationError{Code: ReasonCodeStreamRecoveryLibraryFirstBoundary, Message: "recovery fixture ownership must be source_owned"}
	}
	cases := append([]EventStreamTerminalRecoveryFixtureCase(nil), fixture.Cases...)
	sort.Slice(cases, func(i, j int) bool { return cases[i].Name < cases[j].Name })
	seen := map[string]struct{}{}
	for _, tc := range cases {
		if strings.TrimSpace(tc.Name) == "" || tc.Run.ObserverState == "" || tc.Stream.ObserverState == "" || tc.Expected.ObserverState == "" {
			return nil, &ValidationError{Code: ReasonCodeStreamRecoverySchema, Message: "case name and observer_state are required"}
		}
		for _, replay := range []EventStreamTerminalRecoveryReplay{tc.Run, tc.Stream, tc.Expected} {
			if !isValidEventStreamTerminalRecoveryObserverState(replay.ObserverState) {
				return nil, &ValidationError{Code: ReasonCodeStreamRecoverySchema, Message: fmt.Sprintf("case %q has unsupported observer_state %q", tc.Name, replay.ObserverState)}
			}
			if replay.RetainedEventTotal < 0 || replay.RetainedToolCallTotal < 0 || replay.DeduplicatedEventTotal < 0 {
				return nil, &ValidationError{Code: ReasonCodeStreamRecoverySchema, Message: fmt.Sprintf("case %q has negative retained or deduplicated total", tc.Name)}
			}
		}
		if _, ok := seen[tc.Name]; ok {
			return nil, &ValidationError{Code: ReasonCodeStreamRecoverySchema, Message: fmt.Sprintf("duplicate case %q", tc.Name)}
		}
		seen[tc.Name] = struct{}{}
		if !eventStreamRecoveryReplayEqual(tc.Run, tc.Stream) {
			return nil, &ValidationError{Code: ReasonCodeStreamRecoveryRunStreamParity, Message: fmt.Sprintf("case %q run/stream recovery parity drift", tc.Name)}
		}
		if !eventStreamRecoveryReplayEqual(tc.Expected, tc.Run) {
			return nil, &ValidationError{Code: eventStreamTerminalRecoveryDriftCode(tc.Expected, tc.Run), Message: fmt.Sprintf("case %q recovery drift", tc.Name)}
		}
	}
	return cases, nil
}

func eventStreamTerminalRecoveryDriftCode(expected, actual EventStreamTerminalRecoveryReplay) string {
	if expected.ObserverState != actual.ObserverState || expected.RecoveryReason != actual.RecoveryReason {
		return ReasonCodeStreamRecoveryCursorHandoffDrift
	}
	if expected.DeduplicatedEventTotal != actual.DeduplicatedEventTotal {
		return ReasonCodeStreamRecoveryDedupeDrift
	}
	if expected.RetainedEventTotal != actual.RetainedEventTotal || expected.RetainedToolCallTotal != actual.RetainedToolCallTotal {
		return ReasonCodeStreamRecoveryRetainedDrift
	}
	return ReasonCodeStreamRecoveryTerminalDrift
}

func isValidEventStreamTerminalRecoveryObserverState(state string) bool {
	switch state {
	case "catching_up", "live", "disconnected", "stopped", "terminal_available", "expired", "gap", "backpressure":
		return true
	default:
		return false
	}
}

func eventStreamRecoveryReplayEqual(a, b EventStreamTerminalRecoveryReplay) bool {
	return a == b
}
