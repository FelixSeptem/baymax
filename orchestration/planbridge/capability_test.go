package planbridge

import "testing"

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
