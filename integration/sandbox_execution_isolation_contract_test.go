package integration

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/observability/event"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
	"github.com/FelixSeptem/baymax/tool/local"
)

type sandboxIntegrationModel struct {
	generateFn func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error)
	streamFn   func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error
}

func (m *sandboxIntegrationModel) Generate(
	ctx context.Context,
	req types.ModelRequest,
) (types.ModelResponse, error) {
	if m != nil && m.generateFn != nil {
		return m.generateFn(ctx, req)
	}
	return types.ModelResponse{}, nil
}

func (m *sandboxIntegrationModel) Stream(
	ctx context.Context,
	req types.ModelRequest,
	onEvent func(types.ModelEvent) error,
) error {
	if m != nil && m.streamFn != nil {
		return m.streamFn(ctx, req, onEvent)
	}
	return nil
}

type sandboxIntegrationAdapterTool struct{}

func (sandboxIntegrationAdapterTool) Name() string {
	return "exec"
}

func (sandboxIntegrationAdapterTool) Description() string {
	return "sandbox integration adapter"
}

func (sandboxIntegrationAdapterTool) JSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
	}
}

func (sandboxIntegrationAdapterTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
	_ = ctx
	_ = args
	return types.ToolResult{Content: "host"}, nil
}

func (sandboxIntegrationAdapterTool) BuildSandboxExecSpec(
	ctx context.Context,
	args map[string]any,
) (types.SandboxExecSpec, error) {
	_ = ctx
	_ = args
	return types.SandboxExecSpec{
		Command: "cmd.exe",
		Args:    []string{"/c", "echo integration"},
	}, nil
}

func (sandboxIntegrationAdapterTool) HandleSandboxExecResult(
	ctx context.Context,
	result types.SandboxExecResult,
) (types.ToolResult, error) {
	_ = ctx
	_ = result
	return types.ToolResult{Content: "sandbox"}, nil
}

type sandboxIntegrationExecutor struct {
	probeFn   func(ctx context.Context) (types.SandboxCapabilityProbe, error)
	executeFn func(ctx context.Context, spec types.SandboxExecSpec) (types.SandboxExecResult, error)
}

func (f *sandboxIntegrationExecutor) Probe(ctx context.Context) (types.SandboxCapabilityProbe, error) {
	if f != nil && f.probeFn != nil {
		return f.probeFn(ctx)
	}
	return types.SandboxCapabilityProbe{}, nil
}

func (f *sandboxIntegrationExecutor) Execute(
	ctx context.Context,
	spec types.SandboxExecSpec,
) (types.SandboxExecResult, error) {
	if f != nil && f.executeFn != nil {
		return f.executeFn(ctx, spec)
	}
	return types.SandboxExecResult{}, nil
}

type sandboxCapabilityDeliverySnapshot struct {
	PolicyKind                     string
	Decision                       string
	ReasonCode                     string
	AlertDispatchStatus            string
	AlertDeliveryMode              string
	AlertRetryCount                int
	AlertCircuitState              string
	SandboxMode                    string
	SandboxBackend                 string
	SandboxProfile                 string
	SandboxSessionMode             string
	SandboxRequiredCapabilities    []string
	SandboxDecision                string
	SandboxReasonCode              string
	SandboxCapabilityMismatchTotal int
}

