package adaptercontractreplay

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	adaptermanifest "github.com/FelixSeptem/baymax/adapter/manifest"
	adapterprofile "github.com/FelixSeptem/baymax/adapter/profile"
)

const (
	replayTrackLegacyV1Alpha1 = "v1alpha1"
	replayTrackSandboxV1      = "sandbox.v1"
)

const (
	ReplayDriftSandboxBackendProfile = "sandbox_backend_profile_drift"
	ReplayDriftSandboxManifestCompat = "sandbox_manifest_compat_drift"
	ReplayDriftSandboxSessionMode    = "sandbox_session_mode_drift"
)

const (
	replayCodeUnknownProfileVersion = "adapter-contract-replay.unknown-profile-version"
)

type replayFixture struct {
	ProfileVersion string       `json:"profile_version"`
	Cases          []replayCase `json:"cases"`
}

type replayCase struct {
	Name                  string                   `json:"name"`
	Manifest              json.RawMessage          `json:"manifest"`
	RuntimeVersion        string                   `json:"runtime_version"`
	AvailableCapabilities []string                 `json:"available_capabilities"`
	Request               replayRequest            `json:"request"`
	Window                *replayWindow            `json:"window,omitempty"`
	ActivationContext     *replayActivationContext `json:"activation_context,omitempty"`
	Expected              replayExpected           `json:"expected"`
}

type replayRequest struct {
	Required         []string `json:"required"`
	Optional         []string `json:"optional"`
	StrategyOverride string   `json:"strategy_override"`
}

type replayWindow struct {
	Current       string `json:"current"`
	AllowPrevious bool   `json:"allow_previous"`
}

type replayActivationContext struct {
	HostOS            string   `json:"host_os,omitempty"`
	HostArch          string   `json:"host_arch,omitempty"`
	RequestedSession  string   `json:"requested_session,omitempty"`
	SupportedBackends []string `json:"supported_backends,omitempty"`
}

type replayExpected struct {
	ParseErrorCode         string   `json:"parse_error_code"`
	ParseErrorField        string   `json:"parse_error_field"`
	ActivationErrorCode    string   `json:"activation_error_code"`
	ActivationErrorField   string   `json:"activation_error_field"`
	ContractProfileVersion string   `json:"contract_profile_version"`
	StrategyApplied        string   `json:"strategy_applied"`
	StrategyOverride       bool     `json:"strategy_override"`
	ReasonCodes            []string `json:"reason_codes"`
	OptionalDowngrades     []string `json:"optional_downgrades"`
	DriftClass             string   `json:"drift_class,omitempty"`
}

func TestReplayContractManifestCompatibility(t *testing.T) {
	fixture := loadFixture(t, replayTrackLegacyV1Alpha1, "manifest-compatibility.json")
	for _, tc := range fixture.Cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			runResult := executeCase(t, tc)
			streamResult := executeCase(t, tc)
			if !reflect.DeepEqual(runResult, streamResult) {
				t.Fatalf("run/stream replay mismatch: run=%#v stream=%#v", runResult, streamResult)
			}
		})
	}
}

func TestReplayContractNegotiationTaxonomy(t *testing.T) {
	fixture := loadFixture(t, replayTrackLegacyV1Alpha1, "negotiation-outcomes.json")
	for _, tc := range fixture.Cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			runResult := executeCase(t, tc)
			streamResult := executeCase(t, tc)
			if !reflect.DeepEqual(runResult, streamResult) {
				t.Fatalf("run/stream replay mismatch: run=%#v stream=%#v", runResult, streamResult)
			}
		})
	}
}

func TestReplayContractSandboxProfilePackTrack(t *testing.T) {
	fixture := loadFixture(t, replayTrackSandboxV1, "manifest-compatibility.json")
	for _, tc := range fixture.Cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			runResult := executeCase(t, tc)
			streamResult := executeCase(t, tc)
			if !reflect.DeepEqual(runResult, streamResult) {
				t.Fatalf("sandbox run/stream replay mismatch: run=%#v stream=%#v", runResult, streamResult)
			}
		})
	}
}

