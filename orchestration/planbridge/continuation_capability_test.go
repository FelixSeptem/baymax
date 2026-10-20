package planbridge

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
)

type continuationSource struct {
	caps  types.ContinuationCapabilities
	err   error
	calls int
}

func (s *continuationSource) ContinuationCapabilities(context.Context) (types.ContinuationCapabilities, error) {
	s.calls++
	return s.caps, s.err
}

func evidencedContinuationSource() *continuationSource {
	return &continuationSource{caps: types.ContinuationCapabilities{
		ProfileVersion: "native_continuation.v1", SourceNamespace: "runner/local", ContractVersion: CapabilityManifestContractVersion,
		ModuleRevision: "verified-build", APIVersion: CapabilityManifestAPIVersion, RetentionPolicy: "24h", ExpiryPolicy: "earliest deadline or 24h",
		CheckpointCreation: true, CrossProcessRestore: true, PauseReasons: []string{"input_required", "approval_required", "budget_exhausted"},
		EvidenceReferences: map[string]string{"checkpoint_creation": "contract:save", "cross_process_restore": "contract:subprocess-restore", "input_required": "contract:input-restore", "approval_required": "contract:approval-restore", "budget_exhausted": "contract:budget-restore"},
	}}
}

func TestContinuationManifestQueriesSourceAndDetachesCompleteEvidence(t *testing.T) {
	source := evidencedContinuationSource()
	manifest, err := CapabilityManifestWithContinuation(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	if source.calls != 1 {
		t.Fatal("manifest did not query actual configured source")
	}
	source.caps.EvidenceReferences["input_required"] = "caller-mutated"
	reader, err := NewCapabilityManifestReader(manifest)
	if err != nil {
		t.Fatal(err)
	}
	direct, err := reader.Capability(ExecutionFormDirect)
	if err != nil {
		t.Fatal(err)
	}
	if direct.CheckpointCreation != CapabilitySupported || direct.CrossProcessRestore != CapabilitySupported || direct.InputRequired != CapabilitySupported || direct.ApprovalRequired != CapabilitySupported || direct.BudgetExhausted != CapabilitySupported {
		t.Fatalf("missing evidenced direct cells: %+v", direct)
	}
	if direct.SourceContinuation == nil || direct.SourceContinuation.EvidenceReferences["input_required"] != "contract:input-restore" {
		t.Fatal("source evidence not detached")
	}
	direct.SourceContinuation.PauseReasons[0] = "caller-mutated"
	direct.SourceContinuation.EvidenceReferences["input_required"] = "caller-mutated"
	again, _ := reader.Capability(ExecutionFormDirect)
	if again.SourceContinuation.PauseReasons[0] != "input_required" || again.SourceContinuation.EvidenceReferences["input_required"] != "contract:input-restore" {
		t.Fatal("reader aliases source evidence")
	}
	for _, form := range []ExecutionForm{ExecutionFormSubagent, ExecutionFormTeam, ExecutionFormWorkflow, ExecutionFormMailboxAsync} {
		entry, err := reader.Capability(form)
		if err != nil {
			t.Fatal(err)
		}
		if entry.SourceContinuation != nil || entry.CheckpointCreation != CapabilityUnsupported || entry.CrossProcessRestore != CapabilityUnsupported || entry.InputRequired != CapabilityUnsupported || entry.ApprovalRequired != CapabilityUnsupported || entry.BudgetExhausted != CapabilityUnsupported {
			t.Fatalf("promoted unsupported composite: %+v", entry)
		}
	}
	if direct.DurableHistory != CapabilityUnsupported || direct.AfterCursorReplay != CapabilityUnsupported {
		t.Fatal("continuation source promoted history")
	}
}

func TestContinuationManifestFailsClosedForIncompleteOrUnavailableSource(t *testing.T) {
	for _, change := range []func(*continuationSource){
		func(s *continuationSource) { s.caps.CrossProcessRestore = false },
		func(s *continuationSource) { s.caps.PauseReasons = s.caps.PauseReasons[:2] },
		func(s *continuationSource) { delete(s.caps.EvidenceReferences, "approval_required") },
		func(s *continuationSource) {
			s.caps.EvidenceReferences["cross_process_restore"] = "package:core/runner"
		},
	} {
		source := evidencedContinuationSource()
		change(source)
		manifest, err := CapabilityManifestWithContinuation(context.Background(), source)
		codeOf(t, err, UnsupportedCapability)
		if manifest.Forms[0].CrossProcessRestore != CapabilityUnsupported {
			t.Fatal("failure returned supported fallback")
		}
		if manifest.Forms[0].UnsupportedReasons["cross_process_restore"] != LimitationSourceEvidenceMissing {
			t.Fatal("incomplete source lost stable limitation")
		}
	}
	source := evidencedContinuationSource()
	source.caps.ContractVersion = "future"
	_, err := CapabilityManifestWithContinuation(context.Background(), source)
	codeOf(t, err, UnsupportedVersion)
	unavailable := errors.New("source unavailable")
	source.err = unavailable
	_, err = CapabilityManifestWithContinuation(context.Background(), source)
	if !errors.Is(err, unavailable) {
		t.Fatal("source availability error lost", err)
	}
	_, err = CapabilityManifestWithContinuation(context.Background(), nil)
	codeOf(t, err, SourceUnavailable)
}

func TestResumedPlansRejectBeforeAnySourceCapabilityOrBindingCall(t *testing.T) {
	d, b, runner := directFixture(t)
	d.Trigger = TriggerResumed
	d.CheckpointRef = "source-checkpoint"
	b.Direct = nil
	for _, describe := range []bool{false, true} {
		var err error
		if describe {
			_, err = Describe(d, b, Limits{})
		} else {
			_, err = Compile(d, b, Limits{})
		}
		codeOf(t, err, UnsupportedCapability)
		var planErr *Error
		if !errors.As(err, &planErr) || planErr.Field != "trigger" {
			t.Fatalf("resumed plan reached binding: %v", err)
		}
		if !strings.Contains(err.Error(), LimitationPlanResumedTrigger) {
			t.Fatal("resumed rejection lost stable limitation", err)
		}
	}
	if runner.calls != 0 {
		t.Fatal("resumed plan called fresh Run")
	}
}

func TestDirectContinuationRequirementNeedsBoundRunnerEvidence(t *testing.T) {
	d, b, runner := directFixture(t)
	d.RequiredCapabilities = []string{"approval_required"}
	_, err := Compile(d, b, Limits{})
	codeOf(t, err, UnsupportedCapability)
	if runner.calls != 0 {
		t.Fatal("unsupported direct capability activated runner")
	}

	evidenced := &evidencedRunner{countedRunner: runner, caps: evidencedContinuationSource().caps}
	b.Direct.Runner = evidenced
	d, err = Describe(d, b, Limits{})
	if err != nil {
		t.Fatalf("evidenced direct requirement rejected: %v", err)
	}
}

type evidencedRunner struct {
	*countedRunner
	caps types.ContinuationCapabilities
}

func (r *evidencedRunner) ContinuationCapabilities(context.Context) (types.ContinuationCapabilities, error) {
	return r.caps.Clone(), nil
}

func TestDirectContinuationRequirementsRecheckedBeforeActivation(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(map[bool]string{false: "run", true: "stream"}[stream], func(t *testing.T) {
			d, b, runner := directFixture(t)
			source := &evidencedRunner{countedRunner: runner, caps: evidencedContinuationSource().caps}
			b.Direct.Runner = source
			d.RequiredCapabilities = []string{"approval_required"}
			var err error
			d, err = Describe(d, b, Limits{})
			if err != nil {
				t.Fatal(err)
			}
			compiled, err := Compile(d, b, Limits{})
			if err != nil {
				t.Fatal(err)
			}
			// Source evidence may become unavailable after planner admission.
			source.caps.PauseReasons = []string{"input_required", "budget_exhausted"}
			if stream {
				_, err = compiled.Stream(context.Background(), Execution{})
			} else {
				_, err = compiled.Run(context.Background(), Execution{})
			}
			codeOf(t, err, UnsupportedCapability)
			if runner.calls != 0 {
				t.Fatal("stale capability activated source")
			}
		})
	}
}
