package scenariosimulation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionedScenarioFixturesNormalizeAndRejectPrivacy(t *testing.T) {
	read := func(name string) Scenario {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatalf("read fixture %s: %v", name, err)
		}
		if len(data) > MaxSerializedBytes {
			t.Fatalf("fixture %s exceeds size bound", name)
		}
		var scenario Scenario
		if err := json.Unmarshal(data, &scenario); err != nil {
			t.Fatalf("decode fixture %s: %v", name, err)
		}
		return scenario
	}
	valid := read("scenario-success.json")
	if _, digest, err := NormalizeScenario(valid); err != nil || digest == "" {
		t.Fatalf("valid fixture normalize digest=%q err=%v", digest, err)
	}
	missing := read("scenario-evidence-missing.json")
	if _, _, err := NormalizeScenario(missing); err != nil {
		t.Fatalf("missing evidence is a valid plan, got %v", err)
	}
	privacy := read("scenario-privacy-negative.json")
	if _, _, err := NormalizeScenario(privacy); err == nil || !strings.Contains(err.Error(), ReasonPrivacyViolation) {
		t.Fatalf("privacy fixture error=%v", err)
	}
	for _, name := range []string{"scenario-causation-drift.json", "scenario-evidence-conflict.json", "scenario-outcome-indeterminate.json", "scenario-admission-drift.json"} {
		if _, _, err := NormalizeScenario(read(name)); err != nil {
			t.Fatalf("normalize fixture %s: %v", name, err)
		}
	}
	if _, _, err := NormalizeScenario(read("scenario-malformed.json")); err == nil || !strings.Contains(err.Error(), ReasonScenarioSchemaDrift) {
		t.Fatalf("malformed fixture error=%v", err)
	}
}
