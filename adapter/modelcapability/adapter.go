// Package modelcapability provides an explicit capability-discovery wrapper
// for model clients that intentionally implement only the minimal model SPI.
package modelcapability

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

const discoverySource = "explicit_adapter"

// Config describes the identity and capabilities asserted by an embedding
// application for a wrapped model client.
type Config struct {
	Client       types.ModelClient
	Provider     string
	Model        string
	Capabilities []types.ModelCapability
}

// Adapter delegates model execution and explicitly implements capability
// discovery for local, test, and other opt-in integrations.
type Adapter struct {
	client       types.ModelClient
	provider     string
	model        string
	capabilities []types.ModelCapability
}

// New validates config and constructs an explicit capability adapter.
func New(cfg Config) (*Adapter, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("model capability adapter: client is required")
	}
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" {
		return nil, fmt.Errorf("model capability adapter: provider is required")
	}
	capabilities, err := normalizeCapabilities(cfg.Capabilities)
	if err != nil {
		return nil, err
	}
	return &Adapter{
		client:       cfg.Client,
		provider:     provider,
		model:        strings.TrimSpace(cfg.Model),
		capabilities: capabilities,
	}, nil
}

// Wrap is a convenience alias for New when the client is supplied separately
// from the rest of the configuration.
func Wrap(client types.ModelClient, cfg Config) (*Adapter, error) {
	cfg.Client = client
	return New(cfg)
}

func (a *Adapter) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	return a.client.Generate(ctx, req)
}

func (a *Adapter) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	return a.client.Stream(ctx, req, onEvent)
}

func (a *Adapter) ProviderName() string {
	if a == nil {
		return ""
	}
	return a.provider
}

func (a *Adapter) DiscoverCapabilities(context.Context, types.ModelRequest) (types.ProviderCapabilities, error) {
	if a == nil {
		return types.ProviderCapabilities{}, fmt.Errorf("model capability adapter: adapter is nil")
	}
	support := make(map[types.ModelCapability]types.CapabilitySupport, len(a.capabilities))
	for _, capability := range a.capabilities {
		support[capability] = types.CapabilitySupportSupported
	}
	return types.ProviderCapabilities{
		Provider: a.provider,
		Model:    a.model,
		Support:  support,
		Source:   discoverySource,
	}, nil
}

func normalizeCapabilities(input []types.ModelCapability) ([]types.ModelCapability, error) {
	seen := make(map[types.ModelCapability]struct{}, len(input))
	for _, raw := range input {
		capability := types.ModelCapability(strings.ToLower(strings.TrimSpace(string(raw))))
		if capability == "" {
			return nil, fmt.Errorf("model capability adapter: capability must not be blank")
		}
		switch capability {
		case types.ModelCapabilityStreaming, types.ModelCapabilityToolCall:
		default:
			return nil, fmt.Errorf("model capability adapter: unknown capability %q", raw)
		}
		seen[capability] = struct{}{}
	}
	capabilities := make([]types.ModelCapability, 0, len(seen))
	for capability := range seen {
		capabilities = append(capabilities, capability)
	}
	sort.Slice(capabilities, func(i, j int) bool { return capabilities[i] < capabilities[j] })
	return capabilities, nil
}

var _ types.ModelClient = (*Adapter)(nil)
var _ types.ModelCapabilityDiscovery = (*Adapter)(nil)
