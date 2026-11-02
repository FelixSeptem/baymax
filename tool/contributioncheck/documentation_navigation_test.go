package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentationNavigationBaseline(t *testing.T) {
	root := repoRoot(t)
	readme := mustReadContent(t, filepath.Join(root, "README.md"))
	index := mustReadContent(t, filepath.Join(root, "docs", "README.md"))
	sourceMap := mustReadContent(t, filepath.Join(root, "docs", "documentation-ownership.md"))

	issues := ValidateDocumentationNavigation(root, readme, index, sourceMap)
	if len(issues) > 0 {
		t.Fatalf("[documentation-navigation] docs conflict: %#v", issues)
	}
	if issues := ValidateDocumentationSourceMap(root, sourceMap); len(issues) > 0 {
		t.Fatalf("[documentation-navigation] source map conflict: %#v", issues)
	}
	if issues := ValidateDocumentationLinks(root, publicDocumentationPaths(t, root)); len(issues) > 0 {
		t.Fatalf("[documentation-navigation] relative link conflict: %#v", issues)
	}
}

func TestValidateDocumentationLinksDetectsMissingTarget(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs", "README.md")
	if err := ensureParent(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[missing](missing.md)\n[external](https://example.com/docs)\n# local\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	issues := ValidateDocumentationLinks(root, []string{"docs/README.md"})
	if len(issues) != 1 || issues[0].Code != "documentation-navigation.relative-link-missing" {
		t.Fatalf("expected one missing relative link issue, got %#v", issues)
	}
}

func TestValidateDocumentationSourceMapDetectsMissingVerificationScript(t *testing.T) {
	sourceMap := "| Topic | Source | Entry | Owner | Verification |\n| --- | --- | --- | --- | --- |\n| docs | `docs/README.md` | `docs/README.md` | owner | `scripts/missing-check.ps1` |"
	issues := ValidateDocumentationSourceMap(t.TempDir(), sourceMap)
	if len(issues) != 1 || issues[0].Code != "documentation-navigation.verification-command-missing" {
		t.Fatalf("expected missing verification command issue, got %#v", issues)
	}
}

func TestValidateDocumentationNavigationDetectsMissingCanonicalPath(t *testing.T) {
	root := t.TempDir()
	readme := "docs/architecture/overview.md"
	index := "docs/guides/integration.md"
	sourceMap := "docs/architecture/overview.md"

	issues := ValidateDocumentationNavigation(root, readme, index, sourceMap)
	if len(issues) == 0 {
		t.Fatal("expected missing canonical path and source-map issues")
	}
	seen := false
	for _, issue := range issues {
		if issue.Code == "documentation-navigation.missing-canonical-path" {
			seen = true
		}
	}
	if !seen {
		t.Fatalf("expected missing-canonical-path issue, got %#v", issues)
	}
}

func TestValidateStaleDocumentationStatusDetectsMarkers(t *testing.T) {
	issues := ValidateStaleDocumentationStatus(map[string]string{
		"orchestration/README.md": "## Application outbox（进行中 change）",
	})
	if len(issues) != 1 || issues[0].Code != "documentation-navigation.stale-status-wording" {
		t.Fatalf("expected stale-status issue, got %#v", issues)
	}
}

func TestCurrentDocumentationHasNoStaleStatusWording(t *testing.T) {
	root := repoRoot(t)
	documents := map[string]string{}
	for _, rel := range publicDocumentationPaths(t, root) {
		documents[rel] = mustReadContent(t, filepath.Join(root, filepath.FromSlash(rel)))
	}
	if issues := ValidateStaleDocumentationStatus(documents); len(issues) > 0 {
		t.Fatalf("[documentation-navigation] stale status wording: %#v", issues)
	}
}

func TestPublicIndexesExposeAPIAndOwnerDocumentation(t *testing.T) {
	root := repoRoot(t)
	readme := mustReadContent(t, filepath.Join(root, "README.md"))
	index := mustReadContent(t, filepath.Join(root, "docs", "README.md"))
	sourceMap := mustReadContent(t, filepath.Join(root, "docs", "documentation-ownership.md"))

	for _, rel := range []string{
		"docs/api-reference-d1.md",
		"memory/README.md",
		"host/eventadapter/README.md",
		"context/handoff/README.md",
		"context/hostprojection/README.md",
		"integration/README.md",
	} {
		if !strings.Contains(readme, rel) && !strings.Contains(index, rel) && !strings.Contains(sourceMap, rel) {
			t.Errorf("public documentation indexes do not expose canonical path %s", rel)
		}
	}
}

func TestValidateStaleDocumentationStatusDetectsArchivedHostContextWording(t *testing.T) {
	issues := ValidateStaleDocumentationStatus(map[string]string{
		"context/handoff/README.md": "## Active host-context assembly boundary\nThe task 6.3's fresh successful suite execution remains pending.",
	})
	if len(issues) != 2 {
		t.Fatalf("expected two archived-status issues, got %#v", issues)
	}
}

func TestValidateDocumentationNavigationAcceptsCompleteFixture(t *testing.T) {
	root := t.TempDir()
	for _, rel := range requiredDocumentationPaths {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := ensureParent(path); err != nil {
			t.Fatal(err)
		}
	}
	links := strings.Join(requiredReaderPathLinks, "\n")
	issues := ValidateDocumentationNavigation(root, links, links, strings.Join(requiredDocumentationPaths, "\n"))
	if len(issues) > 0 {
		t.Fatalf("expected complete fixture to pass, got %#v", issues)
	}
}

func ensureParent(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, nil, 0o644)
}

func publicDocumentationPaths(t *testing.T, root string) []string {
	t.Helper()
	paths := []string{"README.md"}
	for _, directory := range []string{"docs", "examples", "core", "orchestration", "host", "integration", "model", "tool", "mcp", "context", "adapter", "observability", "memory", "a2a"} {
		err := filepath.WalkDir(filepath.Join(root, directory), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if entry.Name() == "superpowers" || entry.Name() == "archive" {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".md" {
				return nil
			}
			if entry.Name() != "README.md" && !strings.HasPrefix(filepath.ToSlash(path), filepath.ToSlash(filepath.Join(root, "docs"))) {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			paths = append(paths, filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return paths
}
