package main

import (
	"context"
	"fmt"

	"github.com/FelixSeptem/baymax/adapter/modelcapability"
	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/planbridge"
	"github.com/FelixSeptem/baymax/tool/local"
)

type model struct{ calls int }

func (m *model) Generate(_ context.Context, request types.ModelRequest) (types.ModelResponse, error) {
	m.calls++
	if len(request.ToolResult) != 0 {
		return types.ModelResponse{FinalAnswer: "offline completion"}, nil
	}
	return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "offline-call", Name: "local.echo"}}}, nil
}
func (m *model) Stream(ctx context.Context, request types.ModelRequest, emit func(types.ModelEvent) error) error {
	response, err := m.Generate(ctx, request)
	if err != nil {
		return err
	}
	for _, call := range response.ToolCalls {
		if err := emit(types.ModelEvent{Type: types.ModelEventTypeToolCall, ToolCall: &call}); err != nil {
			return err
		}
	}
	if response.FinalAnswer != "" {
		return emit(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: response.FinalAnswer})
	}
	return nil
}

type echoTool struct{ calls int }

func (*echoTool) Name() string               { return "echo" }
func (*echoTool) Description() string        { return "offline echo" }
func (*echoTool) JSONSchema() map[string]any { return map[string]any{"type": "object"} }
func (t *echoTool) Invoke(context.Context, map[string]any) (types.ToolResult, error) {
	t.calls++
	return types.ToolResult{Content: "offline tool result"}, nil
}

func binding(engine types.Runner) planbridge.Binding {
	return planbridge.Binding{SemanticDigest: "offline-semantics", Direct: &planbridge.DirectBinding{Runner: engine, Request: types.RunRequest{RunID: "offline-run", Input: "offline input", Capabilities: types.CapabilityRequirements{Required: []types.ModelCapability{types.ModelCapabilityStreaming}}}}, Admit: func(context.Context, planbridge.Identity) (planbridge.Admission, error) {
		return planbridge.Admission{AuthorityDigest: "offline-authority", ToolCapabilityDigest: "offline-tools", SemanticDigest: "offline-semantics"}, nil
	}}
}
func descriptor() planbridge.Descriptor {
	return planbridge.Descriptor{SchemaVersion: planbridge.SchemaVersion, PlanID: "offline-plan", PlanVersion: 1, Shape: planbridge.ShapeDirect, Delivery: planbridge.DeliveryInline, Trigger: planbridge.TriggerManual, AuthorityDigest: "offline-authority", ToolCapabilityDigest: "offline-tools", InputRef: "offline-input", InputDigest: "offline-input-digest", HandlerRef: "offline-handler", HandlerDigest: "offline-handler-digest"}
}

func main() {
	base := &model{}
	wrapped, err := modelcapability.Wrap(base, modelcapability.Config{Provider: "offline", Capabilities: []types.ModelCapability{types.ModelCapabilityStreaming, types.ModelCapabilityToolCall}})
	if err != nil {
		panic(err)
	}
	tool := &echoTool{}
	registry := local.NewRegistry()
	if _, err = registry.Register(tool); err != nil {
		panic(err)
	}
	engine := runner.New(wrapped, runner.WithLocalRegistry(registry))
	d, b := descriptor(), binding(engine)
	d, err = planbridge.Describe(d, b, planbridge.Limits{})
	if err != nil {
		panic(err)
	}
	p, err := planbridge.Compile(d, b, planbridge.Limits{})
	if err != nil {
		panic(err)
	}
	if p.Identity().Digest == "" || base.calls != 0 || tool.calls != 0 {
		panic("compile was not side-effect free")
	}
	fmt.Println("plan-compiled")
	run, err := p.Run(context.Background(), planbridge.Execution{})
	if err != nil || run.Direct == nil || run.Direct.TerminalOutcome.State != types.RunStateCompleted {
		panic(fmt.Sprintf("run: %v", err))
	}
	stream, err := p.Stream(context.Background(), planbridge.Execution{})
	if err != nil || stream.Direct == nil || stream.Direct.TerminalOutcome.State != types.RunStateCompleted {
		panic(fmt.Sprintf("stream: %v", err))
	}
	if run.Direct.RunID != stream.Direct.RunID || base.calls == 0 || tool.calls == 0 {
		panic("source owner parity")
	}
	fmt.Println("source-owner-preserved")
	fmt.Println("run-stream-parity")
	bad := d
	bad.PlanVersion++
	bad.SourceFacts["run_id"] = "different"
	if _, err = planbridge.Compile(bad, b, planbridge.Limits{}); err == nil {
		panic("invalid binding accepted")
	}
	fmt.Println("invalid-plan-rejected")
	mail := d
	mail.Delivery = planbridge.DeliveryMailbox
	if _, err = planbridge.Compile(mail, b, planbridge.Limits{}); err == nil {
		panic("mailbox accepted")
	}
	fmt.Println("unsupported-capability-rejected")
	if base.calls != 4 || tool.calls != 2 {
		panic("rejection activated source")
	}
	fmt.Println("no-side-effects-before-rejection")
	// Native completion is deliberately not a business write; the host owns its commit gate.
	fmt.Println("host-commit-required")
}
