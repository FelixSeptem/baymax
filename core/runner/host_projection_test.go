package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/context/budgetprojection"
	"github.com/FelixSeptem/baymax/context/hostprojection"
	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/FelixSeptem/baymax/tool/local"
)

func TestRunAndStreamUseEquivalentHostProjectionAssembly(t *testing.T) {
	mgr := newHostProjectionRuntimeManager(t)
	descriptor := runnerHostProjection(t)
	var runMessages, streamMessages []types.Message
	runResolver := &runnerHostProjectionResolver{material: hostprojection.Material{Role: "system", Content: "current run context"}}
	streamResolver := &runnerHostProjectionResolver{material: hostprojection.Material{Role: "system", Content: "current run context"}}

	runEngine := New(&fakeModel{generate: func(_ context.Context, request types.ModelRequest) (types.ModelResponse, error) {
		runMessages = append([]types.Message(nil), request.Messages...)
		return types.ModelResponse{FinalAnswer: "done"}, nil
	}}, WithRuntimeManager(mgr))
	streamEngine := New(&fakeModel{stream: func(_ context.Context, request types.ModelRequest, emit func(types.ModelEvent) error) error {
		streamMessages = append([]types.Message(nil), request.Messages...)
		return emit(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: "done"})
	}}, WithRuntimeManager(mgr))

	base := types.RunRequest{
		SessionID:             "session-host-projection",
		Input:                 "original input",
		Messages:              []types.Message{{Role: "user", Content: "original request"}},
		HostContextProjection: &descriptor,
	}
	runRequest := base
	runRequest.RunID = "run-host-projection"
	runRequest.HostContextResolver = runResolver
	streamRequest := base
	streamRequest.RunID = "stream-host-projection"
	streamRequest.HostContextResolver = streamResolver

	runResult, runErr := runEngine.Run(context.Background(), runRequest, nil)
	streamResult, streamErr := streamEngine.Stream(context.Background(), streamRequest, nil)
	if runErr != nil || streamErr != nil {
		t.Fatalf("Run/Stream errors = %v/%v", runErr, streamErr)
	}
	if runResult.FinalAnswer != "done" || streamResult.FinalAnswer != "done" {
		t.Fatalf("Run/Stream answers = %q/%q, want done", runResult.FinalAnswer, streamResult.FinalAnswer)
	}
	want := []types.Message{{Role: "user", Content: "original request"}, {Role: "system", Content: "current run context"}}
	if !sameMessages(runMessages, want) || !sameMessages(streamMessages, want) {
		t.Fatalf("Run/Stream messages = %#v/%#v, want %#v", runMessages, streamMessages, want)
	}
	if runResolver.calls != 1 || streamResolver.calls != 1 {
		t.Fatalf("Run/Stream resolver calls = %d/%d, want 1/1", runResolver.calls, streamResolver.calls)
	}
}

