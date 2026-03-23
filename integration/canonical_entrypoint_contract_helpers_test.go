package integration

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func integrationRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve runtime caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), ".."))
}

func mustReadIntegrationFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}

func findLegacyInvokeQualifiedUsages(t *testing.T, root string) []string {
	t.Helper()
	hits := make([]string, 0)
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".gocache" || name == "vendor" || name == "openspec" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		slashPath := filepath.ToSlash(path)
		if strings.Contains(slashPath, "/integration/canonical_entrypoint_contract_helpers_test.go") ||
			strings.Contains(slashPath, "/tool/contributioncheck/canonical_invoke_test.go") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		content := string(raw)
		if strings.Contains(content, "invoke.InvokeSync(") || strings.Contains(content, "invoke.InvokeAsync(") {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				rel = path
			}
			hits = append(hits, filepath.ToSlash(rel))
		}
		return nil
	})
	return hits
}
