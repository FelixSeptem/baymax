package planbridge

import (
	"context"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
)

type workflowContinuationEvidence struct{}

func (workflowContinuationEvidence) ContinuationCapabilities(context.Context) (types.ContinuationCapabilities, error) {
	return types.ContinuationCapabilities{
		ProfileVersion: "native_continuation.v1", SourceNamespace: "workflow", ContractVersion: CapabilityManifestContractVersion, ModuleRevision: "workflow-v1", APIVersion: CapabilityManifestAPIVersion, RetentionPolicy: "same-host-single-writer", ExpiryPolicy: "24h", CheckpointCreation: true, CrossProcessRestore: true,
		PauseReasons:       []string{"input_required", "approval_required", "budget_exhausted"},
		EvidenceReferences: map[string]string{"checkpoint_creation": "fixture:workflow-continuation.v1", "cross_process_restore": "fixture:workflow-continuation.v1", "input_required": "fixture:workflow-continuation.v1", "approval_required": "fixture:workflow-continuation.v1", "budget_exhausted": "fixture:workflow-continuation.v1"},
	}, nil
}

func TestWorkflowContinuationManifestEnablesOnlyWorkflowCells(t *testing.T) {
	manifest, err := CapabilityManifestWithWorkflowContinuation(context.Background(), workflowContinuationEvidence{})
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	reader, err := NewCapabilityManifestReader(manifest)
	if err != nil {
		t.Fatalf("reader: %v", err)
	}
	workflow, err := reader.Capability(ExecutionFormWorkflow)
	if err != nil {
		t.Fatalf("workflow: %v", err)
	}
	for _, state := range []CapabilityState{workflow.CheckpointCreation, workflow.CrossProcessRestore, workflow.InputRequired, workflow.ApprovalRequired, workflow.BudgetExhausted} {
		if state != CapabilitySupported {
			t.Fatalf("workflow continuation state = %q", state)
		}
	}
	for _, form := range []ExecutionForm{ExecutionFormDirect, ExecutionFormSubagent, ExecutionFormTeam, ExecutionFormMailboxAsync} {
		entry, err := reader.Capability(form)
		if err != nil {
			t.Fatal(err)
		}
		if entry.CrossProcessRestore != CapabilityUnsupported {
			t.Fatalf("%s inherited workflow restore: %#v", form, entry)
		}
	}
}

func TestWorkflowContinuationManifestUsesConfiguredWorkflowOwner(t *testing.T) {
	store, err := workflow.NewFileWorkflowContinuationStore(t.TempDir(), "workflow")
	if err != nil {
		t.Fatal(err)
	}
	engine := workflow.New(workflow.WithWorkflowContinuationStore(store, "workflow"))
	manifest, err := CapabilityManifestWithWorkflowContinuation(context.Background(), engine)
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	reader, err := NewCapabilityManifestReader(manifest)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := reader.Capability(ExecutionFormWorkflow)
	if err != nil || entry.SourceContinuation == nil || entry.SourceContinuation.SourceNamespace != "workflow" || entry.CrossProcessRestore != CapabilitySupported {
		t.Fatalf("workflow owner capability = %#v err=%v", entry, err)
	}
	memory := workflow.New(workflow.WithWorkflowContinuationStore(workflow.NewMemoryWorkflowContinuationStore("workflow"), "workflow"))
	if _, err := CapabilityManifestWithWorkflowContinuation(context.Background(), memory); err == nil {
		t.Fatal("memory store advertised cross-process workflow restore")
	}
}

