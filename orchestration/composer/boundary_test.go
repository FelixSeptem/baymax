package composer

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestComposerAndSchedulerDoNotImportRuntimeDiagnosticsDirectly(t *testing.T) {
	repoRoot := locateRepoRoot(t)
	checkNoDirectImport(t, filepath.Join(repoRoot, "orchestration", "composer"))
	checkNoDirectImport(t, filepath.Join(repoRoot, "orchestration", "scheduler"))
}

func checkNoDirectImport(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %q: %v", dir, err)
	}
	target := "\"github.com/FelixSeptem/baymax/runtime/diagnostics\""
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		full := filepath.Join(dir, name)
		fileSet := token.NewFileSet()
		node, err := parser.ParseFile(fileSet, full, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse imports %q: %v", full, err)
		}
		for _, imp := range node.Imports {
			if imp == nil {
				continue
			}
			if strings.TrimSpace(imp.Path.Value) == target {
				t.Fatalf("boundary violation: %s imports runtime/diagnostics directly", full)
			}
		}
	}
}

func locateRepoRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}
