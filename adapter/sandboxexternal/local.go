package sandboxexternal

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

// LocalExecutor runs commands via the OS without isolation. It exists for offline
// conformance and local development only; it is NOT a production isolation
// backend. Real isolation is provided by the cloud adapters or the canonical
// nsjail/bwrap/oci/windows-job backends.
type LocalExecutor struct {
	baseExecutor
}

// NewLocalExecutor builds an offline-runnable executor.
func NewLocalExecutor(cfg Config) (*LocalExecutor, error) {
	caps := []string{
		runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture,
		runtimeconfig.SecuritySandboxCapabilitySessionPerCall,
		runtimeconfig.SecuritySandboxCapabilityNetworkOff,
	}
	modes := []string{runtimeconfig.SecuritySandboxSessionModePerCall, runtimeconfig.SecuritySandboxSessionModePerSession}
	return &LocalExecutor{baseExecutor: newBase(cfg.Name, cfg.CanonicalBackend, caps, modes)}, nil
}

// Execute runs the command directly on the host OS for offline verification.
func (e *LocalExecutor) Execute(ctx context.Context, spec types.SandboxExecSpec) (types.SandboxExecResult, error) {
	if ctx.Err() != nil {
		return types.SandboxExecResult{TimedOut: true}, &ExecError{Reason: ReasonTimeout, Err: ctx.Err()}
	}
	norm, err := types.NormalizeSandboxExecSpec(spec)
	if err != nil {
		return types.SandboxExecResult{LaunchFailed: true}, &ExecError{Reason: ReasonLaunchFailed, Err: err}
	}

	runCtx := ctx
	if norm.ExecTimeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, norm.ExecTimeout)
		defer cancel()
	}

	cmd := exec.CommandContext(runCtx, norm.Command, norm.Args...)
	if norm.Workdir != "" {
		cmd.Dir = norm.Workdir
	}
	if len(norm.Env) > 0 {
		env := os.Environ()
		for k, v := range norm.Env {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	res := types.SandboxExecResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if runErr != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			res.TimedOut = true
			return res, &ExecError{Reason: ReasonTimeout, Err: runErr}
		}
		var exitErr *exec.ExitError
		if os.IsTimeout(runErr) {
			res.TimedOut = true
			return res, &ExecError{Reason: ReasonTimeout, Err: runErr}
		}
		if errors.As(runErr, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
			return res, nil
		}
		res.LaunchFailed = true
		return res, &ExecError{Reason: ReasonLaunchFailed, Err: runErr}
	}
	res.ExitCode = 0
	return res, nil
}