func sandboxCapabilityDeliverySnapshotFromRunRecord(rec runtimediag.RunRecord) sandboxCapabilityDeliverySnapshot {
	return sandboxCapabilityDeliverySnapshot{
		PolicyKind:                     strings.TrimSpace(rec.PolicyKind),
		Decision:                       strings.TrimSpace(rec.Decision),
		ReasonCode:                     strings.TrimSpace(rec.ReasonCode),
		AlertDispatchStatus:            strings.TrimSpace(rec.AlertDispatchStatus),
		AlertDeliveryMode:              strings.TrimSpace(rec.AlertDeliveryMode),
		AlertRetryCount:                rec.AlertRetryCount,
		AlertCircuitState:              strings.TrimSpace(rec.AlertCircuitState),
		SandboxMode:                    strings.TrimSpace(rec.SandboxMode),
		SandboxBackend:                 strings.TrimSpace(rec.SandboxBackend),
		SandboxProfile:                 strings.TrimSpace(rec.SandboxProfile),
		SandboxSessionMode:             strings.TrimSpace(rec.SandboxSessionMode),
		SandboxRequiredCapabilities:    append([]string(nil), rec.SandboxRequiredCapabilities...),
		SandboxDecision:                strings.TrimSpace(rec.SandboxDecision),
		SandboxReasonCode:              strings.TrimSpace(rec.SandboxReasonCode),
		SandboxCapabilityMismatchTotal: rec.SandboxCapabilityMismatchTotal,
	}
}