func TestCapabilityManifestRequiresExplicitDimensionAndPreservesSnapshot(t *testing.T) {
	manifest := CapabilityManifest{
		SchemaVersion:   CapabilityManifestSchemaVersion,
		ContractVersion: CapabilityManifestContractVersion,
		ModuleRevision:  "revision",
		APIVersion:      CapabilityManifestAPIVersion,
		Forms: []ExecutionFormCapability{{
			Form:                ExecutionFormMailboxAsync,
			Admission:           CapabilitySupported,
			NativeEvents:        CapabilitySupported,
			DurableHistory:      CapabilityUnsupported,
			AfterCursorReplay:   CapabilityUnsupported,
			CheckpointCreation:  CapabilityUnsupported,
			CrossProcessRestore: CapabilityUnsupported,
			InputRequired:       CapabilityUnsupported,
			ApprovalRequired:    CapabilityUnsupported,
			BudgetExhausted:     CapabilityUnsupported,
			Cancellation:        CapabilitySupported,
			Timeout:             CapabilitySupported,
			CausalRetry:         CapabilitySupported,
			MailboxContinuation: CapabilitySupported,
			RetentionPolicy:     "source-owned",
			EvidenceReference:   "fixture:mailbox-planbridge.v1",
			UnsupportedReasons: map[string]string{
				"durable_history":       "source journal is not part of Spec 1",
				"after_cursor_replay":   "source journal is not part of Spec 1",
				"checkpoint_creation":   "durable checkpoint is part of Spec 3",
				"cross_process_restore": "durable checkpoint is part of Spec 3",
				"input_required":        "mailbox pause continuation is part of Spec 3",
				"approval_required":     "mailbox pause continuation is part of Spec 3",
				"budget_exhausted":      "mailbox pause continuation is part of Spec 3",
			},
		}},
	}

	reader, err := NewCapabilityManifestReader(manifest)
	if err != nil {
		t.Fatalf("NewCapabilityManifestReader() error = %v", err)
	}
	entry, err := reader.Capability(ExecutionFormMailboxAsync)
	if err != nil {
		t.Fatalf("Capability() error = %v", err)
	}
	entry.UnsupportedReasons["durable_history"] = "mutated"
	again, err := reader.Capability(ExecutionFormMailboxAsync)
	if err != nil {
		t.Fatalf("Capability() after mutation error = %v", err)
	}
	if got := again.UnsupportedReasons["durable_history"]; got != "source journal is not part of Spec 1" {
		t.Fatalf("manifest snapshot was mutable, reason = %q", got)
	}

	manifest.Forms[0].Timeout = CapabilityUnknown
	if _, err := NewCapabilityManifestReader(manifest); err == nil {
		t.Fatal("NewCapabilityManifestReader() accepted missing timeout declaration")
	}
	manifest.Forms[0].Timeout = CapabilityUnsupported
	delete(manifest.Forms[0].UnsupportedReasons, "timeout")
	if _, err := NewCapabilityManifestReader(manifest); err == nil {
		t.Fatal("NewCapabilityManifestReader() accepted unsupported timeout without reason")
	}
}

func TestDefaultCapabilityManifestDeclaresEveryExecutionForm(t *testing.T) {
	reader, err := NewCapabilityManifestReader(DefaultCapabilityManifest())
	if err != nil {
		t.Fatalf("DefaultCapabilityManifest() invalid: %v", err)
	}
	for _, form := range []ExecutionForm{ExecutionFormDirect, ExecutionFormSubagent, ExecutionFormTeam, ExecutionFormWorkflow, ExecutionFormMailboxAsync} {
		entry, err := reader.Capability(form)
		if err != nil {
			t.Fatalf("Capability(%q) error = %v", form, err)
		}
		if entry.Admission == CapabilityUnknown || entry.NativeEvents == CapabilityUnknown || entry.Cancellation == CapabilityUnknown || entry.Timeout == CapabilityUnknown || entry.CausalRetry == CapabilityUnknown {
			t.Fatalf("Capability(%q) has implicit dimension: %#v", form, entry)
		}
	}
}

func TestContinuationManifestDefaultCellsHaveStableSourceLimitations(t *testing.T) {
	manifest := DefaultCapabilityManifest()
	if manifest.SchemaVersion != "host_runtime_capability_manifest.v1" {
		t.Fatal("additive continuation declarations changed manifest version")
	}
	for _, entry := range manifest.Forms {
		want := "composite_graph_restore_unsupported"
		if entry.Form == ExecutionFormDirect {
			want = "durable_source_not_configured"
		}
		for dimension, state := range map[string]CapabilityState{
			"checkpoint_creation": entry.CheckpointCreation, "cross_process_restore": entry.CrossProcessRestore,
			"input_required": entry.InputRequired, "approval_required": entry.ApprovalRequired, "budget_exhausted": entry.BudgetExhausted,
		} {
			if state != CapabilityUnsupported || entry.UnsupportedReasons[dimension] != want {
				t.Errorf("%s.%s = %q (%q), want unsupported (%q)", entry.Form, dimension, state, entry.UnsupportedReasons[dimension], want)
			}
		}
	}
}

func TestContinuationManifestRejectsUnsupportedEvidencePromotion(t *testing.T) {
	for _, form := range []ExecutionForm{ExecutionFormDirect, ExecutionFormSubagent, ExecutionFormTeam, ExecutionFormWorkflow, ExecutionFormMailboxAsync} {
		for _, dimension := range []string{"checkpoint_creation", "cross_process_restore", "input_required", "approval_required", "budget_exhausted"} {
			t.Run(string(form)+"/"+dimension, func(t *testing.T) {
				manifest := DefaultCapabilityManifest()
				for i := range manifest.Forms {
					entry := &manifest.Forms[i]
					if entry.Form != form {
						continue
					}
					switch dimension {
					case "checkpoint_creation":
						entry.CheckpointCreation = CapabilitySupported
					case "cross_process_restore":
						entry.CrossProcessRestore = CapabilitySupported
					case "input_required":
						entry.InputRequired = CapabilitySupported
					case "approval_required":
						entry.ApprovalRequired = CapabilitySupported
					case "budget_exhausted":
						entry.BudgetExhausted = CapabilitySupported
					}
					entry.EvidenceReference = "package:core/runner"
				}
				if _, err := NewCapabilityManifestReader(manifest); err == nil {
					t.Fatal("accepted supported continuation cell with only package-presence evidence")
				}
			})
		}
	}
}
