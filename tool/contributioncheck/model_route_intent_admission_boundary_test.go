package contributioncheck

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestModelRouteIntentAdmissionBoundary(t *testing.T) {
	root := repoRoot(t)
	for _, relative := range []string{
		filepath.Join("model", "catalog", "route_intent.go"),
		filepath.Join("tool", "diagnosticsreplay", "model_route_intent_admission.go"),
	} {
		source := strings.ToLower(mustRead(t, filepath.Join(root, relative)))
		for _, forbidden := range []string{"net/http", "remote discovery", "background refresh", "credential store", "global mutable router", "raw_payload", "raw_response"} {
			if strings.Contains(source, forbidden) {
				t.Fatalf("%s contains forbidden boundary marker %q", filepath.ToSlash(relative), forbidden)
			}
		}
	}
	fixturePath := filepath.Join(root, "tool", "diagnosticsreplay", "testdata", "model_route_intent_admission.v1.json")
	raw := mustRead(t, fixturePath)
	if len(raw) > 2<<20 {
		t.Fatalf("route-intent fixture exceeds bound: %d", len(raw))
	}
	for _, forbidden := range []string{"endpoint", "sk-", "token", "raw_response", "raw_payload", "password", "secret"} {
		if strings.Contains(strings.ToLower(raw), forbidden) {
			t.Fatalf("fixture contains forbidden material %q", forbidden)
		}
	}
	var fixture struct {
		Version string            `json:"version"`
		Cases   []json.RawMessage `json:"cases"`
	}
	if err := json.Unmarshal([]byte(raw), &fixture); err != nil {
		t.Fatalf("decode route-intent fixture: %v", err)
	}
	if fixture.Version != "model_route_intent_admission.v1" || len(fixture.Cases) < 5 {
		t.Fatalf("fixture version/cases invalid: %#v", fixture)
	}
}
