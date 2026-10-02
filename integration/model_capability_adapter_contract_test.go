package integration

import (
	"context"
	"testing"

	"github.com/FelixSeptem/baymax/adapter/modelcapability"
	"github.com/FelixSeptem/baymax/core/runner"
	"github.com/FelixSeptem/baymax/core/types"
)

type capabilityAdapterContractModel struct {
	streamCalls int
}

func (m *capabilityAdapterContractModel) Generate(context.Context, types.ModelRequest) (types.ModelResponse, error) {
	return types.ModelResponse{FinalAnswer: "same-answer"}, nil
}

func (m *capabilityAdapterContractModel) Stream(_ context.Context, _ types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	m.streamCalls++
	return onEvent(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: "same-answer"})
}

func TestExplicitModelCapabilityAdapterPreservesRunStreamAdmission(t *testing.T) {
	client := &capabilityAdapterContractModel{}
	model, err := modelcapability.Wrap(client, modelcapability.Config{
		Provider:     "local",
		Capabilities: []types.ModelCapability{types.ModelCapabilityStreaming},
	})
	if err != nil {
		t.Fatalf("Wrap failed: %v", err)
	}
	engine := runner.New(model)
	run, err := engine.Run(context.Background(), types.RunRequest{
		Input: "hello",
		Capabilities: types.CapabilityRequirements{
			Required: []types.ModelCapability{types.ModelCapabilityStreaming},
		},
	}, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	stream, err := engine.Stream(context.Background(), types.RunRequest{Input: "hello"}, nil)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	if run.FinalAnswer != stream.FinalAnswer || run.FinalAnswer != "same-answer" {
		t.Fatalf("Run/Stream answers = %q/%q, want same-answer", run.FinalAnswer, stream.FinalAnswer)
	}
	if client.streamCalls != 1 {
		t.Fatalf("stream calls = %d, want 1", client.streamCalls)
	}
}

func TestMinimalModelWithoutDiscoveryIsNotInvokedByStrictStream(t *testing.T) {
	client := &capabilityAdapterContractModel{}
	res, err := runner.New(client).Stream(context.Background(), types.RunRequest{Input: "hello"}, nil)
	if err == nil || res.Error == nil {
		t.Fatal("expected strict discovery error")
	}
	if client.streamCalls != 0 {
		t.Fatalf("stream calls = %d, want 0", client.streamCalls)
	}
	if got := res.Error.Details["provider_reason"]; got != "capability_discovery_unavailable" {
		t.Fatalf("provider_reason = %#v, want capability_discovery_unavailable", got)
	}
}
