package sandboxconformance

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

// Frozen failure reasons for external sandbox runtimes. Values match
// adapter/sandboxexternal so the conformance layer and the adapters cannot
// drift apart.
const (
	ReasonTimeout            = "sandbox.timeout"
	ReasonOOM                = "sandbox.oom"
	ReasonEgressDenied       = "sandbox.egress_denied"
	ReasonBackendUnavailable = "sandbox.backend_unavailable"
)

// ExternalFailureKind enumerates the frozen failure taxonomy every external
// runtime must classify identically.
type ExternalFailureKind string

const (
	ExternalFailureLaunchFailed       ExternalFailureKind = "launch_failed"
	ExternalFailureCapabilityMismatch ExternalFailureKind = "capability_mismatch"
	ExternalFailureTimeout            ExternalFailureKind = "timeout"
	ExternalFailureOOM                ExternalFailureKind = "oom"
	ExternalFailureEgressDenied       ExternalFailureKind = "egress_denied"
	ExternalFailureBackendUnavailable ExternalFailureKind = "backend_unavailable"
)

// externalFailureReasons is the single source of truth for kind -> reason code.
var externalFailureReasons = map[ExternalFailureKind]string{
	ExternalFailureLaunchFailed:       ReasonLaunchFailed,
	ExternalFailureCapabilityMismatch: ReasonCapabilityMismatch,
	ExternalFailureTimeout:            ReasonTimeout,
	ExternalFailureOOM:                ReasonOOM,
	ExternalFailureEgressDenied:       ReasonEgressDenied,
	ExternalFailureBackendUnavailable: ReasonBackendUnavailable,
}

// ExternalFailureReason returns the frozen reason code for a failure kind.
func ExternalFailureReason(kind ExternalFailureKind) (string, bool) {
	reason, ok := externalFailureReasons[kind]
	return reason, ok
}

// ExternalFailureFixture binds a failure kind to an external runtime so the
// conformance gate can prove every runtime classifies every failure the same way.
type ExternalFailureFixture struct {
	ID       string
	Runtime  string
	Kind     ExternalFailureKind
	Expected string
}

// externalRuntimeCanonical mirrors runtime/config so fixtures stay aligned with
// the registration mapping.
var externalRuntimeCanonical = map[string]string{
	runtimeconfig.SecuritySandboxExternalBackendE2B:                     runtimeconfig.SecuritySandboxBackendOCIRuntime,
	runtimeconfig.SecuritySandboxExternalBackendOpenSandbox:             runtimeconfig.SecuritySandboxBackendOCIRuntime,
	runtimeconfig.SecuritySandboxExternalBackendAnthropicSandboxRuntime: runtimeconfig.SecuritySandboxBackendOCIRuntime,
}

// ExternalRuntimes returns the sorted external runtime selectors covered by
// conformance fixtures.
func ExternalRuntimes() []string {
	out := make([]string, 0, len(externalRuntimeCanonical))
	for runtimeName := range externalRuntimeCanonical {
		out = append(out, runtimeName)
	}
	sort.Strings(out)
	return out
}

// ExternalFailureFixtures builds the full runtime × kind matrix.
func ExternalFailureFixtures() []ExternalFailureFixture {
	kinds := []ExternalFailureKind{
		ExternalFailureLaunchFailed,
		ExternalFailureCapabilityMismatch,
		ExternalFailureTimeout,
		ExternalFailureOOM,
		ExternalFailureEgressDenied,
		ExternalFailureBackendUnavailable,
	}
	runtimes := ExternalRuntimes()
	out := make([]ExternalFailureFixture, 0, len(runtimes)*len(kinds))
	for _, runtimeName := range runtimes {
		for _, kind := range kinds {
			out = append(out, ExternalFailureFixture{
				ID:       runtimeName + "-" + string(kind),
				Runtime:  runtimeName,
				Kind:     kind,
				Expected: externalFailureReasons[kind],
			})
		}
	}
	return out
}