func TestHostProjectionPreservesReadOnlyBudgetProjectionAndExistingPressureTerminal(t *testing.T) {
	manager := newHostProjectionPressureRuntimeManager(t)
	descriptor := runnerHostProjection(t)
	cases := []struct {
		name     string
		facts    budgetprojection.BudgetFacts
		complete bool
		pressure budgetprojection.BudgetPressureLevel
	}{
		{
			name: "incomplete",
			facts: budgetprojection.BudgetFacts{
				Iteration: &budgetprojection.BudgetLimitUsage{Limit: 4, Used: 1},
				ToolCall:  &budgetprojection.BudgetLimitUsage{Limit: 2, Used: 1},
			},
			complete: false,
			pressure: budgetprojection.PressureElevated,
		},
		{
			name: "saturated",
			facts: budgetprojection.BudgetFacts{
				Iteration: &budgetprojection.BudgetLimitUsage{Limit: 1, Used: 1},
				ToolCall:  &budgetprojection.BudgetLimitUsage{Limit: 1, Used: 1},
				Time:      &budgetprojection.BudgetTimeFacts{BudgetMS: 1, ElapsedMS: 1},
				Cost:      &budgetprojection.BudgetCostFacts{HardThreshold: 1, EstimateTotal: 1},
			},
			complete: true,
			pressure: budgetprojection.PressureSaturated,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := budgetprojection.DeriveBudgetProjection(tc.facts)
			if before.Complete != tc.complete || before.Pressure != tc.pressure {
				t.Fatalf("derived budget=%#v, want complete=%t pressure=%q", before, tc.complete, tc.pressure)
			}
			resolver := &runnerHostProjectionResolver{material: hostprojection.Material{Role: "system", Content: strings.Repeat("current authorized material ", 16)}}
			model := &fakeModel{generate: func(_ context.Context, _ types.ModelRequest) (types.ModelResponse, error) {
				return types.ModelResponse{FinalAnswer: "done"}, nil
			}}
			collector := &eventCollector{}
			result, err := New(model, WithRuntimeManager(manager)).Run(context.Background(), types.RunRequest{
				RunID:                 "run-host-projection-budget-" + tc.name,
				SessionID:             "session-host-projection-budget",
				Input:                 strings.Repeat("input ", 16),
				Messages:              []types.Message{{Role: "user", Content: "original request"}},
				HostContextProjection: &descriptor,
				HostContextResolver:   resolver,
			}, collector)
			if err != nil || result.TerminalOutcome == nil || result.TerminalOutcome.State != types.RunStateCompleted || resolver.calls != 1 {
				t.Fatalf("Run() result=%#v resolver_calls=%d err=%v, want existing completed terminal", result, resolver.calls, err)
			}
			after := budgetprojection.DeriveBudgetProjection(tc.facts)
			if !reflect.DeepEqual(after, before) || after.Digest != before.Digest {
				t.Fatalf("budget projection changed across host assembly: before=%#v after=%#v", before, after)
			}
			finished, ok := collector.lastNonTimelineEvent()
			if !ok || finished.Type != "run.finished" || finished.Payload["terminal_state"] != string(types.RunStateCompleted) {
				t.Fatalf("finished event=%#v, want existing completed terminal", finished)
			}
		})
	}
}

