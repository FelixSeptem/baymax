// Command sandbox-external-backends demonstrates how mainstream external
// sandbox runtimes (e2b, open-sandbox, anthropic-sandbox-runtime) plug into the
// existing SandboxExecutor contract without changing the default posture.
//
// It is fully offline: the local backend executes a real command, the cloud
// adapters report backend_unavailable because no endpoint is configured, and the
// conformance matrix validates the canonical backend mapping.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/FelixSeptem/baymax/adapter/sandboxexternal"
	"github.com/FelixSeptem/baymax/core/types"
	sandboxconformance "github.com/FelixSeptem/baymax/integration/sandboxconformance"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func main() {
	fmt.Println("== sandbox external backends onboarding ==")
	fmt.Printf("registered backends: %s\n", strings.Join(sandboxexternal.RegisteredBackends(), ", "))

	// 1) Default posture must stay unchanged: external backend disabled,
	//    observe mode, DefaultAction=host.
	defaults := runtimeconfig.DefaultConfig()
	if _, _, ok := runtimeconfig.ResolveSandboxExternalBackend(defaults.Security.Sandbox); ok {
		fail("default external backend should be disabled")
	}
	fmt.Printf(
		"DEFAULT enabled=%v mode=%s default_action=%s external=%q\n",
		defaults.Security.Sandbox.Enabled,
		defaults.Security.Sandbox.Mode,
		defaults.Security.Sandbox.Policy.DefaultAction,
		defaults.Security.Sandbox.Executor.ExternalBackend,
	)

	// 2) Opt in to e2b: it must map onto the canonical oci_runtime profile.
	cfg := runtimeconfig.DefaultConfig()
	cfg.Security.Sandbox.Executor.Backend = runtimeconfig.SecuritySandboxBackendOCIRuntime
	cfg.Security.Sandbox.Executor.ExternalBackend = runtimeconfig.SecuritySandboxExternalBackendE2B
	if err := runtimeconfig.Validate(cfg); err != nil {
		fail("config validation: " + err.Error())
	}
	external, canonical, ok := runtimeconfig.ResolveSandboxExternalBackend(cfg.Security.Sandbox)
	fmt.Printf("MAPPING external=%q canonical=%q resolved=%v\n", external, canonical, ok)

	// A mismatched canonical backend must fail fast.
	bad := runtimeconfig.DefaultConfig()
	bad.Security.Sandbox.Executor.Backend = runtimeconfig.SecuritySandboxBackendWindowsJob
	bad.Security.Sandbox.Executor.ExternalBackend = runtimeconfig.SecuritySandboxExternalBackendE2B
	if err := runtimeconfig.Validate(bad); err == nil {
		fail("mismatched canonical backend should be rejected")
	} else {
		fmt.Printf("MISMATCH_REJECTED %v\n", err)
	}

	// 3) Conformance matrix for external backends (offline).
	if err := sandboxconformance.ValidateExternalBackendMatrix(sandboxconformance.ExternalBackendMatrix); err != nil {
		fail("external backend matrix: " + err.Error())
	}
	fmt.Printf("CONFORMANCE=pass scenarios=%d\n", len(sandboxconformance.ExternalBackendMatrix))

	// 4) Local onboarding backend: actually executes a command on this host.
	//    It is not an isolation backend; it exists so wiring can be verified
	//    without credentials.
	local, err := sandboxexternal.NewExecutor(sandboxexternal.Config{
		Name:             sandboxexternal.BackendLocal,
		CanonicalBackend: cfg.Security.Sandbox.Executor.Backend,
	})
	if err != nil {
		fail("build local executor: " + err.Error())
	}
	cmd, args := echoCommand("hello from sandbox external onboarding")
	res, err := local.Execute(context.Background(), types.SandboxExecSpec{Command: cmd, Args: args})
	if err != nil {
		fail("local execute: " + err.Error())
	}
	fmt.Printf("LOCAL_EXEC exit=%d stdout=%q\n", res.ExitCode, strings.TrimSpace(res.Stdout))

	// 5) Cloud adapter without an endpoint: must report backend_unavailable
	//    instead of failing the process or pretending to succeed.
	cloud, err := sandboxexternal.NewExecutor(sandboxexternal.Config{
		Name:             sandboxexternal.BackendE2B,
		CanonicalBackend: runtimeconfig.SecuritySandboxBackendOCIRuntime,
	})
	if err != nil {
		fail("build e2b executor: " + err.Error())
	}
	_, execErr := cloud.Execute(context.Background(), types.SandboxExecSpec{Command: "echo", Args: []string{"hi"}})
	var sandboxErr *sandboxexternal.ExecError
	if !errors.As(execErr, &sandboxErr) || sandboxErr.Reason != sandboxexternal.ReasonBackendUnavailable {
		fail(fmt.Sprintf("cloud adapter should report %s, got %v", sandboxexternal.ReasonBackendUnavailable, execErr))
	}
	fmt.Printf("CLOUD_EXEC reason=%s\n", sandboxErr.Reason)

	fmt.Println("RESULT=ok")
}

func echoCommand(msg string) (string, []string) {
	if runtime.GOOS == "windows" {
		// Quote the message so PowerShell emits one line instead of splitting
		// the words into separate output records.
		return "powershell", []string{"-NoProfile", "-Command", "Write-Output '" + msg + "'"}
	}
	return "echo", []string{msg}
}

func fail(msg string) {
	fmt.Fprintf(os.Stderr, "RESULT=fail %s\n", msg)
	os.Exit(1)
}