// ValidateExternalFailureFixtures requires full runtime × kind coverage with
// frozen, non-empty reason codes.
func ValidateExternalFailureFixtures(fixtures []ExternalFailureFixture) error {
	if len(fixtures) == 0 {
		return fmt.Errorf("empty external failure fixture matrix")
	}
	seen := map[string]struct{}{}
	covered := map[string]map[ExternalFailureKind]struct{}{}
	for i := range fixtures {
		fixture := fixtures[i]
		id := strings.TrimSpace(fixture.ID)
		if id == "" {
			return fmt.Errorf("fixture[%d] id must not be empty", i)
		}
		if _, dup := seen[id]; dup {
			return fmt.Errorf("duplicate fixture id: %s", id)
		}
		seen[id] = struct{}{}
		if _, ok := externalRuntimeCanonical[fixture.Runtime]; !ok {
			return fmt.Errorf("fixture[%s] uses unregistered external runtime %q", id, fixture.Runtime)
		}
		want, ok := externalFailureReasons[fixture.Kind]
		if !ok {
			return fmt.Errorf("fixture[%s] uses unknown failure kind %q", id, fixture.Kind)
		}
		if fixture.Expected != want {
			return fmt.Errorf("fixture[%s] expects %q, want frozen reason %q", id, fixture.Expected, want)
		}
		if covered[fixture.Runtime] == nil {
			covered[fixture.Runtime] = map[ExternalFailureKind]struct{}{}
		}
		covered[fixture.Runtime][fixture.Kind] = struct{}{}
	}
	for runtimeName := range externalRuntimeCanonical {
		for kind := range externalFailureReasons {
			if _, ok := covered[runtimeName][kind]; !ok {
				return fmt.Errorf("missing fixture coverage: runtime=%s kind=%s", runtimeName, kind)
			}
		}
	}
	return nil
}

// ScriptedExecutor replays a fixed outcome. It is the offline stand-in for a
// live remote runtime: no network, no credentials, deterministic results.
type ScriptedExecutor struct {
	ProbeResult types.SandboxCapabilityProbe
	Result      types.SandboxExecResult
	Err         error
	Calls       int
}

func (s *ScriptedExecutor) Probe(_ context.Context) (types.SandboxCapabilityProbe, error) {
	return s.ProbeResult, nil
}

func (s *ScriptedExecutor) Execute(_ context.Context, _ types.SandboxExecSpec) (types.SandboxExecResult, error) {
	s.Calls++
	return CanonicalizeExecResult(s.Result), s.Err
}

// ExecRunner is the minimal execute surface needed for parity checks.
type ExecRunner interface {
	Execute(ctx context.Context, spec types.SandboxExecSpec) (types.SandboxExecResult, error)
}

// ExternalRunStreamParity executes the same spec twice through an executor and
// requires the canonicalized outcomes to be equivalent. It is the
// conformance-level proxy for Run/Stream parity: both paths must observe the
// same result shape and the same failure reason.
func ExternalRunStreamParity(exec ExecRunner, spec types.SandboxExecSpec) error {
	if exec == nil {
		return fmt.Errorf("parity check requires an executor")
	}
	runRes, runErr := exec.Execute(context.Background(), spec)
	streamRes, streamErr := exec.Execute(context.Background(), spec)
	if !execResultsEquivalent(runRes, streamRes) {
		return fmt.Errorf("run/stream result drift: run=%+v stream=%+v", runRes, streamRes)
	}
	if !errorsEquivalent(runErr, streamErr) {
		return fmt.Errorf("run/stream error drift: run=%v stream=%v", runErr, streamErr)
	}
	return nil
}

func execResultsEquivalent(a, b types.SandboxExecResult) bool {
	if a.ExitCode != b.ExitCode ||
		a.Stdout != b.Stdout ||
		a.Stderr != b.Stderr ||
		a.TimedOut != b.TimedOut ||
		a.OOMKilled != b.OOMKilled ||
		a.LaunchFailed != b.LaunchFailed {
		return false
	}
	if len(a.ViolationCodes) != len(b.ViolationCodes) {
		return false
	}
	for i := range a.ViolationCodes {
		if a.ViolationCodes[i] != b.ViolationCodes[i] {
			return false
		}
	}
	return true
}

func errorsEquivalent(a, b error) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Error() == b.Error()
}
