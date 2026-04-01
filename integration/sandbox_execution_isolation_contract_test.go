package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

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

func (m *sandboxIntegrationModel) ProviderName() string {
	return "sandbox-integration"
}

func (m *sandboxIntegrationModel) DiscoverCapabilities(
	ctx context.Context,
	req types.ModelRequest,
) (types.ProviderCapabilities, error) {
	_ = ctx
	return types.ProviderCapabilities{
		Provider: "sandbox-integration",
		Model:    req.Model,
		Support: map[types.ModelCapability]types.CapabilitySupport{
			types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
			types.ModelCapabilityToolCall:  types.CapabilitySupportSupported,
		},
		Source:    "integration-test",
		CheckedAt: time.Now(),
	}, nil
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
	SandboxFallbackUsed            bool
	SandboxFallbackReason          string
	SandboxTimeoutTotal            int
	SandboxLaunchFailedTotal       int
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
		SandboxFallbackUsed:            rec.SandboxFallbackUsed,
		SandboxFallbackReason:          strings.TrimSpace(rec.SandboxFallbackReason),
		SandboxTimeoutTotal:            rec.SandboxTimeoutTotal,
		SandboxLaunchFailedTotal:       rec.SandboxLaunchFailedTotal,
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

func TestSandboxExecutionIsolationContractReactActionResolutionRunStreamParity(t *testing.T) {
	t.Run("host_multi_iteration", func(t *testing.T) {
		runModel := &scriptedReactModel{
			generateSteps: []scriptedGenerateStep{
				{
					response: types.ModelResponse{
						ToolCalls: []types.ToolCall{{CallID: "host-call-1", Name: "local.exec"}},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						return assertSandboxToolFeedback(req, "host-call-1", "host", "")
					},
					response: types.ModelResponse{
						ToolCalls: []types.ToolCall{{CallID: "host-call-2", Name: "local.exec"}},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						return assertSandboxToolFeedback(req, "host-call-2", "host", "")
					},
					response: types.ModelResponse{FinalAnswer: "ok"},
				},
			},
		}
		streamModel := &scriptedReactModel{
			streamSteps: []scriptedStreamStep{
				{
					events: []types.ModelEvent{
						{
							Type: types.ModelEventTypeToolCall,
							ToolCall: &types.ToolCall{
								CallID: "host-call-1",
								Name:   "local.exec",
							},
						},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						return assertSandboxToolFeedback(req, "host-call-1", "host", "")
					},
					events: []types.ModelEvent{
						{
							Type: types.ModelEventTypeToolCall,
							ToolCall: &types.ToolCall{
								CallID: "host-call-2",
								Name:   "local.exec",
							},
						},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						return assertSandboxToolFeedback(req, "host-call-2", "host", "")
					},
					events: []types.ModelEvent{
						{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
			},
		}
		cfg := strings.Join([]string{
			"runtime:",
			"  react:",
			"    enabled: true",
			"    stream_tool_dispatch_enabled: true",
			"security:",
			"  sandbox:",
			"    enabled: true",
			"    mode: enforce",
			"    required: false",
			"    policy:",
			"      default_action: host",
			"      profile: default",
			"      fallback_action: deny",
			"",
		}, "\n")
		runEngine, streamEngine, mgr := newSandboxReactRunStreamHarness(t, cfg, runModel, streamModel, nil)
		runCollector := &eventCollector{}
		streamCollector := &eventCollector{}
		runHandler := newRunStreamDispatcherHandler(mgr, runCollector)
		streamHandler := newRunStreamDispatcherHandler(mgr, streamCollector)
		policy := defaultSandboxReactPolicy()

		runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{
			RunID:  "run-a56-sandbox-react-host-run",
			Input:  "host-multi",
			Policy: &policy,
		}, runHandler)
		streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{
			RunID:  "run-a56-sandbox-react-host-stream",
			Input:  "host-multi",
			Policy: &policy,
		}, streamHandler)
		if runErr != nil || streamErr != nil {
			t.Fatalf("host multi-iteration should succeed, runErr=%v streamErr=%v", runErr, streamErr)
		}
		if runRes.Error != nil || streamRes.Error != nil {
			t.Fatalf("unexpected classified errors run=%#v stream=%#v", runRes.Error, streamRes.Error)
		}
		if runRes.FinalAnswer != "ok" || streamRes.FinalAnswer != "ok" {
			t.Fatalf("host final answer mismatch run=%q stream=%q", runRes.FinalAnswer, streamRes.FinalAnswer)
		}
		runPayload := lastRunFinishedPayload(t, runCollector.events)
		streamPayload := lastRunFinishedPayload(t, streamCollector.events)
		assertReactParityPayload(t, runPayload, streamPayload, runtimeconfig.RuntimeReactTerminationCompleted)
		if runPayload["react_tool_call_total"] != 2 || streamPayload["react_tool_call_total"] != 2 {
			t.Fatalf("host react_tool_call_total mismatch run=%#v stream=%#v", runPayload["react_tool_call_total"], streamPayload["react_tool_call_total"])
		}

		runRecord := findRunRecord(t, mgr.RecentRuns(20), "run-a56-sandbox-react-host-run")
		streamRecord := findRunRecord(t, mgr.RecentRuns(20), "run-a56-sandbox-react-host-stream")
		if !reflect.DeepEqual(
			sandboxCapabilityDeliverySnapshotFromRunRecord(runRecord),
			sandboxCapabilityDeliverySnapshotFromRunRecord(streamRecord),
		) {
			t.Fatalf("host run/stream sandbox snapshot mismatch run=%#v stream=%#v", runRecord, streamRecord)
		}
	})

	t.Run("sandbox_multi_iteration", func(t *testing.T) {
		runModel := &scriptedReactModel{
			generateSteps: []scriptedGenerateStep{
				{
					response: types.ModelResponse{
						ToolCalls: []types.ToolCall{{CallID: "sandbox-call-1", Name: "local.exec"}},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						return assertSandboxToolFeedback(req, "sandbox-call-1", "sandbox", "")
					},
					response: types.ModelResponse{
						ToolCalls: []types.ToolCall{{CallID: "sandbox-call-2", Name: "local.exec"}},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						return assertSandboxToolFeedback(req, "sandbox-call-2", "sandbox", "")
					},
					response: types.ModelResponse{FinalAnswer: "ok"},
				},
			},
		}
		streamModel := &scriptedReactModel{
			streamSteps: []scriptedStreamStep{
				{
					events: []types.ModelEvent{
						{
							Type: types.ModelEventTypeToolCall,
							ToolCall: &types.ToolCall{
								CallID: "sandbox-call-1",
								Name:   "local.exec",
							},
						},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						return assertSandboxToolFeedback(req, "sandbox-call-1", "sandbox", "")
					},
					events: []types.ModelEvent{
						{
							Type: types.ModelEventTypeToolCall,
							ToolCall: &types.ToolCall{
								CallID: "sandbox-call-2",
								Name:   "local.exec",
							},
						},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						return assertSandboxToolFeedback(req, "sandbox-call-2", "sandbox", "")
					},
					events: []types.ModelEvent{
						{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
			},
		}
		cfg := strings.Join([]string{
			"runtime:",
			"  react:",
			"    enabled: true",
			"    stream_tool_dispatch_enabled: true",
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
			"",
		}, "\n")
		executor := &sandboxIntegrationExecutor{
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
		}
		runEngine, streamEngine, mgr := newSandboxReactRunStreamHarness(t, cfg, runModel, streamModel, executor)
		runCollector := &eventCollector{}
		streamCollector := &eventCollector{}
		runHandler := newRunStreamDispatcherHandler(mgr, runCollector)
		streamHandler := newRunStreamDispatcherHandler(mgr, streamCollector)
		policy := defaultSandboxReactPolicy()

		runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{
			RunID:  "run-a56-sandbox-react-sandbox-run",
			Input:  "sandbox-multi",
			Policy: &policy,
		}, runHandler)
		streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{
			RunID:  "run-a56-sandbox-react-sandbox-stream",
			Input:  "sandbox-multi",
			Policy: &policy,
		}, streamHandler)
		if runErr != nil || streamErr != nil {
			t.Fatalf("sandbox multi-iteration should succeed, runErr=%v streamErr=%v", runErr, streamErr)
		}
		if runRes.Error != nil || streamRes.Error != nil {
			t.Fatalf("unexpected classified errors run=%#v stream=%#v", runRes.Error, streamRes.Error)
		}
		if runRes.FinalAnswer != "ok" || streamRes.FinalAnswer != "ok" {
			t.Fatalf("sandbox final answer mismatch run=%q stream=%q", runRes.FinalAnswer, streamRes.FinalAnswer)
		}
		runPayload := lastRunFinishedPayload(t, runCollector.events)
		streamPayload := lastRunFinishedPayload(t, streamCollector.events)
		assertReactParityPayload(t, runPayload, streamPayload, runtimeconfig.RuntimeReactTerminationCompleted)
		if runPayload["react_tool_call_total"] != 2 || streamPayload["react_tool_call_total"] != 2 {
			t.Fatalf("sandbox react_tool_call_total mismatch run=%#v stream=%#v", runPayload["react_tool_call_total"], streamPayload["react_tool_call_total"])
		}

		runRecord := findRunRecord(t, mgr.RecentRuns(20), "run-a56-sandbox-react-sandbox-run")
		streamRecord := findRunRecord(t, mgr.RecentRuns(20), "run-a56-sandbox-react-sandbox-stream")
		runSnapshot := sandboxCapabilityDeliverySnapshotFromRunRecord(runRecord)
		streamSnapshot := sandboxCapabilityDeliverySnapshotFromRunRecord(streamRecord)
		if !reflect.DeepEqual(runSnapshot, streamSnapshot) {
			t.Fatalf("sandbox run/stream snapshot mismatch run=%#v stream=%#v", runSnapshot, streamSnapshot)
		}
		if runSnapshot.SandboxDecision != runtimeconfig.SecuritySandboxActionSandbox ||
			runSnapshot.SandboxReasonCode != "" ||
			runSnapshot.SandboxLaunchFailedTotal != 0 ||
			runSnapshot.SandboxCapabilityMismatchTotal != 0 {
			t.Fatalf("sandbox snapshot mismatch: %#v", runSnapshot)
		}
	})

	t.Run("deny_parity", func(t *testing.T) {
		runModel := &scriptedReactModel{
			generateSteps: []scriptedGenerateStep{
				{
					response: types.ModelResponse{
						ToolCalls: []types.ToolCall{{CallID: "deny-call-1", Name: "local.exec"}},
					},
				},
			},
		}
		streamModel := &scriptedReactModel{
			streamSteps: []scriptedStreamStep{
				{
					events: []types.ModelEvent{
						{
							Type: types.ModelEventTypeToolCall,
							ToolCall: &types.ToolCall{
								CallID: "deny-call-1",
								Name:   "local.exec",
							},
						},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
			},
		}
		cfg := strings.Join([]string{
			"runtime:",
			"  react:",
			"    enabled: true",
			"    stream_tool_dispatch_enabled: true",
			"security:",
			"  sandbox:",
			"    enabled: true",
			"    mode: enforce",
			"    required: true",
			"    policy:",
			"      default_action: deny",
			"      profile: default",
			"      fallback_action: deny",
			"",
		}, "\n")
		runEngine, streamEngine, _ := newSandboxReactRunStreamHarness(t, cfg, runModel, streamModel, nil)
		runCollector := &eventCollector{}
		streamCollector := &eventCollector{}
		runHandler := newRunStreamDispatcherHandler(nil, runCollector)
		streamHandler := newRunStreamDispatcherHandler(nil, streamCollector)
		policy := defaultSandboxReactPolicy()

		runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{
			RunID:  "run-a56-sandbox-react-deny-run",
			Input:  "deny",
			Policy: &policy,
		}, runHandler)
		streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{
			RunID:  "run-a56-sandbox-react-deny-stream",
			Input:  "deny",
			Policy: &policy,
		}, streamHandler)
		if runErr == nil || streamErr == nil {
			t.Fatalf("deny parity should fail, runErr=%v streamErr=%v", runErr, streamErr)
		}
		if runRes.Error == nil || streamRes.Error == nil || runRes.Error.Class != types.ErrSecurity || streamRes.Error.Class != types.ErrSecurity {
			t.Fatalf("deny classified error mismatch run=%#v stream=%#v", runRes.Error, streamRes.Error)
		}
		if runRes.Error.Details["reason_code"] != "sandbox.policy_deny" ||
			streamRes.Error.Details["reason_code"] != "sandbox.policy_deny" {
			t.Fatalf("deny reason mismatch run=%#v stream=%#v", runRes.Error.Details["reason_code"], streamRes.Error.Details["reason_code"])
		}
		runPayload := lastRunFinishedPayload(t, runCollector.events)
		streamPayload := lastRunFinishedPayload(t, streamCollector.events)
		assertReactParityPayload(t, runPayload, streamPayload, runtimeconfig.RuntimeReactTerminationToolDispatchFailed)
		if runPayload["reason_code"] != "sandbox.policy_deny" || streamPayload["reason_code"] != "sandbox.policy_deny" {
			t.Fatalf("deny payload reason mismatch run=%#v stream=%#v", runPayload["reason_code"], streamPayload["reason_code"])
		}
	})
}

func TestSandboxExecutionIsolationContractReactEgressParity(t *testing.T) {
	t.Run("deny", func(t *testing.T) {
		runModel := &scriptedReactModel{
			generateSteps: []scriptedGenerateStep{
				{
					response: types.ModelResponse{
						ToolCalls: []types.ToolCall{{
							CallID: "egress-deny-call-1",
							Name:   "local.exec",
							Args:   map[string]any{"url": "https://blocked.example/v1"},
						}},
					},
				},
			},
		}
		streamModel := &scriptedReactModel{
			streamSteps: []scriptedStreamStep{
				{
					events: []types.ModelEvent{
						{
							Type: types.ModelEventTypeToolCall,
							ToolCall: &types.ToolCall{
								CallID: "egress-deny-call-1",
								Name:   "local.exec",
								Args:   map[string]any{"url": "https://blocked.example/v1"},
							},
						},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
			},
		}
		cfg := strings.Join([]string{
			"runtime:",
			"  react:",
			"    enabled: true",
			"    stream_tool_dispatch_enabled: true",
			"security:",
			"  sandbox:",
			"    enabled: true",
			"    mode: enforce",
			"    required: true",
			"    policy:",
			"      default_action: host",
			"      profile: default",
			"      fallback_action: deny",
			"    egress:",
			"      enabled: true",
			"      default_action: deny",
			"      on_violation: deny",
			"",
		}, "\n")
		runEngine, streamEngine, _ := newSandboxReactRunStreamHarness(t, cfg, runModel, streamModel, nil)
		runCollector := &eventCollector{}
		streamCollector := &eventCollector{}
		runHandler := newRunStreamDispatcherHandler(nil, runCollector)
		streamHandler := newRunStreamDispatcherHandler(nil, streamCollector)
		policy := defaultSandboxReactPolicy()

		runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{
			RunID:  "run-a57-sandbox-react-egress-deny-run",
			Input:  "egress-deny",
			Policy: &policy,
		}, runHandler)
		streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{
			RunID:  "run-a57-sandbox-react-egress-deny-stream",
			Input:  "egress-deny",
			Policy: &policy,
		}, streamHandler)
		if runErr == nil || streamErr == nil {
			t.Fatalf("egress deny parity should fail, runErr=%v streamErr=%v", runErr, streamErr)
		}
		if runRes.Error == nil || streamRes.Error == nil || runRes.Error.Class != types.ErrSecurity || streamRes.Error.Class != types.ErrSecurity {
			t.Fatalf("egress deny classified error mismatch run=%#v stream=%#v", runRes.Error, streamRes.Error)
		}
		if runRes.Error.Details["reason_code"] != "sandbox.egress_deny" ||
			streamRes.Error.Details["reason_code"] != "sandbox.egress_deny" {
			t.Fatalf("egress deny reason mismatch run=%#v stream=%#v", runRes.Error.Details["reason_code"], streamRes.Error.Details["reason_code"])
		}
		runPayload := lastRunFinishedPayload(t, runCollector.events)
		streamPayload := lastRunFinishedPayload(t, streamCollector.events)
		assertReactParityPayload(t, runPayload, streamPayload, runtimeconfig.RuntimeReactTerminationToolDispatchFailed)
		if runPayload["reason_code"] != "sandbox.egress_deny" || streamPayload["reason_code"] != "sandbox.egress_deny" {
			t.Fatalf("egress deny payload reason mismatch run=%#v stream=%#v", runPayload["reason_code"], streamPayload["reason_code"])
		}
	})

	t.Run("allow", func(t *testing.T) {
		runModel := &scriptedReactModel{
			generateSteps: []scriptedGenerateStep{
				{
					response: types.ModelResponse{
						ToolCalls: []types.ToolCall{{
							CallID: "egress-allow-call-1",
							Name:   "local.exec",
							Args:   map[string]any{"url": "https://api.example.com/v1"},
						}},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						return assertSandboxToolFeedback(req, "egress-allow-call-1", "host", "")
					},
					response: types.ModelResponse{FinalAnswer: "ok"},
				},
			},
		}
		streamModel := &scriptedReactModel{
			streamSteps: []scriptedStreamStep{
				{
					events: []types.ModelEvent{
						{
							Type: types.ModelEventTypeToolCall,
							ToolCall: &types.ToolCall{
								CallID: "egress-allow-call-1",
								Name:   "local.exec",
								Args:   map[string]any{"url": "https://api.example.com/v1"},
							},
						},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						return assertSandboxToolFeedback(req, "egress-allow-call-1", "host", "")
					},
					events: []types.ModelEvent{
						{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
			},
		}
		cfg := strings.Join([]string{
			"runtime:",
			"  react:",
			"    enabled: true",
			"    stream_tool_dispatch_enabled: true",
			"security:",
			"  sandbox:",
			"    enabled: true",
			"    mode: enforce",
			"    required: false",
			"    policy:",
			"      default_action: host",
			"      profile: default",
			"      fallback_action: deny",
			"    egress:",
			"      enabled: true",
			"      default_action: deny",
			"      allowlist:",
			"        - api.example.com",
			"      on_violation: deny",
			"",
		}, "\n")
		runEngine, streamEngine, _ := newSandboxReactRunStreamHarness(t, cfg, runModel, streamModel, nil)
		runCollector := &eventCollector{}
		streamCollector := &eventCollector{}
		runHandler := newRunStreamDispatcherHandler(nil, runCollector)
		streamHandler := newRunStreamDispatcherHandler(nil, streamCollector)
		policy := defaultSandboxReactPolicy()

		runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{
			RunID:  "run-a57-sandbox-react-egress-allow-run",
			Input:  "egress-allow",
			Policy: &policy,
		}, runHandler)
		streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{
			RunID:  "run-a57-sandbox-react-egress-allow-stream",
			Input:  "egress-allow",
			Policy: &policy,
		}, streamHandler)
		if runErr != nil || streamErr != nil {
			t.Fatalf("egress allow parity should succeed, runErr=%v streamErr=%v", runErr, streamErr)
		}
		if runRes.Error != nil || streamRes.Error != nil {
			t.Fatalf("egress allow classified error mismatch run=%#v stream=%#v", runRes.Error, streamRes.Error)
		}
		runPayload := lastRunFinishedPayload(t, runCollector.events)
		streamPayload := lastRunFinishedPayload(t, streamCollector.events)
		assertReactParityPayload(t, runPayload, streamPayload, runtimeconfig.RuntimeReactTerminationCompleted)
	})

	t.Run("allow_and_record", func(t *testing.T) {
		runModel := &scriptedReactModel{
			generateSteps: []scriptedGenerateStep{
				{
					response: types.ModelResponse{
						ToolCalls: []types.ToolCall{{
							CallID: "egress-record-call-1",
							Name:   "local.exec",
							Args:   map[string]any{"url": "https://blocked.example/v1"},
						}},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						return assertSandboxEgressToolFeedback(req, "egress-record-call-1", "host", "sandbox.egress_allow_and_record")
					},
					response: types.ModelResponse{FinalAnswer: "ok"},
				},
			},
		}
		streamModel := &scriptedReactModel{
			streamSteps: []scriptedStreamStep{
				{
					events: []types.ModelEvent{
						{
							Type: types.ModelEventTypeToolCall,
							ToolCall: &types.ToolCall{
								CallID: "egress-record-call-1",
								Name:   "local.exec",
								Args:   map[string]any{"url": "https://blocked.example/v1"},
							},
						},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						return assertSandboxEgressToolFeedback(req, "egress-record-call-1", "host", "sandbox.egress_allow_and_record")
					},
					events: []types.ModelEvent{
						{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
			},
		}
		cfg := strings.Join([]string{
			"runtime:",
			"  react:",
			"    enabled: true",
			"    stream_tool_dispatch_enabled: true",
			"security:",
			"  sandbox:",
			"    enabled: true",
			"    mode: enforce",
			"    required: false",
			"    policy:",
			"      default_action: host",
			"      profile: default",
			"      fallback_action: deny",
			"    egress:",
			"      enabled: true",
			"      default_action: deny",
			"      on_violation: allow_and_record",
			"",
		}, "\n")
		runEngine, streamEngine, _ := newSandboxReactRunStreamHarness(t, cfg, runModel, streamModel, nil)
		runCollector := &eventCollector{}
		streamCollector := &eventCollector{}
		runHandler := newRunStreamDispatcherHandler(nil, runCollector)
		streamHandler := newRunStreamDispatcherHandler(nil, streamCollector)
		policy := defaultSandboxReactPolicy()

		runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{
			RunID:  "run-a57-sandbox-react-egress-record-run",
			Input:  "egress-record",
			Policy: &policy,
		}, runHandler)
		streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{
			RunID:  "run-a57-sandbox-react-egress-record-stream",
			Input:  "egress-record",
			Policy: &policy,
		}, streamHandler)
		if runErr != nil || streamErr != nil {
			t.Fatalf("egress allow_and_record parity should succeed, runErr=%v streamErr=%v", runErr, streamErr)
		}
		if runRes.Error != nil || streamRes.Error != nil {
			t.Fatalf("egress allow_and_record classified error mismatch run=%#v stream=%#v", runRes.Error, streamRes.Error)
		}
		runPayload := lastRunFinishedPayload(t, runCollector.events)
		streamPayload := lastRunFinishedPayload(t, streamCollector.events)
		assertReactParityPayload(t, runPayload, streamPayload, runtimeconfig.RuntimeReactTerminationCompleted)
		if runPayload["sandbox_reason_code"] != "sandbox.egress_allow_and_record" ||
			streamPayload["sandbox_reason_code"] != "sandbox.egress_allow_and_record" {
			t.Fatalf("egress allow_and_record payload mismatch run=%#v stream=%#v", runPayload, streamPayload)
		}
	})
}

func TestSandboxExecutionIsolationContractReactFallbackTaxonomyAndCountersParity(t *testing.T) {
	t.Run("allow_and_record", func(t *testing.T) {
		runModel := &scriptedReactModel{
			generateSteps: []scriptedGenerateStep{
				{
					response: types.ModelResponse{
						ToolCalls: []types.ToolCall{{CallID: "fallback-call-1", Name: "local.exec"}},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						return assertSandboxToolFeedback(req, "fallback-call-1", "host", "sandbox.fallback_allow_and_record")
					},
					response: types.ModelResponse{
						ToolCalls: []types.ToolCall{{CallID: "fallback-call-2", Name: "local.exec"}},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						return assertSandboxToolFeedback(req, "fallback-call-2", "host", "sandbox.fallback_allow_and_record")
					},
					response: types.ModelResponse{FinalAnswer: "ok"},
				},
			},
		}
		streamModel := &scriptedReactModel{
			streamSteps: []scriptedStreamStep{
				{
					events: []types.ModelEvent{
						{
							Type: types.ModelEventTypeToolCall,
							ToolCall: &types.ToolCall{
								CallID: "fallback-call-1",
								Name:   "local.exec",
							},
						},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						return assertSandboxToolFeedback(req, "fallback-call-1", "host", "sandbox.fallback_allow_and_record")
					},
					events: []types.ModelEvent{
						{
							Type: types.ModelEventTypeToolCall,
							ToolCall: &types.ToolCall{
								CallID: "fallback-call-2",
								Name:   "local.exec",
							},
						},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
				{
					assertRequest: func(req types.ModelRequest) error {
						return assertSandboxToolFeedback(req, "fallback-call-2", "host", "sandbox.fallback_allow_and_record")
					},
					events: []types.ModelEvent{
						{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
			},
		}
		cfg := strings.Join([]string{
			"runtime:",
			"  react:",
			"    enabled: true",
			"    stream_tool_dispatch_enabled: true",
			"security:",
			"  sandbox:",
			"    enabled: true",
			"    mode: enforce",
			"    required: false",
			"    policy:",
			"      default_action: host",
			"      by_tool:",
			"        local+exec: sandbox",
			"      profile: default",
			"      fallback_action: allow_and_record",
			"",
		}, "\n")
		executor := &sandboxIntegrationExecutor{
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
				return types.SandboxExecResult{}, fmt.Errorf("sandbox launch failed")
			},
		}
		runEngine, streamEngine, mgr := newSandboxReactRunStreamHarness(t, cfg, runModel, streamModel, executor)
		runCollector := &eventCollector{}
		streamCollector := &eventCollector{}
		runHandler := newRunStreamDispatcherHandler(mgr, runCollector)
		streamHandler := newRunStreamDispatcherHandler(mgr, streamCollector)
		policy := defaultSandboxReactPolicy()

		runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{
			RunID:  "run-a56-sandbox-react-fallback-allow-run",
			Input:  "fallback-allow",
			Policy: &policy,
		}, runHandler)
		streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{
			RunID:  "run-a56-sandbox-react-fallback-allow-stream",
			Input:  "fallback-allow",
			Policy: &policy,
		}, streamHandler)
		if runErr != nil || streamErr != nil {
			t.Fatalf("allow_and_record should succeed, runErr=%v streamErr=%v", runErr, streamErr)
		}
		if runRes.Error != nil || streamRes.Error != nil {
			t.Fatalf("allow_and_record classified errors run=%#v stream=%#v", runRes.Error, streamRes.Error)
		}
		runPayload := lastRunFinishedPayload(t, runCollector.events)
		streamPayload := lastRunFinishedPayload(t, streamCollector.events)
		assertReactParityPayload(t, runPayload, streamPayload, runtimeconfig.RuntimeReactTerminationCompleted)
		if runPayload["react_tool_call_total"] != 2 || streamPayload["react_tool_call_total"] != 2 {
			t.Fatalf("allow_and_record react_tool_call_total mismatch run=%#v stream=%#v", runPayload["react_tool_call_total"], streamPayload["react_tool_call_total"])
		}
		if runPayload["sandbox_fallback_used"] != true ||
			streamPayload["sandbox_fallback_used"] != true ||
			runPayload["sandbox_fallback_reason"] != "sandbox.fallback_allow_and_record" ||
			streamPayload["sandbox_fallback_reason"] != "sandbox.fallback_allow_and_record" ||
			runPayload["sandbox_reason_code"] != "sandbox.fallback_allow_and_record" ||
			streamPayload["sandbox_reason_code"] != "sandbox.fallback_allow_and_record" {
			t.Fatalf("allow_and_record fallback taxonomy mismatch run=%#v stream=%#v", runPayload, streamPayload)
		}

		runRecord := findRunRecord(t, mgr.RecentRuns(20), "run-a56-sandbox-react-fallback-allow-run")
		streamRecord := findRunRecord(t, mgr.RecentRuns(20), "run-a56-sandbox-react-fallback-allow-stream")
		runSnapshot := sandboxCapabilityDeliverySnapshotFromRunRecord(runRecord)
		streamSnapshot := sandboxCapabilityDeliverySnapshotFromRunRecord(streamRecord)
		if !reflect.DeepEqual(runSnapshot, streamSnapshot) {
			t.Fatalf("allow_and_record run/stream snapshot mismatch run=%#v stream=%#v", runSnapshot, streamSnapshot)
		}
		if !runSnapshot.SandboxFallbackUsed ||
			runSnapshot.SandboxFallbackReason != "sandbox.fallback_allow_and_record" ||
			runSnapshot.SandboxReasonCode != "sandbox.fallback_allow_and_record" ||
			runSnapshot.SandboxLaunchFailedTotal != 0 ||
			runSnapshot.SandboxCapabilityMismatchTotal != 0 {
			t.Fatalf("allow_and_record snapshot mismatch: %#v", runSnapshot)
		}
	})

	t.Run("deny", func(t *testing.T) {
		runModel := &scriptedReactModel{
			generateSteps: []scriptedGenerateStep{
				{
					response: types.ModelResponse{
						ToolCalls: []types.ToolCall{{CallID: "fallback-deny-call-1", Name: "local.exec"}},
					},
				},
			},
		}
		streamModel := &scriptedReactModel{
			streamSteps: []scriptedStreamStep{
				{
					events: []types.ModelEvent{
						{
							Type: types.ModelEventTypeToolCall,
							ToolCall: &types.ToolCall{
								CallID: "fallback-deny-call-1",
								Name:   "local.exec",
							},
						},
						{Type: types.ModelEventTypeResponseCompleted},
					},
				},
			},
		}
		cfg := strings.Join([]string{
			"runtime:",
			"  react:",
			"    enabled: true",
			"    stream_tool_dispatch_enabled: true",
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
			"",
		}, "\n")
		executor := &sandboxIntegrationExecutor{
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
				return types.SandboxExecResult{}, fmt.Errorf("sandbox launch failed")
			},
		}
		runEngine, streamEngine, mgr := newSandboxReactRunStreamHarness(t, cfg, runModel, streamModel, executor)
		runCollector := &eventCollector{}
		streamCollector := &eventCollector{}
		runHandler := newRunStreamDispatcherHandler(mgr, runCollector)
		streamHandler := newRunStreamDispatcherHandler(mgr, streamCollector)
		policy := defaultSandboxReactPolicy()

		runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{
			RunID:  "run-a56-sandbox-react-fallback-deny-run",
			Input:  "fallback-deny",
			Policy: &policy,
		}, runHandler)
		streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{
			RunID:  "run-a56-sandbox-react-fallback-deny-stream",
			Input:  "fallback-deny",
			Policy: &policy,
		}, streamHandler)
		if runErr == nil || streamErr == nil {
			t.Fatalf("fallback deny should fail, runErr=%v streamErr=%v", runErr, streamErr)
		}
		if runRes.Error == nil || streamRes.Error == nil || runRes.Error.Class != types.ErrSecurity || streamRes.Error.Class != types.ErrSecurity {
			t.Fatalf("fallback deny classified errors mismatch run=%#v stream=%#v", runRes.Error, streamRes.Error)
		}
		if runRes.Error.Details["reason_code"] != "sandbox.launch_failed" ||
			streamRes.Error.Details["reason_code"] != "sandbox.launch_failed" {
			t.Fatalf("fallback deny reason mismatch run=%#v stream=%#v", runRes.Error.Details["reason_code"], streamRes.Error.Details["reason_code"])
		}
		runPayload := lastRunFinishedPayload(t, runCollector.events)
		streamPayload := lastRunFinishedPayload(t, streamCollector.events)
		assertReactParityPayload(t, runPayload, streamPayload, runtimeconfig.RuntimeReactTerminationToolDispatchFailed)
		if runPayload["sandbox_launch_failed_total"] != 1 ||
			streamPayload["sandbox_launch_failed_total"] != 1 ||
			runPayload["sandbox_reason_code"] != "sandbox.launch_failed" ||
			streamPayload["sandbox_reason_code"] != "sandbox.launch_failed" {
			t.Fatalf("fallback deny payload mismatch run=%#v stream=%#v", runPayload, streamPayload)
		}

		runRecord := findRunRecord(t, mgr.RecentRuns(20), "run-a56-sandbox-react-fallback-deny-run")
		streamRecord := findRunRecord(t, mgr.RecentRuns(20), "run-a56-sandbox-react-fallback-deny-stream")
		runSnapshot := sandboxCapabilityDeliverySnapshotFromRunRecord(runRecord)
		streamSnapshot := sandboxCapabilityDeliverySnapshotFromRunRecord(streamRecord)
		if !reflect.DeepEqual(runSnapshot, streamSnapshot) {
			t.Fatalf("fallback deny run/stream snapshot mismatch run=%#v stream=%#v", runSnapshot, streamSnapshot)
		}
		if runSnapshot.SandboxFallbackUsed ||
			runSnapshot.SandboxReasonCode != "sandbox.launch_failed" ||
			runSnapshot.SandboxLaunchFailedTotal != 1 ||
			runSnapshot.SandboxCapabilityMismatchTotal != 0 {
			t.Fatalf("fallback deny snapshot mismatch: %#v", runSnapshot)
		}
	})
}

func TestSandboxExecutionIsolationContractReactCapabilityMismatchRunStreamParity(t *testing.T) {
	runModel := &scriptedReactModel{
		generateSteps: []scriptedGenerateStep{
			{
				response: types.ModelResponse{
					ToolCalls: []types.ToolCall{{CallID: "cap-mismatch-call-1", Name: "local.exec"}},
				},
			},
		},
	}
	streamModel := &scriptedReactModel{
		streamSteps: []scriptedStreamStep{
			{
				events: []types.ModelEvent{
					{
						Type: types.ModelEventTypeToolCall,
						ToolCall: &types.ToolCall{
							CallID: "cap-mismatch-call-1",
							Name:   "local.exec",
						},
					},
					{Type: types.ModelEventTypeResponseCompleted},
				},
			},
		},
	}
	cfg := strings.Join([]string{
		"runtime:",
		"  react:",
		"    enabled: true",
		"    stream_tool_dispatch_enabled: true",
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
		"      required_capabilities:",
		"        - network_off",
		"",
	}, "\n")
	executor := &sandboxIntegrationExecutor{
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
	}
	runEngine, streamEngine, mgr := newSandboxReactRunStreamHarness(t, cfg, runModel, streamModel, executor)
	runCollector := &eventCollector{}
	streamCollector := &eventCollector{}
	runHandler := newRunStreamDispatcherHandler(mgr, runCollector)
	streamHandler := newRunStreamDispatcherHandler(mgr, streamCollector)
	policy := defaultSandboxReactPolicy()

	runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{
		RunID:  "run-a56-sandbox-react-cap-mismatch-run",
		Input:  "capability-mismatch",
		Policy: &policy,
	}, runHandler)
	streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{
		RunID:  "run-a56-sandbox-react-cap-mismatch-stream",
		Input:  "capability-mismatch",
		Policy: &policy,
	}, streamHandler)
	if runErr == nil || streamErr == nil {
		t.Fatalf("capability mismatch should fail, runErr=%v streamErr=%v", runErr, streamErr)
	}
	if runRes.Error == nil || streamRes.Error == nil || runRes.Error.Class != types.ErrSecurity || streamRes.Error.Class != types.ErrSecurity {
		t.Fatalf("capability mismatch classified errors mismatch run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
	if runRes.Error.Details["reason_code"] != "sandbox.capability_mismatch" ||
		streamRes.Error.Details["reason_code"] != "sandbox.capability_mismatch" {
		t.Fatalf("capability mismatch reason mismatch run=%#v stream=%#v", runRes.Error.Details["reason_code"], streamRes.Error.Details["reason_code"])
	}
	runPayload := lastRunFinishedPayload(t, runCollector.events)
	streamPayload := lastRunFinishedPayload(t, streamCollector.events)
	assertReactParityPayload(t, runPayload, streamPayload, runtimeconfig.RuntimeReactTerminationToolDispatchFailed)
	if runPayload["sandbox_capability_mismatch_total"] != 1 ||
		streamPayload["sandbox_capability_mismatch_total"] != 1 ||
		runPayload["sandbox_reason_code"] != "sandbox.capability_mismatch" ||
		streamPayload["sandbox_reason_code"] != "sandbox.capability_mismatch" {
		t.Fatalf("capability mismatch payload mismatch run=%#v stream=%#v", runPayload, streamPayload)
	}

	runRecord := findRunRecord(t, mgr.RecentRuns(20), "run-a56-sandbox-react-cap-mismatch-run")
	streamRecord := findRunRecord(t, mgr.RecentRuns(20), "run-a56-sandbox-react-cap-mismatch-stream")
	runSnapshot := sandboxCapabilityDeliverySnapshotFromRunRecord(runRecord)
	streamSnapshot := sandboxCapabilityDeliverySnapshotFromRunRecord(streamRecord)
	if !reflect.DeepEqual(runSnapshot, streamSnapshot) {
		t.Fatalf("capability mismatch run/stream snapshot mismatch run=%#v stream=%#v", runSnapshot, streamSnapshot)
	}
	if runSnapshot.SandboxFallbackUsed ||
		runSnapshot.SandboxReasonCode != "sandbox.capability_mismatch" ||
		runSnapshot.SandboxLaunchFailedTotal != 0 ||
		runSnapshot.SandboxCapabilityMismatchTotal != 1 {
		t.Fatalf("capability mismatch snapshot mismatch: %#v", runSnapshot)
	}
}

func newRunStreamDispatcherHandler(mgr *runtimeconfig.Manager, collector *eventCollector) types.EventHandler {
	if collector == nil {
		return nil
	}
	if mgr == nil {
		return collector
	}
	return dispatcherHandler{
		dispatcher: event.NewDispatcher(
			event.NewRuntimeRecorder(mgr),
			collector,
		),
	}
}

func newSandboxReactRunStreamHarness(
	t *testing.T,
	cfg string,
	runModel types.ModelClient,
	streamModel types.ModelClient,
	executor types.SandboxExecutor,
) (*runner.Engine, *runner.Engine, *runtimeconfig.Manager) {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "runtime-a56-sandbox-react.yaml")
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A56_SANDBOX_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	if executor != nil {
		mgr.SetSandboxExecutor(executor)
	}

	reg := local.NewRegistry()
	if _, err := reg.Register(sandboxIntegrationAdapterTool{}); err != nil {
		t.Fatalf("register sandbox adapter tool: %v", err)
	}

	runEngine := runner.New(
		runModel,
		runner.WithRuntimeManager(mgr),
		runner.WithLocalRegistry(reg),
	)
	streamEngine := runner.New(
		streamModel,
		runner.WithRuntimeManager(mgr),
		runner.WithLocalRegistry(reg),
	)
	return runEngine, streamEngine, mgr
}

func defaultSandboxReactPolicy() types.LoopPolicy {
	policy := types.DefaultLoopPolicy()
	policy.MaxIterations = 6
	policy.ToolCallLimit = 6
	return policy
}

func assertSandboxToolFeedback(
	req types.ModelRequest,
	wantCallID string,
	wantContent string,
	wantFallbackReason string,
) error {
	if len(req.ToolResult) != 1 {
		return fmt.Errorf("tool result count=%d, want 1: %#v", len(req.ToolResult), req.ToolResult)
	}
	outcome := req.ToolResult[0]
	if strings.TrimSpace(outcome.CallID) != strings.TrimSpace(wantCallID) {
		return fmt.Errorf("tool call id=%q, want %q", outcome.CallID, wantCallID)
	}
	if strings.TrimSpace(outcome.Result.Content) != strings.TrimSpace(wantContent) {
		return fmt.Errorf("tool result content=%q, want %q", outcome.Result.Content, wantContent)
	}
	if strings.TrimSpace(wantFallbackReason) == "" {
		return nil
	}
	if outcome.Result.Structured == nil {
		return fmt.Errorf("tool result structured missing, want fallback reason %q", wantFallbackReason)
	}
	fallbackReason, _ := outcome.Result.Structured["sandbox_fallback_reason"].(string)
	if strings.TrimSpace(fallbackReason) != strings.TrimSpace(wantFallbackReason) {
		return fmt.Errorf("sandbox_fallback_reason=%q, want %q", fallbackReason, wantFallbackReason)
	}
	return nil
}

func assertSandboxEgressToolFeedback(
	req types.ModelRequest,
	wantCallID string,
	wantContent string,
	wantReasonCode string,
) error {
	if len(req.ToolResult) != 1 {
		return fmt.Errorf("tool result count=%d, want 1: %#v", len(req.ToolResult), req.ToolResult)
	}
	outcome := req.ToolResult[0]
	if strings.TrimSpace(outcome.CallID) != strings.TrimSpace(wantCallID) {
		return fmt.Errorf("tool call id=%q, want %q", outcome.CallID, wantCallID)
	}
	if strings.TrimSpace(outcome.Result.Content) != strings.TrimSpace(wantContent) {
		return fmt.Errorf("tool result content=%q, want %q", outcome.Result.Content, wantContent)
	}
	if outcome.Result.Structured == nil {
		return fmt.Errorf("tool result structured missing, want reason %q", wantReasonCode)
	}
	reasonCode, _ := outcome.Result.Structured["sandbox_reason_code"].(string)
	if strings.TrimSpace(reasonCode) != strings.TrimSpace(wantReasonCode) {
		return fmt.Errorf("sandbox_reason_code=%q, want %q", reasonCode, wantReasonCode)
	}
	egressAction, _ := outcome.Result.Structured["sandbox_egress_action"].(string)
	if strings.TrimSpace(egressAction) != runtimeconfig.SecuritySandboxEgressActionAllowAndRecord {
		return fmt.Errorf("sandbox_egress_action=%q, want %q", egressAction, runtimeconfig.SecuritySandboxEgressActionAllowAndRecord)
	}
	return nil
}
