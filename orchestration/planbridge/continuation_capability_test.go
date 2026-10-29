package planbridge

import (
	"context"
	"encoding/json"
	"errors"
	"os"
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
		Deployment:         types.SameHostSingleWriterScope("runner/local", "contract:deployment"),
	}}
}

func TestContinuationManifestQueriesSourceAndDetachesCompleteEvidence(t *testing.T) {
	source := evidencedContinuationSource()
	source.caps.SourceNamespace = "workflow"
	source.caps.Deployment = types.SameHostSingleWriterScope("workflow", "contract:deployment")
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

func TestContinuationManifestPromotesOnlyEvidencedPauseCells(t *testing.T) {
	source := evidencedContinuationSource()
	source.caps.SourceNamespace = "workflow"
	source.caps.Deployment = types.SameHostSingleWriterScope("workflow", "contract:deployment")
	source.caps.PauseReasons = nil
	source.caps.EvidenceReferences = map[string]string{"checkpoint_creation": "contract:save", "cross_process_restore": "contract:subprocess-restore"}
	source.caps.PauseReasonEvidence = map[string]types.PauseReasonCapability{
		"input_required":    {Supported: true, EvidenceReference: "contract:input-restore"},
		"approval_required": {UnsupportedReason: "approval safe point unavailable"},
		"budget_exhausted":  {UnsupportedReason: "budget evidence unavailable"},
	}
	manifest, err := CapabilityManifestWithWorkflowContinuation(context.Background(), source)
	if err != nil {
		t.Fatalf("%v manifest=%#v", err, manifest)
	}
	reader, err := NewCapabilityManifestReader(manifest)
	if err != nil {
		t.Fatal(err)
	}
	workflow, err := reader.Capability(ExecutionFormWorkflow)
	if err != nil {
		t.Fatal(err)
	}
	if workflow.InputRequired != CapabilitySupported || workflow.ApprovalRequired != CapabilityUnsupported || workflow.BudgetExhausted != CapabilityUnsupported {
		t.Fatalf("partial workflow matrix promoted incorrectly: %#v", workflow)
	}
	if workflow.UnsupportedReasons["approval_required"] == "" || workflow.UnsupportedReasons["budget_exhausted"] == "" {
		t.Fatalf("partial workflow matrix lost stable reasons: %#v", workflow.UnsupportedReasons)
	}
}

func TestVersionedPauseReasonMatrixFixtures(t *testing.T) {
	var fixtures struct {
		SchemaVersion string `json:"schema_version"`
		Fixtures      []struct {
			Name       string                         `json:"name"`
			Expected   ExecutionForm                  `json:"expected_form"`
			Capability types.ContinuationCapabilities `json:"capabilities"`
		} `json:"fixtures"`
	}
	raw, err := os.ReadFile("testdata/pause_reason_matrix.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &fixtures); err != nil {
		t.Fatal(err)
	}
	if fixtures.SchemaVersion != "continuation_pause_reason_fixtures.v1" || len(fixtures.Fixtures) != 3 {
		t.Fatalf("fixture version/count = %q/%d", fixtures.SchemaVersion, len(fixtures.Fixtures))
	}
	for _, fixture := range fixtures.Fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			source := &continuationSource{caps: fixture.Capability}
			var manifest CapabilityManifest
			var err error
			switch fixture.Expected {
			case ExecutionFormWorkflow:
				manifest, err = CapabilityManifestWithWorkflowContinuation(context.Background(), source)
			case ExecutionFormMailboxAsync:
				manifest, err = CapabilityManifestWithMailboxContinuation(context.Background(), source)
			default:
				t.Fatalf("unsupported fixture form %q", fixture.Expected)
			}
			if fixture.Name == "sibling_owner_mismatch" {
				codeOf(t, err, UnsupportedVersion)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			reader, err := NewCapabilityManifestReader(manifest)
			if err != nil {
				t.Fatal(err)
			}
			entry, err := reader.Capability(fixture.Expected)
			if err != nil {
				t.Fatal(err)
			}
			if fixture.Expected == ExecutionFormWorkflow {
				if entry.InputRequired != CapabilitySupported || entry.ApprovalRequired != CapabilityUnsupported || entry.BudgetExhausted != CapabilityUnsupported {
					t.Fatalf("partial matrix projection = %#v", entry)
				}
				if entry.SourceContinuation.PauseReasonEvidence["input_required"].EvidenceReference != "fixture:pause-matrix.v1/workflow-input" {
					t.Fatal("manifest did not retain exact source evidence")
				}
				if entry.SourceContinuation.Deployment.OwnerNamespace != "workflow" {
					t.Fatalf("workflow deployment owner = %q", entry.SourceContinuation.Deployment.OwnerNamespace)
				}
				return
			}
			if entry.InputRequired != CapabilityUnsupported || entry.ApprovalRequired != CapabilityUnsupported || entry.BudgetExhausted != CapabilityUnsupported {
				t.Fatalf("mailbox lifecycle fixture promoted handler cells: %#v", entry)
			}
			if entry.HostContextAssembly != CapabilityUnsupported {
				t.Fatal("mailbox lifecycle fixture promoted handler context assembly")
			}
			for _, reason := range []string{"input_required", "approval_required", "budget_exhausted"} {
				if entry.UnsupportedReasons[reason] != types.UnsupportedReasonMailboxHandlerSafePoint {
					t.Fatalf("mailbox.%s limitation = %q", reason, entry.UnsupportedReasons[reason])
				}
			}
		})
	}
}

