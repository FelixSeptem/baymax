package sandboxconformance

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestExternalFailureFixturesCoverEveryRuntimeAndKind(t *testing.T) {
	fixtures := ExternalFailureFixtures()
	if err := ValidateExternalFailureFixtures(fixtures); err != nil {
		t.Fatalf("ValidateExternalFailureFixtures: %v", err)
	}
	wantRuntimes := []string{
		runtimeconfig.SecuritySandboxExternalBackendAnthropicSandboxRuntime,
		runtimeconfig.SecuritySandboxExternalBackendE2B,
		runtimeconfig.SecuritySandboxExternalBackendOpenSandbox,
	}
	got := ExternalRuntimes()
	if len(got) != len(wantRuntimes) {
		t.Fatalf("ExternalRuntimes = %#v, want %#v", got, wantRuntimes)
	}
	for i := range wantRuntimes {
		if got[i] != wantRuntimes[i] {
			t.Fatalf("ExternalRuntimes = %#v, want %#v", got, wantRuntimes)
		}
	}
	if len(fixtures) != len(wantRuntimes)*6 {
		t.Fatalf("fixture count = %d, want %d", len(fixtures), len(wantRuntimes)*6)
	}
}

func TestExternalFailureReasonsAreFrozen(t *testing.T) {
	cases := map[ExternalFailureKind]string{
		ExternalFailureLaunchFailed:       "sandbox.launch_failed",
		ExternalFailureCapabilityMismatch: "sandbox.capability_mismatch",
		ExternalFailureTimeout:            "sandbox.timeout",
		ExternalFailureOOM:                "sandbox.oom",
		ExternalFailureEgressDenied:       "sandbox.egress_denied",
		ExternalFailureBackendUnavailable: "sandbox.backend_unavailable",
	}
	for kind, want := range cases {
		got, ok := ExternalFailureReason(kind)
		if !ok || got != want {
			t.Fatalf("ExternalFailureReason(%q) = (%q,%v), want (%q,true)", kind, got, ok, want)
		}
	}
	if _, ok := ExternalFailureReason("firecracker"); ok {
		t.Fatal("unknown failure kind must not resolve")
	}
}

func TestValidateExternalFailureFixturesRejectsDrift(t *testing.T) {
	fixtures := ExternalFailureFixtures()

	mutated := append([]ExternalFailureFixture(nil), fixtures...)
	mutated[0].Expected = "sandbox.something_else"
	if err := ValidateExternalFailureFixtures(mutated); err == nil ||
		!strings.Contains(err.Error(), "frozen reason") {
		t.Fatalf("expected frozen reason drift error, got %v", err)
	}

	truncated := fixtures[:len(fixtures)-1]
	if err := ValidateExternalFailureFixtures(truncated); err == nil ||
		!strings.Contains(err.Error(), "missing fixture coverage") {
		t.Fatalf("expected coverage error, got %v", err)
	}

	unknownRuntime := append([]ExternalFailureFixture(nil), fixtures...)
	unknownRuntime[0].Runtime = "firecracker"
	if err := ValidateExternalFailureFixtures(unknownRuntime); err == nil ||
		!strings.Contains(err.Error(), "unregistered external runtime") {
		t.Fatalf("expected unregistered runtime error, got %v", err)
	}

	duplicated := append([]ExternalFailureFixture(nil), fixtures...)
	duplicated[1].ID = duplicated[0].ID
	if err := ValidateExternalFailureFixtures(duplicated); err == nil ||
		!strings.Contains(err.Error(), "duplicate fixture id") {
		t.Fatalf("expected duplicate id error, got %v", err)
	}
}

func TestExternalRunStreamParityForSuccessAndFailure(t *testing.T) {
	success := &ScriptedExecutor{
		Result: types.SandboxExecResult{ExitCode: 0, Stdout: "ok\n"},
	}
	if err := ExternalRunStreamParity(success, types.SandboxExecSpec{Command: "echo"}); err != nil {
		t.Fatalf("success parity: %v", err)
	}

	failure := &ScriptedExecutor{
		Result: types.SandboxExecResult{TimedOut: true},
		Err:    errors.New(ReasonTimeout),
	}
	if err := ExternalRunStreamParity(failure, types.SandboxExecSpec{Command: "sleep"}); err != nil {
		t.Fatalf("failure parity: %v", err)
	}

	if success.Calls != 2 {
		t.Fatalf("parity check must execute exactly twice, got %d", success.Calls)
	}
	// A path-dependent executor must be detected as drift.
	if err := ExternalRunStreamParity(&mutatingExecutor{}, types.SandboxExecSpec{Command: "echo"}); err == nil {
		t.Fatal("expected parity drift to be detected")
	}
}

// mutatingExecutor returns a different stdout on each call to model a backend
// whose Run and Stream paths disagree.
type mutatingExecutor struct {
	calls int
}

func (m *mutatingExecutor) Probe(_ context.Context) (types.SandboxCapabilityProbe, error) {
	return types.SandboxCapabilityProbe{}, nil
}

func (m *mutatingExecutor) Execute(_ context.Context, _ types.SandboxExecSpec) (types.SandboxExecResult, error) {
	m.calls++
	if m.calls == 1 {
		return types.SandboxExecResult{ExitCode: 0, Stdout: "run"}, nil
	}
	return types.SandboxExecResult{ExitCode: 0, Stdout: "stream"}, nil
}

func TestCanonicalizeResultIsIdempotentAcrossBackends(t *testing.T) {
	in := types.SandboxExecResult{
		ExitCode:       2,
		ViolationCodes: []string{" Egress_Denied ", "egress_denied", "Capability_Mismatch"},
	}
	once := CanonicalizeExecResult(in)
	twice := CanonicalizeExecResult(once)
	if len(once.ViolationCodes) != 2 {
		t.Fatalf("canonicalized codes = %#v, want 2 entries", once.ViolationCodes)
	}
	if once.ViolationCodes[0] != "capability_mismatch" || once.ViolationCodes[1] != "egress_denied" {
		t.Fatalf("canonicalized codes = %#v, want sorted lowercase", once.ViolationCodes)
	}
	if !execResultsEquivalent(once, twice) {
		t.Fatalf("canonicalization is not idempotent: once=%+v twice=%+v", once, twice)
	}
}
