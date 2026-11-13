package sandboxexternal

import (
	"context"
	"sort"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

// ExecError carries a frozen sandbox reason code alongside the underlying error.
type ExecError struct {
	Reason string
	Err    error
}

func (e *ExecError) Error() string {
	if e.Err != nil {
		return e.Reason + ": " + e.Err.Error()
	}
	return e.Reason
}

// baseExecutor holds shared probe/translation state for external adapters.
type baseExecutor struct {
	name             string
	canonicalBackend string
	probe            types.SandboxCapabilityProbe
}

func newBase(name, canonical string, caps, modes []string) baseExecutor {
	return baseExecutor{
		name:             name,
		canonicalBackend: canonical,
		probe: types.SandboxCapabilityProbe{
			Backend:        canonical,
			Capabilities:   caps,
			SupportedModes: modes,
		},
	}
}

// Probe returns the adapter's static capability declaration.
func (b *baseExecutor) Probe(_ context.Context) (types.SandboxCapabilityProbe, error) {
	return b.probe, nil
}

// CanonicalizeResult normalizes violation codes (lowercase, dedupe, sort) and
// keeps the result stable across backends.
func CanonicalizeResult(in types.SandboxExecResult) types.SandboxExecResult {
	out := in
	if len(out.ViolationCodes) == 0 {
		return out
	}
	normalized := make([]string, 0, len(out.ViolationCodes))
	seen := map[string]struct{}{}
	for _, code := range out.ViolationCodes {
		key := strings.ToLower(strings.TrimSpace(code))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, key)
	}
	sort.Strings(normalized)
	out.ViolationCodes = normalized
	return out
}
