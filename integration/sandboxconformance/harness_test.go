package sandboxconformance

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type fixtureExecutor struct {
	probe  types.SandboxCapabilityProbe
	result types.SandboxExecResult

	mu   sync.Mutex
	seen []types.SandboxExecSpec
}

func (f *fixtureExecutor) Probe(ctx context.Context) (types.SandboxCapabilityProbe, error) {
	_ = ctx
	return f.probe, nil
}

func (f *fixtureExecutor) Execute(ctx context.Context, spec types.SandboxExecSpec) (types.SandboxExecResult, error) {
	_ = ctx
	f.mu.Lock()
	f.seen = append(f.seen, spec)
	f.mu.Unlock()
	return f.result, nil
}

func (f *fixtureExecutor) LastSpec() (types.SandboxExecSpec, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.seen) == 0 {
		return types.SandboxExecSpec{}, false
	}
	return f.seen[len(f.seen)-1], true
}

func TestSandboxExecutorConformanceMinimumMatrixCoverage(t *testing.T) {
	if err := ValidateMinimumMatrix(MinimumMatrix); err != nil {
		t.Fatalf("minimum matrix invalid: %v", err)
	}
}

func TestSandboxExecutorConformanceCanonicalExecSpecInteroperability(t *testing.T) {
	rawSpec := types.SandboxExecSpec{
		NamespaceTool: "  local+exec ",
		Command:       " /bin/echo ",
		Args:          []string{"hello", "sandbox"},
		Env: map[string]string{
			"KEEP":  " value ",
			"EMPTY": " ",
			"SPARE": " 1 ",
			"":      "x",
		},
		Workdir: t.TempDir(),
		Mounts: []types.SandboxMount{
			{Source: "tmp-b", Target: "/b", ReadOnly: true},
			{Source: "tmp-a", Target: "/a", ReadOnly: false},
		},
		Network: types.SandboxNetworkPolicy{
			Mode:            "network_off",
			EgressAllowlist: []string{"127.0.0.1"},
		},
		ResourceLimits: types.SandboxResourceLimits{
			CPUMilli:    300,
			MemoryBytes: 128 * 1024 * 1024,
			PIDLimit:    64,
		},
		SessionMode:   types.SandboxSessionModePerCall,
		LaunchTimeout: 100 * time.Millisecond,
		ExecTimeout:   500 * time.Millisecond,
	}
	normalizedSpec, err := types.NormalizeSandboxExecSpec(rawSpec)
	if err != nil {
		t.Fatalf("normalize exec spec: %v", err)
	}

	var canonicalBaseline *types.SandboxExecResult
	for i := range MinimumMatrix {
		scenario := MinimumMatrix[i]
		violations := []string{types.SandboxViolationMount, types.SandboxViolationTimeout, types.SandboxViolationMount}
		if i%2 == 1 {
			violations = []string{types.SandboxViolationTimeout, types.SandboxViolationMount}
		}
		executor := &fixtureExecutor{
			probe: types.SandboxCapabilityProbe{
				Backend:        scenario.Backend,
				Capabilities:   append([]string(nil), scenario.ProbeCapabilities...),
				SupportedModes: append([]string(nil), scenario.SupportedModes...),
			},
			result: types.SandboxExecResult{
				ExitCode:       0,
				Stdout:         "sandbox-ok",
				ViolationCodes: append([]string(nil), violations...),
				ResourceUsage: types.SandboxResourceUsage{
					CPUTimeMs:       11,
					MemoryPeakBytes: 8192,
				},
			},
		}
		probe, err := executor.Probe(context.Background())
		if err != nil {
			t.Fatalf("probe failed for %s: %v", scenario.ID, err)
		}
		if err := EvaluateCapabilityNegotiation(probe, scenario.RequiredCapabilities, scenario.SessionMode); err != nil {
			t.Fatalf("capability negotiation failed for %s: %v", scenario.ID, err)
		}
		result, err := executor.Execute(context.Background(), normalizedSpec)
		if err != nil {
			t.Fatalf("execute failed for %s: %v", scenario.ID, err)
		}
		canonical := CanonicalizeExecResult(result)
		if canonicalBaseline == nil {
			canonicalBaseline = &canonical
		} else if !reflect.DeepEqual(*canonicalBaseline, canonical) {
			t.Fatalf("canonical result drift for %s: got=%#v want=%#v", scenario.ID, canonical, *canonicalBaseline)
		}

		seenSpec, ok := executor.LastSpec()
		if !ok {
			t.Fatalf("missing captured exec spec for %s", scenario.ID)
		}
		if seenSpec.Command != "/bin/echo" {
			t.Fatalf("command should be normalized for %s: %#v", scenario.ID, seenSpec.Command)
		}
		expectedEnv := map[string]string{"KEEP": "value", "SPARE": "1"}
		if !reflect.DeepEqual(seenSpec.Env, expectedEnv) {
			t.Fatalf("env normalization drift for %s: got=%#v want=%#v", scenario.ID, seenSpec.Env, expectedEnv)
		}
		expectedMounts := []types.SandboxMount{
			{Source: "tmp-a", Target: "/a", ReadOnly: false},
			{Source: "tmp-b", Target: "/b", ReadOnly: true},
		}
		if !reflect.DeepEqual(seenSpec.Mounts, expectedMounts) {
			t.Fatalf("mount normalization drift for %s: got=%#v want=%#v", scenario.ID, seenSpec.Mounts, expectedMounts)
		}
		if seenSpec.SessionMode != types.SandboxSessionModePerCall {
			t.Fatalf("session mode drift for %s: %#v", scenario.ID, seenSpec.SessionMode)
		}
	}
}

