package contributioncheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateOwnerEvidenceIndex(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "docs", "owner-evidence-index.v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	ownerMap := mustRead(t, filepath.Join(root, "docs", "documentation-ownership.md"))
	qualityPS := mustRead(t, filepath.Join(root, "scripts", "check-quality-gate.ps1"))
	qualitySH := mustRead(t, filepath.Join(root, "scripts", "check-quality-gate.sh"))
	if issues := ValidateOwnerEvidenceIndex(root, raw, ownerMap, qualityPS, qualitySH); len(issues) != 0 {
		t.Fatalf("valid index: %v", issues)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name           string
		mutate         func(map[string]any)
		ps, sh, source string
	}{
		{name: "unsupported version", mutate: func(d map[string]any) { d["version"] = "v2" }},
		{name: "duplicate id", mutate: func(d map[string]any) {
			rows := d["semantics"].([]any)
			rows[1].(map[string]any)["id"] = rows[0].(map[string]any)["id"]
		}},
		{name: "missing owner", mutate: func(d map[string]any) { firstEvidence(d)["owner"] = "" }},
		{name: "broad owner", mutate: func(d map[string]any) { firstEvidence(d)["owner"] = "runtime owners" }},
		{name: "multiple owners", mutate: func(d map[string]any) { firstEvidence(d)["owner"] = "context/hostprojection,model/conformance" }},
		{name: "missing spec", mutate: func(d map[string]any) { firstEvidence(d)["main_spec"] = "openspec/specs/missing/spec.md" }},
		{name: "missing test", mutate: func(d map[string]any) { firstEvidence(d)["tests"] = []any{"missing_test.go"} }},
		{name: "missing docs", mutate: func(d map[string]any) { firstEvidence(d)["documentation"] = []any{"docs/missing.md"} }},
		{name: "missing gate", mutate: func(d map[string]any) { firstEvidence(d)["gate"] = "" }},
		{name: "missing fixture", mutate: func(d map[string]any) {
			firstEvidence(d)["replay"] = map[string]any{"mode": "fixture", "fixture": "missing.json"}
		}},
		{name: "unjustified exemption", mutate: func(d map[string]any) {
			firstEvidence(d)["replay"] = map[string]any{"mode": "not_applicable", "reason": " "}
		}},
		{name: "owner contradiction", source: strings.Replace(ownerMap, "| `context/hostprojection` | `scripts/check-host-context", "| `model/conformance` | `scripts/check-host-context", 1)},
		{name: "powershell gate unwired", ps: strings.ReplaceAll(qualityPS, "check-host-context-assembly-handoff-contract.ps1", "removed.ps1")},
		{name: "shell gate unwired", sh: strings.ReplaceAll(qualitySH, "check-host-context-assembly-handoff-contract.sh", "removed.sh")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var copy map[string]any
			if err := json.Unmarshal(raw, &copy); err != nil {
				t.Fatal(err)
			}
			if tc.mutate != nil {
				tc.mutate(copy)
			}
			changed, err := json.Marshal(copy)
			if err != nil {
				t.Fatal(err)
			}
			ps, sh, source := qualityPS, qualitySH, ownerMap
			if tc.ps != "" {
				ps = tc.ps
			}
			if tc.sh != "" {
				sh = tc.sh
			}
			if tc.source != "" {
				source = tc.source
			}
			if issues := ValidateOwnerEvidenceIndex(root, changed, source, ps, sh); len(issues) == 0 {
				t.Fatal("expected validation failure")
			}
		})
	}
	if issues := ValidateOwnerEvidenceIndex(root, []byte(`{"version":`), ownerMap, qualityPS, qualitySH); len(issues) == 0 {
		t.Fatal("malformed JSON accepted")
	}
}

func firstEvidence(d map[string]any) map[string]any {
	return d["semantics"].([]any)[0].(map[string]any)
}

func TestOwnerEvidenceIndexGateScriptsStayPairedAndWired(t *testing.T) {
	root := repoRoot(t)
	ps := mustRead(t, filepath.Join(root, "scripts", "check-owner-evidence-index-contract.ps1"))
	sh := mustRead(t, filepath.Join(root, "scripts", "check-owner-evidence-index-contract.sh"))
	for _, content := range []string{ps, sh} {
		if !strings.Contains(content, "TestValidateOwnerEvidenceIndex") || !strings.Contains(content, "go test ./tool/contributioncheck") {
			t.Fatal("dedicated gate must run the focused validator test")
		}
	}
	if !strings.Contains(ps, "lib/native-strict.ps1") || !strings.Contains(ps, "Invoke-NativeStrict") {
		t.Fatal("PowerShell gate must use native strict execution")
	}
	if !strings.Contains(sh, "set -euo pipefail") {
		t.Fatal("shell gate must fail fast")
	}
	qualityPS := mustRead(t, filepath.Join(root, "scripts", "check-quality-gate.ps1"))
	qualitySH := mustRead(t, filepath.Join(root, "scripts", "check-quality-gate.sh"))
	if !strings.Contains(qualityPS, "check-owner-evidence-index-contract.ps1") || !strings.Contains(qualitySH, "check-owner-evidence-index-contract.sh") {
		t.Fatal("both quality gates must wire the dedicated gate")
	}
}
