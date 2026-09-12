package diagnosticsreplay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// HostTranscriptFixtureV1 identifies the versioned first-profile host contract
// transcript. It is intentionally separate from legacy Agent Runtime Protocol
// fixtures so older replay inputs remain byte-for-byte compatible.
const HostTranscriptFixtureV1 = "embedded_host_protocol.v1"

const (
	ReasonCodeHostTranscriptSchema      = "embedded_host_protocol_schema_mismatch"
	ReasonCodeHostTranscriptDrift       = "embedded_host_protocol_drift"
	ReasonCodeHostTranscriptParityDrift = "embedded_host_protocol_parity_drift"
)

type HostTranscriptFixture struct {
	Version string                      `json:"version"`
	Cases   []HostTranscriptFixtureCase `json:"cases"`
}

type HostTranscriptFixtureCase struct {
	Name            string                    `json:"name"`
	Run             HostTranscriptObservation `json:"run"`
	Stream          HostTranscriptObservation `json:"stream"`
	Expected        HostTranscriptObservation `json:"expected"`
	RunStreamParity string                    `json:"run_stream_parity"`
	Idempotency     ProtocolIdempotency       `json:"idempotency"`
}

// HostTranscriptObservation is a bounded normalized projection of a host
// command/response/event transcript. Payload bodies are deliberately omitted;
// replay verifies contract classifications and source-owned outcomes only.
type HostTranscriptObservation struct {
	Admission string   `json:"admission"`
	Reason    string   `json:"reason,omitempty"`
	Terminal  string   `json:"terminal,omitempty"`
	Events    []string `json:"events,omitempty"`
}

type HostTranscriptReplayOutput struct {
	Version string                         `json:"version"`
	Cases   []HostTranscriptNormalizedCase `json:"cases"`
}

type HostTranscriptNormalizedCase struct {
	Name            string                    `json:"name"`
	Canonical       HostTranscriptObservation `json:"canonical"`
	RunStreamParity string                    `json:"run_stream_parity"`
	Idempotency     ProtocolIdempotency       `json:"idempotency"`
}

func ParseHostTranscriptFixtureJSON(raw []byte) (HostTranscriptFixture, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var fixture HostTranscriptFixture
	if err := dec.Decode(&fixture); err != nil {
		return HostTranscriptFixture{}, &ValidationError{Code: ReasonCodeInvalidJSON, Message: err.Error()}
	}
	if strings.TrimSpace(fixture.Version) != HostTranscriptFixtureV1 {
		return HostTranscriptFixture{}, &ValidationError{Code: ReasonCodeHostTranscriptSchema, Message: fmt.Sprintf("unsupported fixture version %q", fixture.Version)}
	}
	if len(fixture.Cases) == 0 {
		return HostTranscriptFixture{}, &ValidationError{Code: ReasonCodeHostTranscriptSchema, Message: "cases must not be empty"}
	}
	seen := make(map[string]struct{}, len(fixture.Cases))
	for i := range fixture.Cases {
		c := &fixture.Cases[i]
		c.Name = strings.TrimSpace(c.Name)
		if c.Name == "" {
			return HostTranscriptFixture{}, &ValidationError{Code: ReasonCodeHostTranscriptSchema, Message: fmt.Sprintf("cases[%d].name is required", i)}
		}
		if _, ok := seen[c.Name]; ok {
			return HostTranscriptFixture{}, &ValidationError{Code: ReasonCodeHostTranscriptSchema, Message: fmt.Sprintf("duplicate case %q", c.Name)}
		}
		seen[c.Name] = struct{}{}
		for label, obs := range map[string]HostTranscriptObservation{"run": c.Run, "stream": c.Stream, "expected": c.Expected} {
			if err := validateHostTranscriptObservation(obs, c.Name, label); err != nil {
				return HostTranscriptFixture{}, err
			}
		}
		if c.RunStreamParity != "equivalent" {
			return HostTranscriptFixture{}, &ValidationError{Code: ReasonCodeHostTranscriptSchema, Message: fmt.Sprintf("case %q run_stream_parity must be equivalent", c.Name)}
		}
		if c.Idempotency.FirstLogicalIngestTotal <= 0 || c.Idempotency.FirstLogicalIngestTotal != c.Idempotency.ReplayLogicalIngestTotal {
			return HostTranscriptFixture{}, &ValidationError{Code: ReasonCodeHostTranscriptSchema, Message: fmt.Sprintf("case %q idempotency mismatch", c.Name)}
		}
	}
	fixture.Version = strings.TrimSpace(fixture.Version)
	return fixture, nil
}

func validateHostTranscriptObservation(obs HostTranscriptObservation, name, label string) error {
	if strings.TrimSpace(obs.Admission) == "" {
		return &ValidationError{Code: ReasonCodeHostTranscriptSchema, Message: fmt.Sprintf("case %q %s admission is required", name, label)}
	}
	if len(obs.Events) > 64 {
		return &ValidationError{Code: ReasonCodeHostTranscriptSchema, Message: fmt.Sprintf("case %q %s events exceed bound", name, label)}
	}
	return nil
}

func EvaluateHostTranscriptFixtureJSON(raw []byte) (HostTranscriptReplayOutput, error) {
	fixture, err := ParseHostTranscriptFixtureJSON(raw)
	if err != nil {
		return HostTranscriptReplayOutput{}, err
	}
	return EvaluateHostTranscriptFixture(fixture)
}

func EvaluateHostTranscriptFixture(fixture HostTranscriptFixture) (HostTranscriptReplayOutput, error) {
	if strings.TrimSpace(fixture.Version) != HostTranscriptFixtureV1 {
		return HostTranscriptReplayOutput{}, &ValidationError{Code: ReasonCodeHostTranscriptSchema, Message: fmt.Sprintf("unsupported fixture version %q", fixture.Version)}
	}
	cases := append([]HostTranscriptFixtureCase(nil), fixture.Cases...)
	sort.Slice(cases, func(i, j int) bool { return cases[i].Name < cases[j].Name })
	out := HostTranscriptReplayOutput{Version: fixture.Version, Cases: make([]HostTranscriptNormalizedCase, 0, len(cases))}
	for _, c := range cases {
		if c.RunStreamParity != "equivalent" || !reflect.DeepEqual(c.Run, c.Stream) {
			return HostTranscriptReplayOutput{}, &ValidationError{Code: ReasonCodeHostTranscriptParityDrift, Message: fmt.Sprintf("case %q run/stream parity drift", c.Name)}
		}
		if !reflect.DeepEqual(c.Run, c.Expected) {
			return HostTranscriptReplayOutput{}, &ValidationError{Code: ReasonCodeHostTranscriptDrift, Message: fmt.Sprintf("case %q run transcript drift", c.Name)}
		}
		out.Cases = append(out.Cases, HostTranscriptNormalizedCase{Name: c.Name, Canonical: c.Expected, RunStreamParity: c.RunStreamParity, Idempotency: c.Idempotency})
	}
	return out, nil
}
