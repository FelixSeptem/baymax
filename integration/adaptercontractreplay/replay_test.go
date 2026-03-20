package adaptercontractreplay

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	adaptermanifest "github.com/FelixSeptem/baymax/adapter/manifest"
	adapterprofile "github.com/FelixSeptem/baymax/adapter/profile"
)

type replayFixture struct {
	ProfileVersion string       `json:"profile_version"`
	Cases          []replayCase `json:"cases"`
}

type replayCase struct {
	Name                  string          `json:"name"`
	Manifest              json.RawMessage `json:"manifest"`
	RuntimeVersion        string          `json:"runtime_version"`
	AvailableCapabilities []string        `json:"available_capabilities"`
	Request               replayRequest   `json:"request"`
	Window                *replayWindow   `json:"window,omitempty"`
	Expected              replayExpected  `json:"expected"`
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
}

func TestReplayContractManifestCompatibility(t *testing.T) {
	fixture := loadFixture(t, "manifest-compatibility.json")
	if fixture.ProfileVersion != adapterprofile.CurrentProfile {
		t.Fatalf("unexpected fixture profile version: %s", fixture.ProfileVersion)
	}
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
	fixture := loadFixture(t, "negotiation-outcomes.json")
	if fixture.ProfileVersion != adapterprofile.CurrentProfile {
		t.Fatalf("unexpected fixture profile version: %s", fixture.ProfileVersion)
	}
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

	result, err := adaptermanifest.ActivateWithRequestAndProfileWindow(manifest, tc.RuntimeVersion, tc.AvailableCapabilities, adaptermanifest.CapabilityRequest{
		Required:         append([]string(nil), tc.Request.Required...),
		Optional:         append([]string(nil), tc.Request.Optional...),
		StrategyOverride: tc.Request.StrategyOverride,
	}, window)
	if tc.Expected.ActivationErrorCode != "" {
		ce := contractErr(t, err)
		if ce.Code != tc.Expected.ActivationErrorCode || ce.Field != tc.Expected.ActivationErrorField {
			t.Fatalf("unexpected activation error: %#v", ce)
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

func loadFixture(t *testing.T, name string) replayFixture {
	t.Helper()
	path := filepath.Join(repoRoot(t), "integration", "testdata", "adapter-contract-replay", adapterprofile.CurrentProfile, name)
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
	if len(fixture.Cases) == 0 {
		t.Fatalf("fixture %s has no cases", path)
	}
	return fixture
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
