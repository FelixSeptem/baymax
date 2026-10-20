package types

import (
	"strings"
	"testing"
)

func TestContinuationCapabilitiesRequireCompleteSourceEvidence(t *testing.T) {
	complete := ContinuationCapabilities{
		ProfileVersion: "native_continuation.v1", SourceNamespace: "runner/source", ContractVersion: "baymax.runtime.contract.v1",
		ModuleRevision: "revision", APIVersion: "host_runtime_capability_manifest.v1", RetentionPolicy: "24h", ExpiryPolicy: "min(deadline,24h)",
		CheckpointCreation: true, CrossProcessRestore: true, PauseReasons: []string{"input_required", "approval_required", "budget_exhausted"},
		EvidenceReferences: map[string]string{"checkpoint_creation": "contract:checkpoint", "cross_process_restore": "contract:process-reopen", "input_required": "contract:input", "approval_required": "contract:approval", "budget_exhausted": "contract:budget"},
	}
	if err := complete.Validate(); err != nil {
		t.Fatal(err)
	}
	clone := complete.Clone()
	if clone.EvidenceReferences["input_required"] != complete.EvidenceReferences["input_required"] {
		t.Fatal("clone lost evidence")
	}
	clone.EvidenceReferences["input_required"] = "changed"
	clone.PauseReasons[0] = "changed"
	if complete.EvidenceReferences["input_required"] == "changed" || complete.PauseReasons[0] == "changed" {
		t.Fatal("clone aliases source")
	}
	for _, change := range []func(*ContinuationCapabilities){
		func(c *ContinuationCapabilities) { c.ProfileVersion = "future" },
		func(c *ContinuationCapabilities) { c.SourceNamespace = "" },
		func(c *ContinuationCapabilities) { c.ExpiryPolicy = "" },
		func(c *ContinuationCapabilities) { c.CrossProcessRestore = false },
		func(c *ContinuationCapabilities) { c.PauseReasons = c.PauseReasons[:2] },
		func(c *ContinuationCapabilities) {
			c.PauseReasons = []string{"input_required", "input_required", "budget_exhausted"}
		},
		func(c *ContinuationCapabilities) {
			c.EvidenceReferences = map[string]string{"checkpoint_creation": "package:runner"}
		},
		func(c *ContinuationCapabilities) { c.ModuleRevision = strings.Repeat("x", 257) },
		func(c *ContinuationCapabilities) {
			c.UnsupportedReasons = map[string]string{"input_required": strings.Repeat("x", 257)}
		},
		func(c *ContinuationCapabilities) { c.UnsupportedReasons = map[string]string{"unknown": "reason"} },
	} {
		invalid := complete
		change(&invalid)
		if err := invalid.Validate(); err == nil {
			t.Fatal("accepted incomplete or unbounded capability evidence")
		}
	}
	complete.ModuleRevision = strings.Repeat("x", 256)
	if err := complete.Validate(); err != nil {
		t.Fatalf("exact boundary rejected: %v", err)
	}
}
