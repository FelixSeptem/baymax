package contributioncheck

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestAttentionGateAndOwnership(t *testing.T) {
	root := repoRoot(t)
	f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "runtime/diagnostics/attention.go"), nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range f.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.Split(p, "/")[0], ".") || p == "os" || p == "net" || p == "net/http" {
			t.Fatalf("projection cannot read or execute owners: %s", p)
		}
	}
	for _, ext := range []string{"ps1", "sh"} {
		p := "scripts/check-attention-diagnostics-contract." + ext
		b, err := os.ReadFile(filepath.Join(root, p))
		if err != nil {
			t.Fatal(err)
		}
		for _, marker := range []string{"./integration", "TestAttention", "attention.invalid.json", "attention.replay_drift", "healthy_retrying_ignored", "effect_pending_explicit", "truncation_visible"} {
			if !strings.Contains(string(b), marker) {
				t.Fatalf("%s missing %s", p, marker)
			}
		}
		q, err := os.ReadFile(filepath.Join(root, "scripts/check-quality-gate."+ext))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(q), p) {
			t.Fatalf("unwired gate: %s", p)
		}
	}
}