func TestReplayContractMixedTracksBackwardCompatible(t *testing.T) {
	legacy := loadFixture(t, replayTrackLegacyV1Alpha1, "manifest-compatibility.json")
	sandbox := loadFixture(t, replayTrackSandboxV1, "manifest-compatibility.json")
	all := append([]replayCase{}, legacy.Cases...)
	all = append(all, sandbox.Cases...)
	for _, tc := range all {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			runResult := executeCase(t, tc)
			streamResult := executeCase(t, tc)
			if !reflect.DeepEqual(runResult, streamResult) {
				t.Fatalf("mixed-track run/stream replay mismatch: run=%#v stream=%#v", runResult, streamResult)
			}
		})
	}
}

func TestReplayContractProfileVersionValidation(t *testing.T) {
	if err := validateReplayProfileVersion(replayTrackLegacyV1Alpha1); err != nil {
		t.Fatalf("legacy profile track should be accepted: %v", err)
	}
	if err := validateReplayProfileVersion(replayTrackSandboxV1); err != nil {
		t.Fatalf("sandbox profile track should be accepted: %v", err)
	}
	err := validateReplayProfileVersion("sandbox.v99")
	if err == nil {
		t.Fatal("unknown profile track must fail fast")
	}
	ce := contractErr(t, err)
	if ce.Code != replayCodeUnknownProfileVersion || ce.Field != "profile_version" {
		t.Fatalf("unexpected profile version error classification: %#v", ce)
	}
}

type replayResult struct {
	ParseErrorCode         string
	ParseErrorField        string
	ActivationErrorCode    string
	ActivationErrorField   string
	ContractProfileVersion string
	StrategyApplied        string
	StrategyOverride       bool
	ReasonCodes            []string
	OptionalDowngrades     []string
}

func executeCase(t *testing.T, tc replayCase) replayResult {
	t.Helper()

	manifest, err := adaptermanifest.Parse(tc.Manifest)
	if tc.Expected.ParseErrorCode != "" {
		ce := contractErr(t, err)
		if ce.Code != tc.Expected.ParseErrorCode || ce.Field != tc.Expected.ParseErrorField {
			t.Fatalf("unexpected parse error: %#v", ce)
		}
		return replayResult{
			ParseErrorCode:  ce.Code,
			ParseErrorField: ce.Field,
		}
	}
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}

	window := adapterprofile.DefaultWindow()
	if tc.Window != nil {
		window, err = adapterprofile.NewWindow(tc.Window.Current, tc.Window.AllowPrevious)
		if err != nil {
			t.Fatalf("parse replay window: %v", err)
		}
	}

	activationCtx := adaptermanifest.ActivationContext{}
	if tc.ActivationContext != nil {
		activationCtx = adaptermanifest.ActivationContext{
			HostOS:            strings.ToLower(strings.TrimSpace(tc.ActivationContext.HostOS)),
			HostArch:          strings.ToLower(strings.TrimSpace(tc.ActivationContext.HostArch)),
			RequestedSession:  strings.ToLower(strings.TrimSpace(tc.ActivationContext.RequestedSession)),
			SupportedBackends: append([]string(nil), tc.ActivationContext.SupportedBackends...),
		}
	}
	result, err := adaptermanifest.ActivateWithRequestAndProfileWindowWithContext(manifest, tc.RuntimeVersion, tc.AvailableCapabilities, adaptermanifest.CapabilityRequest{
		Required:         append([]string(nil), tc.Request.Required...),
		Optional:         append([]string(nil), tc.Request.Optional...),
		StrategyOverride: tc.Request.StrategyOverride,
	}, window, activationCtx)
	if tc.Expected.ActivationErrorCode != "" {
		ce := contractErr(t, err)
		if ce.Code != tc.Expected.ActivationErrorCode || ce.Field != tc.Expected.ActivationErrorField {
			t.Fatalf("unexpected activation error: %#v", ce)
		}
		if tc.Expected.DriftClass != "" {
			drift := classifySandboxReplayDrift(ce.Code)
			if drift != tc.Expected.DriftClass {
				t.Fatalf("unexpected drift class: got=%q want=%q for code=%q", drift, tc.Expected.DriftClass, ce.Code)
			}
		}
		return replayResult{
			ActivationErrorCode:  ce.Code,
			ActivationErrorField: ce.Field,
		}
	}
	if err != nil {
		t.Fatalf("activate manifest: %v", err)
	}

	if tc.Expected.ContractProfileVersion != "" && result.ContractProfileVersion != tc.Expected.ContractProfileVersion {
		t.Fatalf("unexpected contract profile version: got=%s want=%s", result.ContractProfileVersion, tc.Expected.ContractProfileVersion)
	}
	if tc.Expected.StrategyApplied != "" && result.StrategyApplied != tc.Expected.StrategyApplied {
		t.Fatalf("unexpected strategy applied: got=%s want=%s", result.StrategyApplied, tc.Expected.StrategyApplied)
	}
	if result.StrategyOverride != tc.Expected.StrategyOverride {
		t.Fatalf("unexpected strategy override flag: got=%v want=%v", result.StrategyOverride, tc.Expected.StrategyOverride)
	}
	if tc.Expected.ReasonCodes != nil && !equalStringSlices(result.ReasonCodes, tc.Expected.ReasonCodes) {
		t.Fatalf("unexpected reason codes: got=%#v want=%#v", result.ReasonCodes, tc.Expected.ReasonCodes)
	}
	if tc.Expected.OptionalDowngrades != nil && !equalStringSlices(optionalCapabilities(result.OptionalDowngrades), tc.Expected.OptionalDowngrades) {
		t.Fatalf("unexpected optional downgrades: got=%#v want=%#v", optionalCapabilities(result.OptionalDowngrades), tc.Expected.OptionalDowngrades)
	}

	return replayResult{
		ContractProfileVersion: result.ContractProfileVersion,
		StrategyApplied:        result.StrategyApplied,
		StrategyOverride:       result.StrategyOverride,
		ReasonCodes:            append([]string(nil), result.ReasonCodes...),
		OptionalDowngrades:     optionalCapabilities(result.OptionalDowngrades),
	}
}