func TestSandboxExecutionIsolationContractRunStreamSecurityDeliveryParity(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-a51-sandbox-integration.yaml")
	cfg := `
security:
  tool_governance:
    enabled: false
  sandbox:
    enabled: true
    mode: enforce
    required: true
    policy:
      default_action: deny
      profile: default
      fallback_action: deny
  security_event:
    enabled: true
    alert:
      trigger_policy: deny_only
      sink: callback
      callback:
        require_registered: true
    delivery:
      mode: sync
      timeout: 200ms
      retry:
        max_attempts: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A51_INTEGRATION_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	reg := local.NewRegistry()
	if _, err := reg.Register(sandboxIntegrationAdapterTool{}); err != nil {
		t.Fatalf("register sandbox adapter tool: %v", err)
	}

	runModel := &sandboxIntegrationModel{
		generateFn: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			_ = ctx
			_ = req
			return types.ModelResponse{
				ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.exec"}},
			}, nil
		},
	}
	streamModel := &sandboxIntegrationModel{
		streamFn: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			_ = ctx
			_ = req
			return onEvent(types.ModelEvent{
				Type: types.ModelEventTypeToolCall,
				ToolCall: &types.ToolCall{
					CallID: "c1",
					Name:   "local.exec",
				},
			})
		},
	}

	recorder := event.NewRuntimeRecorder(mgr)
	var callbackMu sync.Mutex
	runEvents := make([]types.SecurityEvent, 0, 1)
	streamEvents := make([]types.SecurityEvent, 0, 1)

	runEngine := runner.New(
		runModel,
		runner.WithRuntimeManager(mgr),
		runner.WithLocalRegistry(reg),
		runner.WithSecurityAlertCallback(func(ctx context.Context, event types.SecurityEvent) error {
			_ = ctx
			callbackMu.Lock()
			runEvents = append(runEvents, event)
			callbackMu.Unlock()
			return nil
		}),
	)
	streamEngine := runner.New(
		streamModel,
		runner.WithRuntimeManager(mgr),
		runner.WithLocalRegistry(reg),
		runner.WithSecurityAlertCallback(func(ctx context.Context, event types.SecurityEvent) error {
			_ = ctx
			callbackMu.Lock()
			streamEvents = append(streamEvents, event)
			callbackMu.Unlock()
			return nil
		}),
	)

	runID := "run-a51-integration-capability-run"
	streamID := "run-a51-integration-capability-stream"
	runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{
		RunID: runID,
		Input: "policy-deny-run",
	}, recorder)
	streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{
		RunID: streamID,
		Input: "policy-deny-stream",
	}, recorder)
	if runErr == nil || runRes.Error == nil || runRes.Error.Class != types.ErrSecurity {
		t.Fatalf("expected run security deny, got err=%v result=%#v", runErr, runRes.Error)
	}
	if streamErr == nil || streamRes.Error == nil || streamRes.Error.Class != types.ErrSecurity {
		t.Fatalf("expected stream security deny, got err=%v result=%#v", streamErr, streamRes.Error)
	}
	if got := runRes.Error.Details["reason_code"]; got != "sandbox.policy_deny" {
		t.Fatalf("run reason_code=%#v, want sandbox.policy_deny", got)
	}
	if got := streamRes.Error.Details["reason_code"]; got != "sandbox.policy_deny" {
		t.Fatalf("stream reason_code=%#v, want sandbox.policy_deny", got)
	}
	if got := runRes.Error.Details["alert_dispatch_status"]; got != "succeeded" {
		t.Fatalf("run alert_dispatch_status=%#v, want succeeded", got)
	}
	if got := streamRes.Error.Details["alert_dispatch_status"]; got != "succeeded" {
		t.Fatalf("stream alert_dispatch_status=%#v, want succeeded", got)
	}

	callbackMu.Lock()
	runEventCount := len(runEvents)
	streamEventCount := len(streamEvents)
	var runEvent, streamEvent types.SecurityEvent
	if runEventCount > 0 {
		runEvent = runEvents[0]
	}
	if streamEventCount > 0 {
		streamEvent = streamEvents[0]
	}
	callbackMu.Unlock()
	if runEventCount != 1 || streamEventCount != 1 {
		t.Fatalf("callback count mismatch run=%d stream=%d", runEventCount, streamEventCount)
	}
	if runEvent.PolicyKind != "sandbox" || runEvent.Decision != "deny" || runEvent.ReasonCode != "sandbox.policy_deny" {
		t.Fatalf("run callback taxonomy mismatch: %#v", runEvent)
	}
	if streamEvent.PolicyKind != "sandbox" || streamEvent.Decision != "deny" || streamEvent.ReasonCode != "sandbox.policy_deny" {
		t.Fatalf("stream callback taxonomy mismatch: %#v", streamEvent)
	}

	runRecord := findRunRecord(t, mgr.RecentRuns(20), runID)
	streamRecord := findRunRecord(t, mgr.RecentRuns(20), streamID)
	runSnapshot := sandboxCapabilityDeliverySnapshotFromRunRecord(runRecord)
	streamSnapshot := sandboxCapabilityDeliverySnapshotFromRunRecord(streamRecord)
	if !reflect.DeepEqual(runSnapshot, streamSnapshot) {
		t.Fatalf("sandbox run/stream parity mismatch run=%#v stream=%#v", runSnapshot, streamSnapshot)
	}
	if runSnapshot.PolicyKind != "sandbox" ||
		runSnapshot.Decision != "deny" ||
		runSnapshot.ReasonCode != "sandbox.policy_deny" ||
		runSnapshot.AlertDispatchStatus != "succeeded" ||
		runSnapshot.AlertDeliveryMode != runtimeconfig.SecurityEventDeliveryModeSync ||
		runSnapshot.SandboxReasonCode != "" ||
		runSnapshot.SandboxCapabilityMismatchTotal != 0 {
		t.Fatalf("sandbox snapshot mismatch: %#v", runSnapshot)
	}
}

func TestSandboxExecutionIsolationContractCapabilityNegotiationDeny(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-a51-sandbox-capability-negotiation.yaml")
	cfg := `
security:
  sandbox:
    enabled: true
    mode: enforce
    required: true
    policy:
      default_action: host
      by_tool:
        local+exec: sandbox
      profile: default
      fallback_action: deny
    executor:
      required_capabilities:
        - network_off
  security_event:
    enabled: true
    alert:
      trigger_policy: deny_only
      sink: callback
      callback:
        require_registered: true
    delivery:
      mode: sync
      timeout: 200ms
      retry:
        max_attempts: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A51_INTEGRATION_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	mgr.SetSandboxExecutor(&sandboxIntegrationExecutor{
		probeFn: func(ctx context.Context) (types.SandboxCapabilityProbe, error) {
			_ = ctx
			return types.SandboxCapabilityProbe{
				Backend:        runtimeconfig.SecuritySandboxBackendWindowsJob,
				Capabilities:   []string{runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture},
				SupportedModes: []string{runtimeconfig.SecuritySandboxSessionModePerCall},
			}, nil
		},
		executeFn: func(ctx context.Context, spec types.SandboxExecSpec) (types.SandboxExecResult, error) {
			_ = ctx
			_ = spec
			return types.SandboxExecResult{ExitCode: 0}, nil
		},
	})

	reg := local.NewRegistry()
	if _, err := reg.Register(sandboxIntegrationAdapterTool{}); err != nil {
		t.Fatalf("register sandbox adapter tool: %v", err)
	}

	model := &sandboxIntegrationModel{
		generateFn: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			_ = ctx
			_ = req
			return types.ModelResponse{
				ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.exec"}},
			}, nil
		},
	}

	runEvents := make([]types.SecurityEvent, 0, 1)
	engine := runner.New(
		model,
		runner.WithRuntimeManager(mgr),
		runner.WithLocalRegistry(reg),
		runner.WithSecurityAlertCallback(func(ctx context.Context, event types.SecurityEvent) error {
			_ = ctx
			runEvents = append(runEvents, event)
			return nil
		}),
	)

	runID := "run-a51-integration-capability-negotiation"
	recorder := event.NewRuntimeRecorder(mgr)
	res, runErr := engine.Run(context.Background(), types.RunRequest{
		RunID: runID,
		Input: "capability-negotiation",
	}, recorder)
	if runErr == nil || res.Error == nil || res.Error.Class != types.ErrSecurity {
		t.Fatalf("expected security deny, got err=%v result=%#v", runErr, res.Error)
	}
	if got := res.Error.Details["reason_code"]; got != "sandbox.capability_mismatch" {
		t.Fatalf("reason_code=%#v, want sandbox.capability_mismatch", got)
	}
	if got := res.Error.Details["alert_dispatch_status"]; got != "succeeded" {
		t.Fatalf("alert_dispatch_status=%#v, want succeeded", got)
	}
	if len(runEvents) != 1 {
		t.Fatalf("callback events len=%d, want 1", len(runEvents))
	}
	if runEvents[0].PolicyKind != "sandbox" ||
		runEvents[0].Decision != "deny" ||
		runEvents[0].ReasonCode != "sandbox.capability_mismatch" {
		t.Fatalf("callback taxonomy mismatch: %#v", runEvents[0])
	}

	record := findRunRecord(t, mgr.RecentRuns(20), runID)
	snapshot := sandboxCapabilityDeliverySnapshotFromRunRecord(record)
	if snapshot.PolicyKind != "sandbox" ||
		snapshot.Decision != "deny" ||
		snapshot.ReasonCode != "sandbox.capability_mismatch" ||
		snapshot.AlertDispatchStatus != "succeeded" ||
		snapshot.AlertDeliveryMode != runtimeconfig.SecurityEventDeliveryModeSync ||
		snapshot.SandboxMode != runtimeconfig.SecuritySandboxModeEnforce ||
		snapshot.SandboxBackend != runtimeconfig.SecuritySandboxBackendWindowsJob ||
		snapshot.SandboxReasonCode != "sandbox.capability_mismatch" ||
		snapshot.SandboxCapabilityMismatchTotal != 1 {
		t.Fatalf("capability negotiation snapshot mismatch: %#v", snapshot)
	}
}

