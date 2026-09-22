package contributioncheck

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModelCatalogRoutingAdmissionBoundary(t *testing.T) {
	root := repoRoot(t)
	for _, relative := range []string{
		filepath.Join("model", "catalog"),
		filepath.Join("tool", "diagnosticsreplay"),
	} {
		assertNoRoutingProviderSDKImports(t, root, relative)
	}

	forbidden := []string{
		"net/http",
		"credential store",
		"remote discovery",
		"background refresh",
		"global mutable router",
		"provider_response",
		"raw_payload",
	}
	for _, relative := range []string{
		filepath.Join("model", "catalog", "routing_admission.go"),
		filepath.Join("tool", "diagnosticsreplay", "model_catalog_routing_admission.go"),
	} {
		source := mustRead(t, filepath.Join(root, relative))
		lower := strings.ToLower(source)
		for _, needle := range forbidden {
			if strings.Contains(lower, strings.ToLower(needle)) {
				t.Fatalf("%s contains forbidden boundary marker %q", filepath.ToSlash(relative), needle)
			}
		}
	}

	fixturePath := filepath.Join(root, "tool", "diagnosticsreplay", "testdata", "model_catalog_routing_admission.v1.json")
	info, err := os.Stat(fixturePath)
	if err != nil {
		t.Fatalf("routing admission fixture missing: %v", err)
	}
	if info.Size() > 2<<20 {
		t.Fatalf("routing admission fixture exceeds bound: %d", info.Size())
	}
	var fixture struct {
		Version string            `json:"version"`
		Cases   []json.RawMessage `json:"cases"`
	}
	raw := mustRead(t, fixturePath)
	if err := json.Unmarshal([]byte(raw), &fixture); err != nil {
		t.Fatalf("decode routing admission fixture: %v", err)
	}
	if fixture.Version != "model_catalog_routing_admission.v1" || len(fixture.Cases) == 0 {
		t.Fatalf("fixture version/cases invalid: %#v", fixture)
	}
}

func assertNoRoutingProviderSDKImports(t *testing.T, root, relativeDir string) {
	t.Helper()
	forbidden := []string{
		"github.com/openai/openai-go",
		"github.com/anthropics/anthropic-sdk-go",
		"google.golang.org/genai",
	}
	err := filepath.WalkDir(filepath.Join(root, relativeDir), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return parseErr
		}
		for _, imp := range file.Imports {
			pkg := strings.Trim(imp.Path.Value, `"`)
			for _, prefix := range forbidden {
				if pkg == prefix || strings.HasPrefix(pkg, prefix+"/") {
					t.Fatalf("%s imports provider SDK %q", filepath.ToSlash(path), pkg)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan %s: %v", filepath.ToSlash(relativeDir), err)
	}
}