func TestRunAndStreamNormalizeProjectionBudgetHandoffAndRejectionEquivalently(t *testing.T) {
	descriptor := runnerHostProjection(t)
	budget := budgetprojection.DeriveBudgetProjection(budgetprojection.BudgetFacts{
		Iteration: &budgetprojection.BudgetLimitUsage{Limit: 4, Used: 1},
		ToolCall:  &budgetprojection.BudgetLimitUsage{Limit: 2, Used: 1},
		Time:      &budgetprojection.BudgetTimeFacts{BudgetMS: 100, ElapsedMS: 25},
		Cost:      &budgetprojection.BudgetCostFacts{HardThreshold: 10, EstimateTotal: 2},
	})

	newRegistry := func(t *testing.T) *local.Registry {
		t.Helper()
		registry := local.NewRegistry()
		if _, err := registry.Register(&fakeTool{name: "echo", invoke: func(context.Context, map[string]any) (types.ToolResult, error) {
			return types.ToolResult{Content: "tool-ok"}, nil
		}}); err != nil {
			t.Fatalf("register tool: %v", err)
		}
		return registry
	}
	var runRequests, streamRequests [][]types.Message
	runTurns := 0
	runEngine := New(&fakeModel{generate: func(_ context.Context, request types.ModelRequest) (types.ModelResponse, error) {
		runRequests = append(runRequests, append([]types.Message(nil), request.Messages...))
		runTurns++
		if runTurns == 1 {
			return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "run-host-projection-tool", Name: "local.echo"}}}, nil
		}
		if len(request.ToolResult) != 1 || request.ToolResult[0].Result.Content != "tool-ok" {
			return types.ModelResponse{}, errors.New("Run tool result was not returned to the model")
		}
		return types.ModelResponse{FinalAnswer: "done"}, nil
	}}, WithRuntimeManager(newHostProjectionHandoffRuntimeManager(t, "BAYMAX_HOST_PROJECTION_RUN_HANDOFF_TEST")), WithLocalRegistry(newRegistry(t)))
	streamTurns := 0
	streamEngine := New(&fakeModel{stream: func(_ context.Context, request types.ModelRequest, emit func(types.ModelEvent) error) error {
		streamRequests = append(streamRequests, append([]types.Message(nil), request.Messages...))
		streamTurns++
		if streamTurns == 1 {
			return emit(types.ModelEvent{Type: types.ModelEventTypeToolCall, ToolCall: &types.ToolCall{CallID: "stream-host-projection-tool", Name: "local.echo"}})
		}
		if len(request.ToolResult) != 1 || request.ToolResult[0].Result.Content != "tool-ok" {
			return errors.New("Stream tool result was not returned to the model")
		}
		return emit(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "done"})
	}}, WithRuntimeManager(newHostProjectionHandoffRuntimeManager(t, "BAYMAX_HOST_PROJECTION_STREAM_HANDOFF_TEST")), WithLocalRegistry(newRegistry(t)))
	runResolver := &runnerHostProjectionResolver{material: hostprojection.Material{Role: "system", Content: "current authorized context"}}
	streamResolver := &runnerHostProjectionResolver{material: hostprojection.Material{Role: "system", Content: "current authorized context"}}
	base := types.RunRequest{
		SessionID: "session-host-projection-normalized", Input: "original input",
		Messages: []types.Message{{Role: "user", Content: "original request"}}, HostContextProjection: &descriptor,
	}
	runRequest := base
	runRequest.RunID, runRequest.HostContextResolver = "run-host-projection-normalized", runResolver
	streamRequest := base
	streamRequest.RunID, streamRequest.HostContextResolver = "stream-host-projection-normalized", streamResolver
	runCollector, streamCollector := &eventCollector{}, &eventCollector{}
	runResult, runErr := runEngine.Run(context.Background(), runRequest, runCollector)
	streamResult, streamErr := streamEngine.Stream(context.Background(), streamRequest, streamCollector)
	if runErr != nil || streamErr != nil || runResult.FinalAnswer != "done" || streamResult.FinalAnswer != "done" || runResolver.calls != 2 || streamResolver.calls != 2 {
		t.Fatalf("Run/Stream results=%#v/%#v errors=%v/%v resolver_calls=%d/%d", runResult, streamResult, runErr, streamErr, runResolver.calls, streamResolver.calls)
	}
	if !reflect.DeepEqual(runRequests, streamRequests) {
		t.Fatalf("Run/Stream assembled messages=%#v/%#v", runRequests, streamRequests)
	}
	if again := budgetprojection.DeriveBudgetProjection(budgetprojection.BudgetFacts{
		Iteration: &budgetprojection.BudgetLimitUsage{Limit: 4, Used: 1}, ToolCall: &budgetprojection.BudgetLimitUsage{Limit: 2, Used: 1},
		Time: &budgetprojection.BudgetTimeFacts{BudgetMS: 100, ElapsedMS: 25}, Cost: &budgetprojection.BudgetCostFacts{HardThreshold: 10, EstimateTotal: 2},
	}); !reflect.DeepEqual(budget, again) {
		t.Fatalf("budget projection is not stable: first=%#v second=%#v", budget, again)
	}
	if got, want := normalizedHostProjectionHandoffEvents(runCollector.evs), normalizedHostProjectionHandoffEvents(streamCollector.evs); !reflect.DeepEqual(got, want) {
		t.Fatalf("Run/Stream normalized handoff events=%#v/%#v", got, want)
	}

	invalid := descriptor
	invalid.SchemaVersion = "host_context_projection.v2"
	invalidRunResolver := &runnerHostProjectionResolver{material: hostprojection.Material{Role: "system", Content: "must not resolve"}}
	invalidStreamResolver := &runnerHostProjectionResolver{material: hostprojection.Material{Role: "system", Content: "must not resolve"}}
	invalidRun := base
	invalidRun.RunID, invalidRun.HostContextProjection, invalidRun.HostContextResolver = "run-host-projection-invalid-normalized", &invalid, invalidRunResolver
	invalidStream := base
	invalidStream.RunID, invalidStream.HostContextProjection, invalidStream.HostContextResolver = "stream-host-projection-invalid-normalized", &invalid, invalidStreamResolver
	_, runErr = runEngine.Run(context.Background(), invalidRun, nil)
	_, streamErr = streamEngine.Stream(context.Background(), invalidStream, nil)
	if hostProjectionCode(runErr) != hostprojection.CodeUnsupportedVersion || hostProjectionCode(streamErr) != hostprojection.CodeUnsupportedVersion || invalidRunResolver.calls != 0 || invalidStreamResolver.calls != 0 {
		t.Fatalf("Run/Stream invalid classifications=%q/%q resolver_calls=%d/%d, want equal pre-resolution rejection", hostProjectionCode(runErr), hostProjectionCode(streamErr), invalidRunResolver.calls, invalidStreamResolver.calls)
	}
}

