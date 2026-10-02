package modelcapability

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
)

type testModel struct {
	generateErr error
	streamErr   error
	events      []types.ModelEvent
}

func (m *testModel) Generate(context.Context, types.ModelRequest) (types.ModelResponse, error) {
	return types.ModelResponse{FinalAnswer: "generated"}, m.generateErr
}

func (m *testModel) Stream(_ context.Context, _ types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	for _, event := range m.events {
		if err := onEvent(event); err != nil {
			return err
		}
	}
	return m.streamErr
}

func TestNewNormalizesAndDiscoversExplicitCapabilities(t *testing.T) {
	client := &testModel{}
	adapter, err := New(Config{
		Client:       client,
		Provider:     " Local ",
		Model:        " test-model ",
		Capabilities: []types.ModelCapability{" streaming ", types.ModelCapabilityToolCall, types.ModelCapabilityStreaming},
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if got := adapter.ProviderName(); got != "local" {
		t.Fatalf("ProviderName() = %q, want local", got)
	}
	report, err := adapter.DiscoverCapabilities(context.Background(), types.ModelRequest{})
	if err != nil {
		t.Fatalf("DiscoverCapabilities failed: %v", err)
	}
	if report.Provider != "local" || report.Model != "test-model" || report.Source != "explicit_adapter" {
		t.Fatalf("unexpected report identity: %#v", report)
	}
	want := map[types.ModelCapability]types.CapabilitySupport{
		types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
		types.ModelCapabilityToolCall:  types.CapabilitySupportSupported,
	}
	if !reflect.DeepEqual(report.Support, want) {
		t.Fatalf("Support = %#v, want %#v", report.Support, want)
	}
}

func TestNewRejectsInvalidConfiguration(t *testing.T) {
	tests := []Config{
		{Provider: "local"},
		{Client: &testModel{}},
		{Client: &testModel{}, Provider: "local", Capabilities: []types.ModelCapability{"  "}},
		{Client: &testModel{}, Provider: "local", Capabilities: []types.ModelCapability{"unknown"}},
	}
	for index, cfg := range tests {
		if _, err := New(cfg); err == nil {
			t.Errorf("case %d: New returned nil error", index)
		}
	}
}

func TestAdapterDelegatesModelCallsAndErrors(t *testing.T) {
	generateErr := errors.New("generate failed")
	streamErr := errors.New("stream failed")
	client := &testModel{
		generateErr: generateErr,
		streamErr:   streamErr,
		events:      []types.ModelEvent{{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "hello"}},
	}
	adapter, err := New(Config{Client: client, Provider: "local"})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if _, err := adapter.Generate(context.Background(), types.ModelRequest{}); !errors.Is(err, generateErr) {
		t.Fatalf("Generate error = %v, want %v", err, generateErr)
	}
	var got []types.ModelEvent
	if err := adapter.Stream(context.Background(), types.ModelRequest{}, func(event types.ModelEvent) error {
		got = append(got, event)
		return nil
	}); !errors.Is(err, streamErr) {
		t.Fatalf("Stream error = %v, want %v", err, streamErr)
	}
	if !reflect.DeepEqual(got, client.events) {
		t.Fatalf("events = %#v, want %#v", got, client.events)
	}
}

var _ types.ModelClient = (*Adapter)(nil)
var _ types.ModelCapabilityDiscovery = (*Adapter)(nil)
