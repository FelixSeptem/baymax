package sandboxexternal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

// HTTPClient is the minimal interface used by cloud adapters. It is injectable so
// tests can supply a fake without real network access.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// CloudExecutor is the shared implementation for cloud sandbox backends
// (e2b, open-sandbox, anthropic-sandbox-runtime). It translates ExecSpec into a
// normalized exec request against the backend's local-or-cloud endpoint. Live
// calls require BaseURL + APIKey; when unconfigured or unreachable it reports
// backend_unavailable so hosts can fall back per DefaultAction.
type CloudExecutor struct {
	baseExecutor
	name    string
	client  HTTPClient
	baseURL string
	apiKey  string
	runPath string
}

func newCloud(cfg Config, caps, modes []string, runPath string) *CloudExecutor {
	var client HTTPClient
	if cfg.BaseURL != "" {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &CloudExecutor{
		baseExecutor: newBase(cfg.Name, cfg.CanonicalBackend, caps, modes),
		name:         cfg.Name,
		client:       client,
		baseURL:      strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:       cfg.APIKey,
		runPath:      runPath,
	}
}

// Execute performs a real translated HTTP call when configured. It never returns
// a silent success for transport failures; those surface as backend_unavailable.
func (e *CloudExecutor) Execute(ctx context.Context, spec types.SandboxExecSpec) (types.SandboxExecResult, error) {
	if e.client == nil || e.baseURL == "" {
		return types.SandboxExecResult{LaunchFailed: true},
			&ExecError{Reason: ReasonBackendUnavailable, Err: fmt.Errorf("backend %q not configured (set BaseURL/APIKey)", e.name)}
	}
	norm, err := types.NormalizeSandboxExecSpec(spec)
	if err != nil {
		return types.SandboxExecResult{LaunchFailed: true}, &ExecError{Reason: ReasonLaunchFailed, Err: err}
	}

	payload, err := json.Marshal(norm)
	if err != nil {
		return types.SandboxExecResult{LaunchFailed: true}, &ExecError{Reason: ReasonLaunchFailed, Err: err}
	}

	url := e.baseURL + e.runPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return types.SandboxExecResult{LaunchFailed: true}, &ExecError{Reason: ReasonLaunchFailed, Err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return types.SandboxExecResult{LaunchFailed: true}, &ExecError{Reason: ReasonBackendUnavailable, Err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 500 {
		return types.SandboxExecResult{LaunchFailed: true},
			&ExecError{Reason: ReasonBackendUnavailable, Err: fmt.Errorf("backend %q returned HTTP %d", e.name, resp.StatusCode)}
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.SandboxExecResult{LaunchFailed: true}, &ExecError{Reason: ReasonLaunchFailed, Err: err}
	}

	var out types.SandboxExecResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return types.SandboxExecResult{LaunchFailed: true}, &ExecError{Reason: ReasonLaunchFailed, Err: err}
	}
	out = CanonicalizeResult(out)
	return out, nil
}

func egressCaps() []string {
	return []string{
		runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture,
		runtimeconfig.SecuritySandboxCapabilitySessionPerCall,
		runtimeconfig.SecuritySandboxCapabilityNetworkEgressAllow,
	}
}

func perCallPerSessionModes() []string {
	return []string{
		runtimeconfig.SecuritySandboxSessionModePerCall,
		runtimeconfig.SecuritySandboxSessionModePerSession,
	}
}
