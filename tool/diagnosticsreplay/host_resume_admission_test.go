package diagnosticsreplay

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestHostResumeReplayProvenance(t *testing.T) {
	raw, err := os.ReadFile("testdata/host_resume_admission.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]any
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	first := fixture["cases"].([]any)[0].(map[string]any)
	delete(first, "parity_group")
	fixture["cases"] = []any{first}
	for _, field := range []string{"matching", "admission_reference", "policy_revision", "source_scope_digest", "partial", "empty", "effectful"} {
		t.Run(field, func(t *testing.T) {
			original := map[string]any{"admission_reference": "admission", "policy_revision": "policy", "source_scope_digest": "scope"}
			current := map[string]any{"admission_reference": "admission", "policy_revision": "policy", "source_scope_digest": "scope"}
			first["original_provenance"], first["current_provenance"] = original, current
			first["status"], first["expected_reason"] = "rejected", "host_resume.invalid_binding"
			first["resolutions"], first["terminal_events"], first["provider_calls"], first["effects"] = 0, 0, 0, 0
			first["iteration_after"], first["terminal_state"] = 1, ""
			switch field {
			case "matching":
				first["status"], first["expected_reason"] = "accepted", ""
				first["resolutions"], first["terminal_events"], first["provider_calls"], first["effects"] = 1, 1, 1, 1
				first["iteration_after"], first["terminal_state"] = 2, "completed"
			case "admission_reference":
				current[field] = "changed"
				first["expected_reason"] = "host_resume.binding_mismatch"
			case "policy_revision":
				current[field] = "changed"
				first["expected_reason"] = "host_resume.authority_denied"
			case "source_scope_digest", "effectful":
				current["source_scope_digest"] = "changed"
				first["expected_reason"] = "host_resume.scope_drift"
			case "partial":
				first["current_provenance"] = nil
			case "empty":
				current["policy_revision"] = ""
			}
			if field == "effectful" {
				first["effects"] = 1
			}
			body, _ := json.Marshal(fixture)
			_, err := ReplayHostResumeAdmissionFixtureJSON(body)
			if (err != nil) != (field == "effectful") {
				t.Fatalf("replay err=%v", err)
			}
		})
	}
}

func TestHostResumeReplayProvenanceCombinedFailures(t *testing.T) {
	raw, err := os.ReadFile("testdata/host_resume_admission.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture HostResumeAdmissionFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"partial-authority", "empty-binding", "oversized-authority", "identity-authority", "scope-revision", "binding-sentinel-collision"} {
		t.Run(name, func(t *testing.T) {
			x := fixture.Cases[0]
			x.ParityGroup = ""
			x.Status, x.ExpectedReason = "rejected", "host_resume.invalid_binding"
			x.Resolutions, x.TerminalEvents, x.ProviderCalls, x.Effects = 0, 0, 0, 0
			x.IterationAfter, x.TerminalState = x.IterationBefore, ""
			original := &types.HostAdmissionProvenance{AdmissionReference: "admission", PolicyRevision: "policy", SourceScopeDigest: "scope"}
			current := *original
			x.OriginalProvenance, x.CurrentProvenance = original, &current
			switch name {
			case "partial-authority":
				x.AuthorityValid, x.CurrentProvenance = false, nil
			case "empty-binding":
				x.BindingMatches = false
				original.PolicyRevision = ""
			case "oversized-authority":
				x.AuthorityValid = false
				original.AdmissionReference = strings.Repeat("x", types.HostResumeMaxReferenceBytes+1)
			case "identity-authority":
				x.AuthorityValid = false
				current.AdmissionReference = "changed"
				x.ExpectedReason = "host_resume.binding_mismatch"
			case "scope-revision":
				x.RevisionMatches = false
				current.SourceScopeDigest = "changed"
				x.ExpectedReason = "host_resume.scope_drift"
			case "binding-sentinel-collision":
				x.BindingMatches = false
				x.CheckpointID = "changed"
				x.ExpectedReason = "host_resume.binding_mismatch"
			}
			body, _ := json.Marshal(HostResumeAdmissionFixture{Version: fixture.Version, Cases: []HostResumeAdmissionReplayCase{x}})
			if _, err := ReplayHostResumeAdmissionFixtureJSON(body); err != nil {
				t.Fatalf("live-validator reason %s rejected: %v", x.ExpectedReason, err)
			}
		})
	}
}

func TestHostResumeReplayGoldenAndDrift(t *testing.T) {
	path := os.Getenv("BAYMAX_HOST_RESUME_ADMISSION_FIXTURE")
	if path == "" {
		path = "testdata/host_resume_admission.golden.json"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	results, err := ReplayHostResumeAdmissionFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 17 {
		t.Fatalf("cases=%d", len(results))
	}
	var f HostResumeAdmissionFixture
	if err = json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"resolution", "effect", "continuation", "iteration", "terminal", "parity", "binding", "privacy", "version", "duplicate", "unknown"} {
		t.Run(name, func(t *testing.T) {
			var x HostResumeAdmissionFixture
			if err := json.Unmarshal(raw, &x); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "resolution":
				x.Cases[2].Resolutions = 1
			case "effect":
				x.Cases[2].Effects = 1
			case "continuation":
				x.Cases[2].ProviderCalls = 1
			case "iteration":
				x.Cases[2].IterationAfter++
			case "terminal":
				x.Cases[2].TerminalEvents = 1
			case "parity":
				x.Cases[1].Resolutions = 0
			case "binding":
				x.Cases[0].BindingMatches = false
			case "privacy":
				x.Cases[0].RunID = "body with spaces"
			case "version":
				x.Version = "unsupported"
			case "duplicate":
				x.Cases[7].Resolutions = 1
			case "unknown":
				x.Cases[13].ProviderCalls = 1
			}
			bad, _ := json.Marshal(x)
			if _, err := ReplayHostResumeAdmissionFixtureJSON(bad); err == nil || !strings.Contains(err.Error(), "host_resume.replay_drift") {
				t.Fatalf("drift err=%v", err)
			}
		})
	}
	for _, bad := range [][]byte{nil, []byte(`{"version":"host_resume_admission.v1","body":"secret","cases":[]}`), append(raw, []byte(` {}`)...)} {
		if _, err := ReplayHostResumeAdmissionFixtureJSON(bad); err == nil {
			t.Fatal("malformed fixture accepted")
		}
	}
}