func optionalCapabilities(items []adaptermanifest.OptionalDowngrade) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Capability)
	}
	return out
}

func equalStringSlices(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

func loadFixture(t *testing.T, track string, name string) replayFixture {
	t.Helper()
	path := filepath.Join(repoRoot(t), "integration", "testdata", "adapter-contract-replay", track, name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	var fixture replayFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("parse fixture %s: %v", path, err)
	}
	if fixture.ProfileVersion == "" {
		t.Fatalf("fixture %s missing profile_version", path)
	}
	if err := validateReplayProfileVersion(fixture.ProfileVersion); err != nil {
		ce := contractErr(t, err)
		t.Fatalf("fixture %s has unsupported profile version %q (%s)", path, fixture.ProfileVersion, ce.Code)
	}
	if len(fixture.Cases) == 0 {
		t.Fatalf("fixture %s has no cases", path)
	}
	return fixture
}

func validateReplayProfileVersion(version string) error {
	normalized := strings.ToLower(strings.TrimSpace(version))
	switch normalized {
	case replayTrackLegacyV1Alpha1, replayTrackSandboxV1:
		return nil
	default:
		return &adaptermanifest.ContractError{
			Code:    replayCodeUnknownProfileVersion,
			Field:   "profile_version",
			Message: "profile version is not recognized in replay track",
		}
	}
}

func classifySandboxReplayDrift(code string) string {
	switch strings.TrimSpace(code) {
	case adaptermanifest.CodeSandboxProfileUnknown, adaptermanifest.CodeSandboxBackendUnsupported:
		return ReplayDriftSandboxBackendProfile
	case adaptermanifest.CodeSandboxHostMismatch, adaptermanifest.CodeCompatibilityMismatch:
		return ReplayDriftSandboxManifestCompat
	case adaptermanifest.CodeSandboxSessionUnsupported:
		return ReplayDriftSandboxSessionMode
	default:
		return ""
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve runtime caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

func contractErr(t *testing.T, err error) *adaptermanifest.ContractError {
	t.Helper()
	if err == nil {
		t.Fatal("expected contract error")
	}
	ce := &adaptermanifest.ContractError{}
	if !errors.As(err, &ce) {
		t.Fatalf("expected contract error, got %T (%v)", err, err)
	}
	return ce
}
