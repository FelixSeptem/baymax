package contributioncheck

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestContextPackagesDoNotDirectlyImportProviderSDKs(t *testing.T) {
	root := repoRoot(t)
	contextRoot := filepath.Join(root, "context")
	forbidden := []string{
		"github.com/openai/openai-go",
		"github.com/anthropics/anthropic-sdk-go",
		"google.golang.org/genai",
	}
	// Baseline allowlist: existing legacy embedding adapter still needs migration to model/*.
	allowedLegacyFiles := map[string]struct{}{
		filepath.ToSlash(filepath.Join("context", "assembler", "embedding_adapter.go")): {},
	}

	walkErr := filepath.WalkDir(contextRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if _, ok := allowedLegacyFiles[rel]; ok {
			return nil
		}
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return parseErr
		}
		for _, imp := range file.Imports {
			pkg := strings.TrimSpace(strings.Trim(imp.Path.Value, `"`))
			if slices.Contains(forbidden, pkg) {
				t.Fatalf("context boundary violation: %s directly imports provider sdk %q", rel, pkg)
			}
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("scan context package imports failed: %v", walkErr)
	}
}
