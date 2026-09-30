package scenariosimulation

import (
	"encoding/json"
	"fmt"
	"sort"
)

type ReplayCase struct {
	ID       string    `json:"id"`
	Scenario Scenario  `json:"scenario"`
	Expected RunResult `json:"expected"`
	Observed RunResult `json:"observed"`
}

type ReplayResult struct {
	ID             string `json:"id"`
	ExpectedDigest string `json:"expected_digest"`
	ObservedDigest string `json:"observed_digest"`
	Classification string `json:"classification,omitempty"`
	Passed         bool   `json:"passed"`
}

func normalizeRunResult(in RunResult) (RunResult, string, error) {
	in.Version = clean(in.Version, MaxIdentifierLength)
	if in.Version == "" {
		in.Version = RunResultVersionV1
	}
	in.ScenarioID = clean(in.ScenarioID, MaxIdentifierLength)
	if in.Version != RunResultVersionV1 || in.ScenarioID == "" || len(in.Events) > MaxScenarioEvents || len(in.EvidenceReferences) > MaxEvidenceReferences {
		return RunResult{}, "", fmt.Errorf("%s", ReasonScenarioSchemaDrift)
	}
	in.Unknown = nil
	for i := range in.Events {
		e := &in.Events[i]
		e.ID = clean(e.ID, MaxIdentifierLength)
		e.Kind = clean(e.Kind, MaxIdentifierLength)
		e.Owner = clean(e.Owner, MaxIdentifierLength)
		e.CausationID = clean(e.CausationID, MaxIdentifierLength)
		if e.ID == "" || e.Kind == "" || e.Owner == "" || e.Sequence < 0 {
			return RunResult{}, "", fmt.Errorf("%s", ReasonScenarioSchemaDrift)
		}
	}
	sort.SliceStable(in.Events, func(i, j int) bool {
		if in.Events[i].Sequence == in.Events[j].Sequence {
			return in.Events[i].ID < in.Events[j].ID
		}
		return in.Events[i].Sequence < in.Events[j].Sequence
	})
	for i := range in.EvidenceReferences {
		if err := normalizeReference(&in.EvidenceReferences[i]); err != nil {
			return RunResult{}, "", err
		}
	}
	sort.SliceStable(in.EvidenceReferences, func(i, j int) bool {
		if in.EvidenceReferences[i].Owner == in.EvidenceReferences[j].Owner {
			return in.EvidenceReferences[i].ID < in.EvidenceReferences[j].ID
		}
		return in.EvidenceReferences[i].Owner < in.EvidenceReferences[j].Owner
	})
	if in.OutcomeReference != nil {
		normalizeOutcome(in.OutcomeReference)
	}
	b, err := json.Marshal(in)
	if err != nil || len(b) > MaxSerializedBytes {
		return RunResult{}, "", fmt.Errorf("%s", ReasonScenarioSchemaDrift)
	}
	d, err := digestBytes(b)
	return in, d, err
}

func Replay(in ReplayCase) (ReplayResult, error) {
	in.ID = clean(in.ID, MaxIdentifierLength)
	if in.ID == "" {
		return ReplayResult{}, fmt.Errorf("%s", ReasonScenarioSchemaDrift)
	}
	_, _, err := NormalizeScenario(in.Scenario)
	if err != nil {
		return ReplayResult{}, err
	}
	expected, expectedDigest, err := normalizeRunResult(in.Expected)
	if err != nil {
		return ReplayResult{}, err
	}
	observed, observedDigest, err := normalizeRunResult(in.Observed)
	if err != nil {
		return ReplayResult{}, err
	}
	result := ReplayResult{ID: in.ID, ExpectedDigest: expectedDigest, ObservedDigest: observedDigest, Passed: expectedDigest == observedDigest}
	if result.Passed {
		return result, nil
	}
	result.Classification = classifyResultDrift(expected, observed)
	return result, fmt.Errorf("%s", result.Classification)
}

func classifyResultDrift(expected, observed RunResult) string {
	if len(expected.Events) != len(observed.Events) {
		return ReasonCompletionDrift
	}
	for i := range expected.Events {
		e, o := expected.Events[i], observed.Events[i]
		if e.ID != o.ID || e.Kind != o.Kind || e.Owner != o.Owner || e.Sequence != o.Sequence {
			return ReasonCompletionDrift
		}
		if e.CausationID != o.CausationID {
			return ReasonCausationDrift
		}
	}
	if expected.Evidence.Status != observed.Evidence.Status || expected.Evidence.Reason != observed.Evidence.Reason || !sameEvidence(expected.EvidenceReferences, observed.EvidenceReferences) {
		return ReasonEvidenceIncomplete
	}
	if expected.Execution != observed.Execution || !sameCompletion(expected.Completion, observed.Completion) {
		return ReasonCompletionDrift
	}
	if expected.Outcome != observed.Outcome || !sameOutcome(expected.OutcomeReference, observed.OutcomeReference) {
		return ReasonOutcomeDrift
	}
	if expected.Admission != observed.Admission {
		return ReasonAdmissionDrift
	}
	return ReasonScenarioSchemaDrift
}

func sameEvidence(a, b []EvidenceReference) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}
func sameOutcome(a, b *OutcomeReference) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}
func sameCompletion(a, b *CompletionReference) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

// SemanticParity compares verdict axes and source-owned terminal/completion
// events. Stream-only incremental events are intentionally ignored.
func SemanticParity(run, stream RunResult) bool {
	if run.Execution != stream.Execution || run.Evidence != stream.Evidence || run.Outcome != stream.Outcome || run.Admission != stream.Admission || !sameOutcome(run.OutcomeReference, stream.OutcomeReference) || !sameCompletion(run.Completion, stream.Completion) || !sameEvidence(run.EvidenceReferences, stream.EvidenceReferences) {
		return false
	}
	canonical := func(events []ObservedEvent) []ObservedEvent {
		out := make([]ObservedEvent, 0, len(events))
		for _, event := range events {
			if event.Kind == "stream_chunk" || event.Kind == "heartbeat" {
				continue
			}
			event.Sequence = len(out) + 1
			out = append(out, event)
		}
		return out
	}
	return sameObservedEvents(canonical(run.Events), canonical(stream.Events))
}

func sameObservedEvents(a, b []ObservedEvent) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

func ReplayBatch(cases []ReplayCase) ([]ReplayResult, error) {
	seen := make(map[string]string, len(cases))
	results := make([]ReplayResult, 0, len(cases))
	for _, c := range cases {
		identityDigest, err := canonicalCaseDigest(c)
		if err != nil {
			return nil, err
		}
		if prior, ok := seen[c.ID]; ok {
			if prior != identityDigest {
				return nil, fmt.Errorf("%s", ReasonDuplicateConflict)
			}
			continue
		}
		seen[c.ID] = identityDigest
		result, replayErr := Replay(c)
		results = append(results, result)
		if replayErr != nil {
			return results, replayErr
		}
	}
	return results, nil
}

func canonicalCaseDigest(c ReplayCase) (string, error) {
	scenario, _, err := NormalizeScenario(c.Scenario)
	if err != nil {
		return "", err
	}
	expected, _, err := normalizeRunResult(c.Expected)
	if err != nil {
		return "", err
	}
	observed, _, err := normalizeRunResult(c.Observed)
	if err != nil {
		return "", err
	}
	return Digest(struct {
		ID       string    `json:"id"`
		Scenario Scenario  `json:"scenario"`
		Expected RunResult `json:"expected"`
		Observed RunResult `json:"observed"`
	}{ID: c.ID, Scenario: scenario, Expected: expected, Observed: observed})
}
