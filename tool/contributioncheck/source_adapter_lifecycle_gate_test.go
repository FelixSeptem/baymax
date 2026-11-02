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

func TestSourceAdapterGateAndOwnership(t *testing.T) {
	root := repoRoot(t)
	files, err := filepath.Glob(filepath.Join(root, "host/sourceadapter/*.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("source adapter missing")
	}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(strings.Split(p, "/")[0], ".") {
				t.Fatalf("source adapter must use standard library only: %s", p)
			}
		}
	}
	raw, err := os.ReadFile(filepath.Join(root, "tool/diagnosticsreplay/source_adapter_lifecycle.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{".Poll(", ".Consume(", ".RunOnce(", ".Offer(", "OpenFileStore(", "NewMemoryStore(", ".Record(", "go func"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("replay side effect: %s", forbidden)
		}
	}
	for _, ext := range []string{"ps1", "sh"} {
		p := "scripts/check-source-adapter-lifecycle-contract." + ext
		b, err := os.ReadFile(filepath.Join(root, p))
		if err != nil {
			t.Fatal(err)
		}
		for _, marker := range []string{"./integration", "TestSourceAdapter", "source_adapter_lifecycle.invalid.json", "source_adapter.replay_drift", "duplicate_no_effect", "restart_confirmed_cursor", "abort_preserved"} {
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
