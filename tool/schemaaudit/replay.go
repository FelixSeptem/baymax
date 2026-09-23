package schemaaudit

import (
	"encoding/json"
	"reflect"
)

// ParseFixtureJSON decodes a bounded audit fixture. Unknown JSON fields are
// ignored for additive compatibility; raw payloads are never copied into the
// normalized result.
func ParseFixtureJSON(raw []byte) (Input, error) {
	var in Input
	if err := json.Unmarshal(raw, &in); err != nil {
		return Input{}, &Error{Code: CodeInvalidSchemaFacts, Message: err.Error()}
	}
	if in.Version == "" {
		in.Version = AuditVersion
	}
	return in, nil
}

// ReplayJSON evaluates a fixture twice and proves that its normalized digest
// is stable without model, tool, provider, filesystem, or network access.
func ReplayJSON(raw []byte) (Result, error) {
	in, err := ParseFixtureJSON(raw)
	if err != nil {
		return Result{}, err
	}
	first, err := Audit(in)
	if err != nil {
		return Result{}, err
	}
	second, err := Audit(in)
	if err != nil {
		return Result{}, err
	}
	if first.Digest != second.Digest {
		return Result{}, &Error{Code: CodeReplayNotIdempotent, Message: "audit digest changed during replay"}
	}
	return first, nil
}

// CompareParity checks semantic Run/Stream equivalence while ignoring the
// transport marker. It is intentionally read-only and does not inspect timing.
func CompareParity(run, stream Result) error {
	if !reflect.DeepEqual(run.Pressure, stream.Pressure) ||
		run.Baseline != stream.Baseline ||
		run.Conclusion != stream.Conclusion ||
		!reflect.DeepEqual(run.Reasons, stream.Reasons) ||
		!reflect.DeepEqual(run.Corpus, stream.Corpus) ||
		!reflect.DeepEqual(run.Cases, stream.Cases) ||
		len(run.Strategies) != len(stream.Strategies) {
		return &Error{Code: CodeRunStreamParityDrift, Message: "run and stream audit semantics differ"}
	}
	for i := range run.Strategies {
		if !reflect.DeepEqual(run.Strategies[i], stream.Strategies[i]) {
			return &Error{Code: CodeRunStreamParityDrift, Message: "strategy parity differs"}
		}
	}
	return nil
}
