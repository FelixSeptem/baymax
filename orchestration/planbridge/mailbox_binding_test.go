package planbridge

import (
	"context"
	"testing"

	"github.com/FelixSeptem/baymax/orchestration/mailbox"
)

func TestMailboxPlanRequiresManifestAndDelegatesOneLogicalAdmission(t *testing.T) {
	owner, err := mailbox.New(mailbox.NewMemoryStore(mailbox.Policy{}))
	if err != nil {
		t.Fatal(err)
	}
	d := Descriptor{
		SchemaVersion: SchemaVersion, PlanID: "mailbox-plan", PlanVersion: 1,
		Shape: ShapeDirect, Delivery: DeliveryMailbox, Trigger: TriggerManual,
		AuthorityDigest: "authority", ToolCapabilityDigest: "tools",
		InputRef: "input", InputDigest: "input-digest", HandlerRef: "handler", HandlerDigest: "handler-digest",
	}
	b := Binding{
		SemanticDigest: "semantics",
		Admit: func(context.Context, Identity) (Admission, error) {
			return Admission{AuthorityDigest: "authority", ToolCapabilityDigest: "tools", SemanticDigest: "semantics"}, nil
		},
		Mailbox: &MailboxBinding{Owner: owner, Binding: mailbox.HostPlanBinding{
			ContractVersion: mailbox.HostPlanBindingContractVersion,
			MessageID:       "message-1", IdempotencyKey: "logical-1", RunID: "run-1", ParentRunID: "parent-1", CausationID: "cause-1",
			AuthorityDigest: "authority", PlanDigest: "", WriteScope: "scope-1", EffectNamespace: "effect-1",
			Occurrence: "occurrence-1", HostCorrelation: "host-1", ResourceLimit: 1,
		}},
	}
	if _, err := Describe(d, b, Limits{}); err == nil {
		t.Fatal("Describe() accepted mailbox binding without manifest")
	}
	b.Mailbox.Manifest = testMailboxManifest(t)
	d, err = Describe(d, b, Limits{})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	p, err := Compile(d, b, Limits{})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if b.Mailbox.Binding.PlanDigest != "" {
		t.Fatalf("Compile() mutated the caller mailbox binding: %#v", b.Mailbox.Binding)
	}
	result, err := p.Run(context.Background(), Execution{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Mailbox == nil || result.Mailbox.MessageID != "message-1" || result.Mailbox.Duplicate {
		t.Fatalf("first mailbox result = %#v", result.Mailbox)
	}
	if result.Mailbox.PlanDigest != p.Identity().Digest {
		t.Fatalf("mailbox plan digest = %q, want compiled digest %q", result.Mailbox.PlanDigest, p.Identity().Digest)
	}
	again, err := p.Stream(context.Background(), Execution{})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if again.Mailbox == nil || !again.Mailbox.Duplicate || again.Mailbox.ParentRunID != "parent-1" || again.Mailbox.EffectNamespace != "effect-1" {
		t.Fatalf("duplicate mailbox result = %#v", again.Mailbox)
	}
	snapshot, err := owner.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Records) != 1 {
		t.Fatalf("logical mailbox admission count = %d, want 1", len(snapshot.Records))
	}
}

func TestMailboxPlanDigestConflictRejectsBeforeAdmission(t *testing.T) {
	owner, err := mailbox.New(mailbox.NewMemoryStore(mailbox.Policy{}))
	if err != nil {
		t.Fatal(err)
	}
	d := Descriptor{SchemaVersion: SchemaVersion, PlanID: "digest-plan", PlanVersion: 1, Shape: ShapeDirect, Delivery: DeliveryMailbox, Trigger: TriggerManual, AuthorityDigest: "authority", ToolCapabilityDigest: "tools", InputRef: "input", InputDigest: "input-digest", HandlerRef: "handler", HandlerDigest: "handler-digest"}
	b := Binding{SemanticDigest: "semantics", Admit: func(context.Context, Identity) (Admission, error) {
		return Admission{AuthorityDigest: "authority", ToolCapabilityDigest: "tools", SemanticDigest: "semantics"}, nil
	}, Mailbox: &MailboxBinding{Owner: owner, Manifest: testMailboxManifest(t), Binding: mailbox.HostPlanBinding{
		ContractVersion: mailbox.HostPlanBindingContractVersion, MessageID: "digest-message", IdempotencyKey: "digest-logical", RunID: "run", ParentRunID: "parent", CausationID: "cause", AuthorityDigest: "authority", PlanDigest: "sha256:wrong", WriteScope: "scope", EffectNamespace: "effects", Occurrence: "occurrence", HostCorrelation: "host", ResourceLimit: 1,
	}}}
	if _, err = Describe(d, b, Limits{}); err == nil {
		t.Fatal("Describe() accepted a mismatched mailbox plan digest")
	}
	snapshot, err := owner.Snapshot(context.Background())
	if err != nil || len(snapshot.Records) != 0 {
		t.Fatalf("digest rejection mutated mailbox: records=%d err=%v", len(snapshot.Records), err)
	}
}

func TestMailboxActivationRechecksOwnerAvailability(t *testing.T) {
	owner, err := mailbox.New(mailbox.NewMemoryStore(mailbox.Policy{}))
	if err != nil {
		t.Fatal(err)
	}
	d := Descriptor{SchemaVersion: SchemaVersion, PlanID: "availability-plan", PlanVersion: 1, Shape: ShapeDirect, Delivery: DeliveryMailbox, Trigger: TriggerManual, AuthorityDigest: "authority", ToolCapabilityDigest: "tools", InputRef: "input", InputDigest: "input-digest", HandlerRef: "handler", HandlerDigest: "handler-digest"}
	b := Binding{SemanticDigest: "semantics", Admit: func(context.Context, Identity) (Admission, error) {
		return Admission{AuthorityDigest: "authority", ToolCapabilityDigest: "tools", SemanticDigest: "semantics"}, nil
	}, Mailbox: &MailboxBinding{Owner: owner, Manifest: testMailboxManifest(t), Binding: mailbox.HostPlanBinding{ContractVersion: mailbox.HostPlanBindingContractVersion, MessageID: "availability-message", IdempotencyKey: "availability-logical", RunID: "run", ParentRunID: "parent", CausationID: "cause", AuthorityDigest: "authority", WriteScope: "scope", EffectNamespace: "effects", Occurrence: "occurrence", HostCorrelation: "host", ResourceLimit: 1}}}
	d, err = Describe(d, b, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	p, err := Compile(d, b, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	p.binding.Mailbox.Owner = nil
	if _, err = p.Run(context.Background(), Execution{}); err == nil {
		t.Fatal("Run() accepted an unavailable mailbox owner")
	}
	snapshot, snapshotErr := owner.Snapshot(context.Background())
	if snapshotErr != nil || len(snapshot.Records) != 0 {
		t.Fatalf("unavailable activation mutated mailbox: records=%d err=%v", len(snapshot.Records), snapshotErr)
	}
}

func testMailboxManifest(t *testing.T) *CapabilityManifestReader {
	t.Helper()
	manifest := CapabilityManifest{
		SchemaVersion: CapabilityManifestSchemaVersion, ContractVersion: CapabilityManifestContractVersion, ModuleRevision: "revision", APIVersion: CapabilityManifestAPIVersion,
		Forms: []ExecutionFormCapability{{
			Form: ExecutionFormMailboxAsync, Admission: CapabilitySupported, NativeEvents: CapabilitySupported,
			DurableHistory: CapabilityUnsupported, AfterCursorReplay: CapabilityUnsupported,
			CheckpointCreation: CapabilityUnsupported, CrossProcessRestore: CapabilityUnsupported,
			InputRequired: CapabilityUnsupported, ApprovalRequired: CapabilityUnsupported, BudgetExhausted: CapabilityUnsupported,
			Cancellation: CapabilitySupported, Timeout: CapabilitySupported, CausalRetry: CapabilitySupported,
			MailboxContinuation: CapabilitySupported, RetentionPolicy: "source-owned", EvidenceReference: "fixture:mailbox-planbridge.v1",
			UnsupportedReasons: map[string]string{
				"durable_history": "Spec 2", "after_cursor_replay": "Spec 2", "checkpoint_creation": "Spec 3",
				"cross_process_restore": "Spec 3", "input_required": "Spec 3", "approval_required": "Spec 3", "budget_exhausted": "Spec 3",
			},
		}},
	}
	reader, err := NewCapabilityManifestReader(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return reader
}
