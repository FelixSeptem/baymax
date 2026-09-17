package extensionauthoring

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/FelixSeptem/baymax/extension"
)

func TestReplayContractExternalExtensionAuthoringFixtures(t *testing.T) {
	root := filepath.Join("..", "testdata", "diagnostics-replay", "external-extension-authoring", "v1")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) < 6 {
		t.Fatalf("fixture count=%d, want at least 6", len(names))
	}
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		var envelope struct {
			Case     string                    `json:"case"`
			Expected extension.AuthoringResult `json:"expected"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		got, err := extension.EvaluateAuthoringCase(raw)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !reflect.DeepEqual(got, envelope.Expected) {
			t.Errorf("%s: result=%#v want=%#v", name, got, envelope.Expected)
		}
		gotAgain, _ := extension.EvaluateAuthoringCase(raw)
		if !reflect.DeepEqual(got, gotAgain) {
			t.Errorf("%s: replay is not idempotent", name)
		}
	}
}

func TestReplayContractExternalExtensionAuthoringRunStreamProjectionParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "testdata", "diagnostics-replay", "external-extension-authoring", "v1", "valid-activation.json"))
	if err != nil {
		t.Fatal(err)
	}
	run, err := extension.EvaluateAuthoringProjection(raw, extension.AuthoringProjectionRun)
	if err != nil {
		t.Fatal(err)
	}
	stream, err := extension.EvaluateAuthoringProjection(raw, extension.AuthoringProjectionStream)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(run, stream) {
		t.Fatalf("run=%#v stream=%#v", run, stream)
	}
}
