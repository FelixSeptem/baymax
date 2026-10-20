package planbridge

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
)

type countedRunner struct {
	calls int
	input string
}

func TestHostExecutionPlanCanonicalGoldenVector(t *testing.T) {
	d, b, _ := directFixture(t)
	p, err := Compile(d, b, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/host_execution_plan.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	want = []byte(strings.TrimSpace(string(want)))
	if got := p.Canonical(); string(got) != string(want) {
		t.Fatalf("canonical vector drift\nwant=%s\ngot=%s", want, got)
	}
	if p.Identity().Digest != digest(want) {
		t.Fatalf("digest not derived from canonical vector: %s", p.Identity().Digest)
	}
}

func (r *countedRunner) Run(_ context.Context, q types.RunRequest, _ types.EventHandler) (types.RunResult, error) {
	r.calls++
	r.input = q.Input
	return types.RunResult{RunID: q.RunID}, nil
}
func (r *countedRunner) Stream(c context.Context, q types.RunRequest, h types.EventHandler) (types.RunResult, error) {
	return r.Run(c, q, h)
}
func directFixture(t *testing.T) (Descriptor, Binding, *countedRunner) {
	t.Helper()
	r := &countedRunner{}
	d := Descriptor{SchemaVersion: SchemaVersion, PlanID: "plan", PlanVersion: 1, Shape: ShapeDirect, Delivery: DeliveryInline, Trigger: TriggerManual, AuthorityDigest: "authority", ToolCapabilityDigest: "tools", InputRef: "input", InputDigest: "input-digest", HandlerRef: "handler", HandlerDigest: "handler-digest"}
	b := Binding{SemanticDigest: "semantics", Direct: &DirectBinding{Runner: r, Request: types.RunRequest{RunID: "run", Input: "secret-body"}}, Admit: func(context.Context, Identity) (Admission, error) {
		return Admission{AuthorityDigest: "authority", ToolCapabilityDigest: "tools", SemanticDigest: "semantics"}, nil
	}}
	var e error
	d, e = Describe(d, b, Limits{})
	if e != nil {
		t.Fatal(e)
	}
	return d, b, r
}
func codeOf(t *testing.T, e error, code Code) {
	t.Helper()
	var pe *Error
	if !errors.As(e, &pe) || pe.Code != code {
		t.Fatalf("want %s, got %v", code, e)
	}
	if strings.Contains(e.Error(), "secret") {
		t.Fatal("body leaked")
	}
}
func TestHostExecutionPlanCompileDetachedAndPreflight(t *testing.T) {
	d, b, r := directFixture(t)
	p, e := Compile(d, b, Limits{})
	if e != nil {
		t.Fatal(e)
	}
	before := string(p.Canonical())
	d.SourceFacts["run_id"] = "changed"
	b.Direct.Request.Input = "changed"
	if _, e = p.Run(context.Background(), Execution{}); e != nil {
		t.Fatal(e)
	}
	if r.input != "secret-body" || string(p.Canonical()) != before {
		t.Fatal("snapshot mutated")
	}
	d, b, r = directFixture(t)
	d.SourceFacts["run_id"] = "different"
	_, e = Compile(d, b, Limits{})
	codeOf(t, e, BindingMismatch)
	if r.calls != 0 {
		t.Fatal("activation during rejection")
	}
}
func TestHostExecutionPlanErrorsAndLimits(t *testing.T) {
	cases := []struct {
		name   string
		code   Code
		change func(*Descriptor)
	}{
		{"version", UnsupportedVersion, func(d *Descriptor) { d.SchemaVersion = "future" }},
		{"identity", InvalidPlan, func(d *Descriptor) { d.PlanVersion = 0 }},
		{"mailbox", UnsupportedCapability, func(d *Descriptor) { d.Delivery = DeliveryMailbox }},
		{"resume", UnsupportedCapability, func(d *Descriptor) { d.Trigger = TriggerResumed }},
		{"ref", LimitExceeded, func(d *Descriptor) { d.PlanID = strings.Repeat("x", 1025) }},
		{"capability", UnsupportedCapability, func(d *Descriptor) { d.RequiredCapabilities = []string{"durable_continuation"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, b, r := directFixture(t)
			tc.change(&d)
			_, e := Compile(d, b, Limits{})
			codeOf(t, e, tc.code)
			if r.calls != 0 {
				t.Fatal("activated")
			}
		})
	}
	d, b, _ := directFixture(t)
	for _, l := range []Limits{{ReferenceBytes: -1}, {ReferenceBytes: 1025}, {BindingBytes: 65537}} {
		_, e := Compile(d, b, l)
		codeOf(t, e, InvalidPlan)
	}
	d.PlanID = strings.Repeat("x", 1024)
	if _, e := Compile(d, b, Limits{}); e != nil {
		t.Fatal(e)
	}
	b.Direct.Request.Input = strings.Repeat("secret", 12000)
	_, e := Compile(d, b, Limits{})
	codeOf(t, e, LimitExceeded)
}
func TestHostExecutionPlanAdmissionAndDrift(t *testing.T) {
	d, b, r := directFixture(t)
	deny := errors.New("host denied")
	b.Admit = func(context.Context, Identity) (Admission, error) { return Admission{}, deny }
	p, e := Compile(d, b, Limits{})
	if e != nil {
		t.Fatal(e)
	}
	_, e = p.Run(context.Background(), Execution{})
	if !errors.Is(e, deny) || r.calls != 0 {
		t.Fatal("admission bypass")
	}
	b.Admit = func(context.Context, Identity) (Admission, error) {
		return Admission{AuthorityDigest: "new", ToolCapabilityDigest: "tools", SemanticDigest: "semantics"}, nil
	}
	p, e = Compile(d, b, Limits{})
	if e != nil {
		t.Fatal(e)
	}
	_, e = p.Stream(context.Background(), Execution{})
	codeOf(t, e, BindingMismatch)
	if r.calls != 0 {
		t.Fatal("drift activated")
	}
}

func TestCompositeContinuationRequirementsRejectBeforeBindingPreflight(t *testing.T) {
	for _, shape := range []Shape{ShapeSubagent, ShapeTeam, ShapeWorkflow, ShapeDirect} {
		for _, capability := range []string{"dynamic_pause", "checkpoint_creation", "cross_process_restore", "input_required", "approval_required", "budget_exhausted"} {
			t.Run(string(shape)+"/"+capability, func(t *testing.T) {
				d, b, runner := directFixture(t)
				d.Shape = shape
				if shape == ShapeDirect {
					d.Delivery = DeliveryMailbox
				}
				d.RequiredCapabilities = []string{capability}
				admissions := 0
				b.Admit = func(context.Context, Identity) (Admission, error) { admissions++; return Admission{}, nil }
				for _, describe := range []bool{false, true} {
					var err error
					if describe {
						_, err = Describe(d, b, Limits{})
					} else {
						_, err = Compile(d, b, Limits{})
					}
					codeOf(t, err, UnsupportedCapability)
					var planErr *Error
					if !errors.As(err, &planErr) || planErr.Field != "required_capabilities" {
						t.Fatalf("describe=%v: continuation requirement reached binding preflight: %v", describe, err)
					}
				}
				if admissions != 0 || runner.calls != 0 {
					t.Fatal("unsupported requirement activated source")
				}
			})
		}
	}
}
