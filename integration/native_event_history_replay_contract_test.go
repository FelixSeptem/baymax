package integration

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/FelixSeptem/baymax/tool/diagnosticsreplay"
)

func TestNativeHistoryDiagnosticsReplayUsesReferenceOnlyFixture(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(file), "testdata", "diagnostics-replay", "native-event-history", "v1", "conformance.json"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := diagnosticsreplay.ReplayNativeHistoryFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := diagnosticsreplay.ReplayNativeHistoryFixtureJSON(raw)
	if err != nil || len(first) != 14 || !reflect.DeepEqual(first, second) {
		t.Fatalf("replay first=%d second=%d err=%v", len(first), len(second), err)
	}
}