func newHostProjectionRuntimeManager(t *testing.T) *runtimeconfig.Manager {
	t.Helper()
	journalPath := filepath.ToSlash(filepath.Join(t.TempDir(), "journal.jsonl"))
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := "context_assembler:\n  enabled: true\n  journal_path: '" + journalPath + "'\n  stage2_routing_and_disclosure:\n    enabled: false\n  pressure_compaction_and_swapback:\n    enabled: false\n"
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write runtime config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_HOST_PROJECTION_TEST"})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	return mgr
}

func newHostProjectionPressureRuntimeManager(t *testing.T) *runtimeconfig.Manager {
	t.Helper()
	journalPath := filepath.ToSlash(filepath.Join(t.TempDir(), "journal.jsonl"))
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := "context_assembler:\n  enabled: true\n  journal_path: '" + journalPath + "'\n  stage2_routing_and_disclosure:\n    enabled: false\n  pressure_compaction_and_swapback:\n    enabled: true\n    max_context_tokens: 20\n    percent_thresholds:\n      safe: 10\n      comfort: 20\n      warning: 30\n      danger: 40\n      emergency: 50\n    absolute_thresholds:\n      safe: 2\n      comfort: 4\n      warning: 6\n      danger: 8\n      emergency: 10\n    compaction:\n      mode: truncate\n"
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write runtime config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_HOST_PROJECTION_PRESSURE_TEST"})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	return mgr
}

func newHostProjectionHandoffRuntimeManager(t *testing.T, prefix string) *runtimeconfig.Manager {
	t.Helper()
	journalPath := filepath.ToSlash(filepath.Join(t.TempDir(), "journal.jsonl"))
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := "runtime:\n  context:\n    jit:\n      compaction:\n        handoff:\n          enabled: true\ncontext_assembler:\n  enabled: true\n  journal_path: '" + journalPath + "'\n  stage2_routing_and_disclosure:\n    enabled: false\n  pressure_compaction_and_swapback:\n    enabled: false\n"
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write runtime config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: prefix})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	return mgr
}

func runnerHostProjection(t *testing.T) hostprojection.Descriptor {
	t.Helper()
	descriptor := hostprojection.Descriptor{
		SchemaVersion:     hostprojection.SchemaVersionV1,
		ProjectionID:      "projection-runner",
		ProjectionVersion: 1,
		CapabilityDigest:  "capability-runner",
		Capabilities:      []hostprojection.Capability{hostprojection.CapabilityReference},
		SourceReferences:  []hostprojection.SourceReference{{Reference: "opaque://source/runner", Digest: "source-digest-runner"}},
	}
	digest, err := hostprojection.CanonicalDigest(descriptor)
	if err != nil {
		t.Fatalf("CanonicalDigest() error = %v", err)
	}
	descriptor.Digest = digest
	return descriptor
}

type runnerHostProjectionResolver struct {
	calls    int
	material hostprojection.Material
	err      error
}

func (r *runnerHostProjectionResolver) ResolveHostContext(_ context.Context, _ hostprojection.SourceReference) (hostprojection.Material, error) {
	r.calls++
	if r.err != nil {
		return hostprojection.Material{}, r.err
	}
	return r.material, nil
}

func sameMessages(got, want []types.Message) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range want {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

func normalizedHostProjectionHandoffEvents(events []types.Event) []map[string]any {
	normalized := make([]map[string]any, 0, len(events))
	for _, event := range events {
		if event.Type != types.EventTypeContextHandoff {
			continue
		}
		payload := make(map[string]any, len(event.Payload))
		for key, value := range event.Payload {
			payload[key] = value
		}
		normalized = append(normalized, payload)
	}
	return normalized
}

func hostProjectionCode(err error) hostprojection.Code {
	var classified *hostprojection.Error
	if errors.As(err, &classified) {
		return classified.Code
	}
	return ""
}
