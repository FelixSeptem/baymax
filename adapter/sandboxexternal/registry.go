package sandboxexternal

import (
	"fmt"
	"sort"
	"sync"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

// Reason codes: frozen failure classification for external sandbox adapters.
// They reuse the existing sandbox-*-matrix taxonomy and do not introduce a
// parallel contract.
const (
	ReasonLaunchFailed       = "sandbox.launch_failed"
	ReasonCapabilityMismatch = "sandbox.capability_mismatch"
	ReasonTimeout            = "sandbox.timeout"
	ReasonOOM                = "sandbox.oom"
	ReasonEgressDenied       = "sandbox.egress_denied"
	ReasonBackendUnavailable = "sandbox.backend_unavailable"
)

// Backend identifiers for mainstream external sandbox runtimes.
const (
	BackendE2B                = "e2b"
	BackendOpenSandbox        = "open-sandbox"
	BackendAnthropicSandboxRT = "anthropic-sandbox-runtime"
	BackendLocal              = "local"
)

// Config carries registration and runtime configuration for an external backend.
type Config struct {
	// Name is the external backend identifier (e.g. "e2b").
	Name string
	// CanonicalBackend is one of the four canonical profiles the adapter maps to.
	CanonicalBackend string
	// BaseURL and APIKey are required for cloud backends. Left empty disables
	// live execution (the executor reports backend_unavailable).
	//
	// There is deliberately no Enabled flag here: whether an external executor
	// is wired at all is decided by the host from
	// security.sandbox.enabled + security.sandbox.executor.external_backend.
	// Keeping the gate in config avoids two competing enable switches.
	BaseURL string
	APIKey  string
}

// Factory builds a SandboxExecutor from configuration.
type Factory func(cfg Config) (types.SandboxExecutor, error)

var (
	regMu    sync.RWMutex
	registry = map[string]Factory{}
)

func Register(name string, f Factory) {
	regMu.Lock()
	defer regMu.Unlock()
	registry[name] = f
}

func init() {
	Register(BackendE2B, NewE2BExecutor)
	Register(BackendOpenSandbox, NewOpenSandboxExecutor)
	Register(BackendAnthropicSandboxRT, NewAnthropicExecutor)
	// NewLocalExecutor returns a concrete type, so it needs a thin wrapper.
	Register(BackendLocal, func(c Config) (types.SandboxExecutor, error) { return NewLocalExecutor(c) })
}

// IsCanonicalBackend reports whether b is one of the four canonical profiles.
func IsCanonicalBackend(b string) bool {
	switch b {
	case runtimeconfig.SecuritySandboxBackendLinuxNSJail,
		runtimeconfig.SecuritySandboxBackendLinuxBwrap,
		runtimeconfig.SecuritySandboxBackendOCIRuntime,
		runtimeconfig.SecuritySandboxBackendWindowsJob:
		return true
	}
	return false
}

// NewExecutor resolves a registered backend and validates its canonical mapping.
func NewExecutor(cfg Config) (types.SandboxExecutor, error) {
	regMu.RLock()
	f, ok := registry[cfg.Name]
	regMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("sandboxexternal: unknown backend %q", cfg.Name)
	}
	if !IsCanonicalBackend(cfg.CanonicalBackend) {
		return nil, fmt.Errorf("sandboxexternal: backend %q maps to unsupported canonical profile %q", cfg.Name, cfg.CanonicalBackend)
	}
	return f(cfg)
}

// RegisteredBackends returns sorted registered backend identifiers.
func RegisteredBackends() []string {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