func TestSandboxExecutorConformanceCapabilityNegotiationDriftDeterministic(t *testing.T) {
	missingCapabilityProbe := types.SandboxCapabilityProbe{
		Backend: runtimeconfig.SecuritySandboxBackendLinuxNSJail,
		Capabilities: []string{
			runtimeconfig.SecuritySandboxCapabilitySessionPerCall,
		},
		SupportedModes: []string{
			runtimeconfig.SecuritySandboxSessionModePerCall,
		},
	}
	required := []string{
		runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture,
		runtimeconfig.SecuritySandboxCapabilitySessionPerCall,
	}
	err1 := EvaluateCapabilityNegotiation(missingCapabilityProbe, required, types.SandboxSessionModePerCall)
	err2 := EvaluateCapabilityNegotiation(missingCapabilityProbe, required, types.SandboxSessionModePerCall)
	expectConformanceErrorReason(t, err1, ReasonCapabilityMismatch)
	expectConformanceErrorReason(t, err2, ReasonCapabilityMismatch)
	if err1.Error() != err2.Error() {
		t.Fatalf("capability mismatch classification must be deterministic: err1=%q err2=%q", err1, err2)
	}

	sessionUnsupportedProbe := types.SandboxCapabilityProbe{
		Backend: runtimeconfig.SecuritySandboxBackendWindowsJob,
		Capabilities: []string{
			runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture,
			runtimeconfig.SecuritySandboxCapabilitySessionPerCall,
		},
		SupportedModes: []string{
			runtimeconfig.SecuritySandboxSessionModePerCall,
		},
	}
	sessionErr := EvaluateCapabilityNegotiation(
		sessionUnsupportedProbe,
		[]string{
			runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture,
			runtimeconfig.SecuritySandboxCapabilitySessionPerCall,
		},
		types.SandboxSessionModePerSession,
	)
	expectConformanceErrorReason(t, sessionErr, ReasonSessionModeUnsupported)
}

func TestSandboxExecutorConformanceSessionLifecycleDeterministic(t *testing.T) {
	h := NewSessionLifecycleHarness()
	perSession1 := h.Acquire(types.SandboxSessionModePerSession, "run-1")
	perSession2 := h.Acquire(types.SandboxSessionModePerSession, "run-1")
	if perSession1 != perSession2 {
		t.Fatalf("per_session lifecycle should reuse token: first=%s second=%s", perSession1, perSession2)
	}

	perCall1 := h.Acquire(types.SandboxSessionModePerCall, "run-1")
	perCall2 := h.Acquire(types.SandboxSessionModePerCall, "run-1")
	if perCall1 == perCall2 {
		t.Fatalf("per_call lifecycle should allocate isolated token: first=%s second=%s", perCall1, perCall2)
	}

	if !h.Close("run-1") {
		t.Fatal("first close should report active session")
	}
	if h.Close("run-1") {
		t.Fatal("repeated close should be idempotent and report inactive")
	}
	perSession3 := h.Acquire(types.SandboxSessionModePerSession, "run-1")
	if perSession3 == perSession1 {
		t.Fatalf("session should be recreated after close: previous=%s current=%s", perSession1, perSession3)
	}
}

func TestSandboxExecutorConformanceLaunchFallbackSemantics(t *testing.T) {
	allow1, err := ResolveLaunchFailureFallback(runtimeconfig.SecuritySandboxFallbackAllowAndRecord)
	if err != nil {
		t.Fatalf("allow_and_record fallback should succeed: %v", err)
	}
	allow2, err := ResolveLaunchFailureFallback(runtimeconfig.SecuritySandboxFallbackAllowAndRecord)
	if err != nil {
		t.Fatalf("allow_and_record fallback should stay deterministic: %v", err)
	}
	if !reflect.DeepEqual(allow1, allow2) {
		t.Fatalf("allow_and_record fallback must be deterministic: first=%#v second=%#v", allow1, allow2)
	}
	if allow1.Decision != runtimeconfig.SecuritySandboxActionHost || allow1.ReasonCode != ReasonFallbackAllowAndRecord {
		t.Fatalf("allow_and_record fallback mismatch: %#v", allow1)
	}

	deny, err := ResolveLaunchFailureFallback(runtimeconfig.SecuritySandboxFallbackDeny)
	if err != nil {
		t.Fatalf("deny fallback should succeed: %v", err)
	}
	if deny.Decision != runtimeconfig.SecuritySandboxActionDeny || deny.ReasonCode != ReasonLaunchFailed {
		t.Fatalf("deny fallback mismatch: %#v", deny)
	}

	if _, err := ResolveLaunchFailureFallback("unknown"); err == nil || !strings.Contains(strings.ToLower(err.Error()), "unsupported fallback action") {
		t.Fatalf("unexpected invalid fallback result: %v", err)
	}
}

func expectConformanceErrorReason(t *testing.T, err error, reason string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected conformance error reason=%s", reason)
	}
	var ce *ConformanceError
	if !strings.Contains(err.Error(), reason) {
		t.Fatalf("reason mismatch want=%s got=%v", reason, err)
	}
	if !errors.As(err, &ce) {
		t.Fatalf("expected conformance error type, got %T (%v)", err, err)
	}
	if ce.ReasonCode != reason {
		t.Fatalf("reason code mismatch want=%s got=%s", reason, ce.ReasonCode)
	}
}