func TestBoundPauseRequirementsUseExactOwnerEvidenceAtCompileAndActivation(t *testing.T) {
	runner := &countedRunner{}
	source := &evidencedRunner{countedRunner: runner, caps: evidencedContinuationSource().caps}
	source.caps.PauseReasons = nil
	source.caps.EvidenceReferences = map[string]string{
		"checkpoint_creation":   "contract:save",
		"cross_process_restore": "contract:restore",
	}
	source.caps.PauseReasonEvidence = map[string]types.PauseReasonCapability{
		"input_required":    {Supported: true, EvidenceReference: "contract:input"},
		"approval_required": {UnsupportedReason: "approval safe point unavailable"},
		"budget_exhausted":  {UnsupportedReason: "budget safe point unavailable"},
	}
	d, binding, _ := directFixture(t)
	binding.Direct.Runner = source
	d.Deployment = &types.DeploymentRequest{OwnerNamespace: "runner/local", WriterFence: "fence:runner/local"}
	d.RequiredCapabilities = []string{"approval_required"}
	if _, err := Describe(d, binding, Limits{}); err == nil {
		t.Fatal("Describe accepted a pause reason explicitly unsupported by its owner")
	} else {
		codeOf(t, err, UnsupportedCapability)
	}
	if runner.calls != 0 {
		t.Fatal("unsupported reason activated source during Describe")
	}

	d.RequiredCapabilities = []string{"input_required"}
	d, err := Describe(d, binding, Limits{})
	if err != nil {
		t.Fatalf("Describe rejected supported owner reason: %v", err)
	}
	plan, err := Compile(d, binding, Limits{})
	if err != nil {
		t.Fatalf("Compile rejected supported owner reason: %v", err)
	}
	source.caps.PauseReasonEvidence["input_required"] = types.PauseReasonCapability{UnsupportedReason: "safe point evidence became stale"}
	for _, stream := range []bool{false, true} {
		var runErr error
		if stream {
			_, runErr = plan.Stream(context.Background(), Execution{})
		} else {
			_, runErr = plan.Run(context.Background(), Execution{})
		}
		codeOf(t, runErr, UnsupportedCapability)
	}
	if runner.calls != 0 {
		t.Fatal("stale owner reason activated source")
	}
}

func TestBoundPauseRequirementsRequireMatchingDeploymentOwnerAndVersion(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*types.ContinuationCapabilities, *types.DeploymentRequest)
		want   Code
	}{
		{
			name: "foreign deployment request owner",
			mutate: func(_ *types.ContinuationCapabilities, request *types.DeploymentRequest) {
				request.OwnerNamespace = "runner/other"
				request.WriterFence = "fence:runner/other"
			},
			want: SourceUnavailable,
		},
		{
			name: "missing deployment evidence",
			mutate: func(caps *types.ContinuationCapabilities, _ *types.DeploymentRequest) {
				caps.Deployment = types.DeploymentScope{}
			},
			want: UnsupportedCapability,
		},
		{
			name: "foreign API version",
			mutate: func(caps *types.ContinuationCapabilities, _ *types.DeploymentRequest) {
				caps.APIVersion = "host_runtime_capability_manifest.v2"
			},
			want: UnsupportedVersion,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := &countedRunner{}
			source := &evidencedRunner{countedRunner: runner, caps: evidencedContinuationSource().caps}
			source.caps.PauseReasons = nil
			source.caps.EvidenceReferences = map[string]string{
				"checkpoint_creation": "contract:save", "cross_process_restore": "contract:restore",
			}
			source.caps.PauseReasonEvidence = map[string]types.PauseReasonCapability{
				"input_required":    {Supported: true, EvidenceReference: "contract:input"},
				"approval_required": {UnsupportedReason: "approval safe point unavailable"},
				"budget_exhausted":  {UnsupportedReason: "budget safe point unavailable"},
			}
			d, binding, _ := directFixture(t)
			binding.Direct.Runner = source
			d.RequiredCapabilities = []string{"input_required"}
			request := types.DeploymentRequest{OwnerNamespace: "runner/local", WriterFence: "fence:runner/local"}
			test.mutate(&source.caps, &request)
			d.Deployment = &request
			_, err := Describe(d, binding, Limits{})
			codeOf(t, err, test.want)
			if runner.calls != 0 {
				t.Fatal("invalid source binding activated runner")
			}
		})
	}
}

func TestChildContinuationManifestPromotesOnlyChildForm(t *testing.T) {
	source := evidencedContinuationSource()
	source.caps.SourceNamespace = "child"
	source.caps.Deployment = types.SameHostSingleWriterScope("child", "contract:deployment")
	manifest, err := CapabilityManifestWithChildContinuation(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := NewCapabilityManifestReader(manifest)
	if err != nil {
		t.Fatal(err)
	}
	child, err := reader.Capability(ExecutionFormSubagent)
	if err != nil || child.CrossProcessRestore != CapabilitySupported || child.SourceContinuation == nil {
		t.Fatalf("child capability = %#v, err=%v", child, err)
	}
	for _, form := range []ExecutionForm{ExecutionFormTeam, ExecutionFormWorkflow, ExecutionFormMailboxAsync} {
		entry, err := reader.Capability(form)
		if err != nil || entry.CrossProcessRestore != CapabilityUnsupported {
			t.Fatalf("unsupported form %s = %#v, err=%v", form, entry, err)
		}
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
