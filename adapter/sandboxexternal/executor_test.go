package sandboxexternal

import (
	"context"
	"errors"
	"io"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

// runnableCommand returns a command that prints msg, cross-platform.
func runnableCommand(msg string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "powershell", []string{"-NoProfile", "-Command", "Write-Output " + msg}
	}
	return "echo", []string{msg}
}

// slowCommand returns a command that blocks for several seconds, cross-platform.
func slowCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "ping", []string{"-n", "6", "127.0.0.1"}
	}
	return "sleep", []string{"5"}
}

func TestRegistryRejectsUnknownBackend(t *testing.T) {
	_, err := NewExecutor(Config{Name: "does-not-exist", CanonicalBackend: runtimeconfig.SecuritySandboxBackendOCIRuntime})
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
}

func TestNewExecutorRejectsBadCanonicalProfile(t *testing.T) {
	_, err := NewExecutor(Config{Name: BackendLocal, CanonicalBackend: "made_up_profile"})
	if err == nil {
		t.Fatal("expected error for unsupported canonical profile")
	}
}

func TestLocalExecutorRunsCommand(t *testing.T) {
	ex, err := NewLocalExecutor(Config{Name: BackendLocal, CanonicalBackend: runtimeconfig.SecuritySandboxBackendOCIRuntime})
	if err != nil {
		t.Fatalf("NewLocalExecutor: %v", err)
	}
	cmd, args := runnableCommand("hello-baymax")
	spec := types.SandboxExecSpec{
		Command: cmd,
		Args:    args,
	}
	res, err := ex.Execute(context.Background(), spec)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0", res.ExitCode)
	}
	if !strings.Contains(res.Stdout, "hello-baymax") {
		t.Fatalf("stdout = %q, want contains hello-baymax", res.Stdout)
	}
}

func TestLocalExecutorTimeout(t *testing.T) {
	ex, _ := NewLocalExecutor(Config{Name: BackendLocal, CanonicalBackend: runtimeconfig.SecuritySandboxBackendOCIRuntime})
	cmd, args := slowCommand()
	spec := types.SandboxExecSpec{
		Command:     cmd,
		Args:        args,
		ExecTimeout: 200 * time.Millisecond,
	}
	start := time.Now()
	res, err := ex.Execute(context.Background(), spec)
	elapsed := time.Since(start)
	if !res.TimedOut {
		t.Fatalf("TimedOut = false, want true (err=%v)", err)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("timeout took too long: %v", elapsed)
	}
}

func TestCloudAdapterUnconfiguredReportsBackendUnavailable(t *testing.T) {
	for _, name := range []string{BackendE2B, BackendOpenSandbox, BackendAnthropicSandboxRT} {
		ex, err := NewExecutor(Config{Name: name, CanonicalBackend: runtimeconfig.SecuritySandboxBackendOCIRuntime})
		if err != nil {
			t.Fatalf("%s: NewExecutor: %v", name, err)
		}
		_, err = ex.Execute(context.Background(), types.SandboxExecSpec{Command: "echo", Args: []string{"hi"}})
		var ee *ExecError
		if !errors.As(err, &ee) {
			t.Fatalf("%s: expected *ExecError, got %T (%v)", name, err, err)
		}
		if ee.Reason != ReasonBackendUnavailable {
			t.Fatalf("%s: reason = %q, want %q", name, ee.Reason, ReasonBackendUnavailable)
		}
	}
}

func TestCloudAdapterProbeExposesCapabilities(t *testing.T) {
	ex, _ := NewE2BExecutor(Config{Name: BackendE2B, CanonicalBackend: runtimeconfig.SecuritySandboxBackendOCIRuntime})
	probe, err := ex.Probe(context.Background())
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if !probe.Supports(runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture) {
		t.Fatalf("probe missing stdout_stderr_capture capability")
	}
	if !probe.SupportsSessionMode(types.SandboxSessionModePerCall) {
		t.Fatalf("probe missing per_call session mode")
	}
}

// fakeClient is an injectable HTTPClient returning a canned normalized result.
type fakeClient struct {
	status int
	body   string
}

func (f *fakeClient) Do(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: f.status,
		Body:       io.NopCloser(strings.NewReader(f.body)),
		Header:     make(http.Header),
	}, nil
}

func TestCloudAdapterExecuteParsesNormalizedResponse(t *testing.T) {
	ex := newCloud(Config{Name: BackendE2B, CanonicalBackend: runtimeconfig.SecuritySandboxBackendOCIRuntime, BaseURL: "http://localhost:9999"},
		egressCaps(), perCallPerSessionModes(), "/v1/exec")
	ex.client = &fakeClient{status: 200, body: `{"exit_code":0,"stdout":"ok","stderr":"","violation_codes":["SANDBOX.TIMEOUT","sandbox.timeout"]}`}

	res, err := ex.Execute(context.Background(), types.SandboxExecSpec{Command: "echo", Args: []string{"ok"}})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.ExitCode != 0 || res.Stdout != "ok" {
		t.Fatalf("unexpected result: %+v", res)
	}
	// violation codes must be canonicalized (deduped + lowercased).
	if len(res.ViolationCodes) != 1 || res.ViolationCodes[0] != "sandbox.timeout" {
		t.Fatalf("violation codes not canonicalized: %v", res.ViolationCodes)
	}
}

func TestCloudAdapterServerErrorReportsBackendUnavailable(t *testing.T) {
	ex := newCloud(Config{Name: BackendE2B, CanonicalBackend: runtimeconfig.SecuritySandboxBackendOCIRuntime, BaseURL: "http://localhost:9999"},
		egressCaps(), perCallPerSessionModes(), "/v1/exec")
	ex.client = &fakeClient{status: 503, body: ""}

	_, err := ex.Execute(context.Background(), types.SandboxExecSpec{Command: "echo", Args: []string{"x"}})
	var ee *ExecError
	if !errors.As(err, &ee) || ee.Reason != ReasonBackendUnavailable {
		t.Fatalf("expected backend_unavailable, got %v", err)
	}
}

func TestRegisteredBackendsIncludesMainstreamRuntimes(t *testing.T) {
	got := RegisteredBackends()
	want := map[string]bool{BackendE2B: true, BackendOpenSandbox: true, BackendAnthropicSandboxRT: true, BackendLocal: true}
	for _, b := range got {
		delete(want, b)
	}
	if len(want) != 0 {
		t.Fatalf("registered backends missing: %v (got %v)", want, got)
	}
}