func TestSandboxExecutionIsolationContractBackendCompatibilityMatrixSmoke(t *testing.T) {
	tests := []struct {
		name    string
		backend string
	}{
		{
			name:    "linux-nsjail",
			backend: runtimeconfig.SecuritySandboxBackendLinuxNSJail,
		},
		{
			name:    "windows-job",
			backend: runtimeconfig.SecuritySandboxBackendWindowsJob,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cfgPath := filepath.Join(t.TempDir(), "runtime-a51-sandbox-backend-"+tc.name+".yaml")
			cfg := strings.Join([]string{
				"security:",
				"  sandbox:",
				"    enabled: true",
				"    mode: enforce",
				"    required: true",
				"    policy:",
				"      default_action: host",
				"      by_tool:",
				"        local+exec: sandbox",
				"      profile: default",
				"      fallback_action: deny",
				"    executor:",
				"      backend: " + tc.backend,
				"      required_capabilities:",
				"        - stdout_stderr_capture",
				"",
			}, "\n")
			if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
				t.Fatalf("write config: %v", err)
			}
			mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
				FilePath:  cfgPath,
				EnvPrefix: "BAYMAX_A51_INTEGRATION_TEST",
			})
			if err != nil {
				t.Fatalf("new runtime manager: %v", err)
			}
			t.Cleanup(func() { _ = mgr.Close() })
			mgr.SetSandboxExecutor(&sandboxIntegrationExecutor{
				probeFn: func(ctx context.Context) (types.SandboxCapabilityProbe, error) {
					_ = ctx
					return types.SandboxCapabilityProbe{
						Backend:        tc.backend,
						Capabilities:   []string{runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture},
						SupportedModes: []string{runtimeconfig.SecuritySandboxSessionModePerCall},
					}, nil
				},
				executeFn: func(ctx context.Context, spec types.SandboxExecSpec) (types.SandboxExecResult, error) {
					_ = ctx
					_ = spec
					return types.SandboxExecResult{
						ExitCode: 0,
						Stdout:   "sandbox-ok",
					}, nil
				},
			})

			reg := local.NewRegistry()
			if _, err := reg.Register(sandboxIntegrationAdapterTool{}); err != nil {
				t.Fatalf("register sandbox adapter tool: %v", err)
			}

			var turns int
			model := &sandboxIntegrationModel{
				generateFn: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
					_ = ctx
					turns++
					if turns == 1 {
						return types.ModelResponse{
							ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.exec"}},
						}, nil
					}
					if len(req.ToolResult) != 1 || req.ToolResult[0].Result.Content != "sandbox" {
						t.Fatalf("sandbox tool result feedback mismatch: %#v", req.ToolResult)
					}
					return types.ModelResponse{FinalAnswer: "ok"}, nil
				},
			}

			runID := "run-a51-backend-smoke-" + strings.ReplaceAll(tc.name, "-", "_")
			recorder := event.NewRuntimeRecorder(mgr)
			engine := runner.New(model, runner.WithRuntimeManager(mgr), runner.WithLocalRegistry(reg))
			res, runErr := engine.Run(context.Background(), types.RunRequest{
				RunID: runID,
				Input: "backend-matrix-smoke",
			}, recorder)
			if runErr != nil {
				t.Fatalf("run should succeed: %v", runErr)
			}
			if res.Error != nil || res.FinalAnswer != "ok" {
				t.Fatalf("unexpected run result: %#v", res)
			}

			record := findRunRecord(t, mgr.RecentRuns(20), runID)
			if record.SandboxMode != runtimeconfig.SecuritySandboxModeEnforce ||
				record.SandboxBackend != tc.backend ||
				record.SandboxProfile != runtimeconfig.SecuritySandboxDefaultProfile ||
				record.SandboxSessionMode != runtimeconfig.SecuritySandboxSessionModePerCall ||
				record.SandboxDecision != runtimeconfig.SecuritySandboxActionSandbox ||
				!reflect.DeepEqual(record.SandboxRequiredCapabilities, []string{runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture}) ||
				record.SandboxLaunchFailedTotal != 0 ||
				record.SandboxTimeoutTotal != 0 ||
				record.SandboxCapabilityMismatchTotal != 0 {
				t.Fatalf("backend compatibility smoke mismatch for %s: %#v", tc.backend, record)
			}
		})
	}
}
