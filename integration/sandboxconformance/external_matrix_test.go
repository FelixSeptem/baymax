package sandboxconformance

import (
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestExternalBackendMatrixConforms(t *testing.T) {
	if err := ValidateExternalBackendMatrix(ExternalBackendMatrix); err != nil {
		t.Fatalf("external backend matrix invalid: %v", err)
	}
}

func TestExternalBackendCapabilityNegotiation(t *testing.T) {
	for _, s := range ExternalBackendMatrix {
		probe := types.SandboxCapabilityProbe{
			Backend:        s.Backend,
			Capabilities:   s.ProbeCapabilities,
			SupportedModes: s.SupportedModes,
		}
		if err := EvaluateCapabilityNegotiation(probe, s.RequiredCapabilities, s.SessionMode); err != nil {
			t.Fatalf("scenario %s capability negotiation failed: %v", s.ID, err)
		}
	}
}

func TestExternalBackendMissingCapabilityFails(t *testing.T) {
	s := ExternalBackendMatrix[0]
	probe := types.SandboxCapabilityProbe{
		Backend:        s.Backend,
		Capabilities:   []string{runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture}, // missing session_per_call
		SupportedModes: s.SupportedModes,
	}
	err := EvaluateCapabilityNegotiation(probe, s.RequiredCapabilities, s.SessionMode)
	if err == nil {
		t.Fatal("expected capability mismatch error")
	}
	if _, ok := err.(*ConformanceError); !ok {
		t.Fatalf("expected *ConformanceError, got %T", err)
	}
}
