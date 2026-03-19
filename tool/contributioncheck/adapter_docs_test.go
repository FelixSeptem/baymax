package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdapterOnboardingDocsConsistency(t *testing.T) {
	root := repoRoot(t)

	requiredDocs := []string{
		"docs/external-adapter-template-index.md",
		"docs/adapter-migration-mapping.md",
	}
	for _, rel := range requiredDocs {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("[adapter-docs] missing required adapter doc file: %s (%v)", rel, err)
		}
	}

	readme := mustRead(t, filepath.Join(root, "README.md"))
	apiRef := mustRead(t, filepath.Join(root, "docs", "api-reference-d1.md"))
	for _, rel := range requiredDocs {
		if !strings.Contains(readme, rel) {
			t.Fatalf("[adapter-docs] README missing adapter doc link: %s", rel)
		}
		if !strings.Contains(apiRef, rel) {
			t.Fatalf("[adapter-docs] docs/api-reference-d1.md missing adapter doc link: %s", rel)
		}
	}

	categoryMarkers := []string{
		"MCP adapter template",
		"Model provider adapter template",
		"Tool adapter template",
	}
	for _, marker := range categoryMarkers {
		if !strings.Contains(apiRef, marker) {
			t.Fatalf("[adapter-docs] docs/api-reference-d1.md missing onboarding marker: %s", marker)
		}
	}

	templateIndex := mustRead(t, filepath.Join(root, "docs", "external-adapter-template-index.md"))
	for _, marker := range []string{
		"MCP adapter template",
		"Model provider adapter template",
		"Tool adapter template",
		"onboarding skeleton",
		"check-adapter-conformance.ps1",
		"check-adapter-conformance.sh",
	} {
		if !strings.Contains(templateIndex, marker) {
			t.Fatalf("[adapter-docs] docs/external-adapter-template-index.md missing marker: %s", marker)
		}
	}

	mappingDoc := mustRead(t, filepath.Join(root, "docs", "adapter-migration-mapping.md"))
	for _, marker := range []string{
		"capability-domain",
		"code-snippet",
		"previous pattern",
		"recommended pattern",
		"compatibility notes",
		"additive + nullable + default + fail-fast",
		"check-adapter-conformance.ps1",
		"check-adapter-conformance.sh",
	} {
		if !strings.Contains(mappingDoc, marker) {
			t.Fatalf("[adapter-docs] docs/adapter-migration-mapping.md missing marker: %s", marker)
		}
	}

	for _, rel := range []string{
		"scripts/check-adapter-conformance.sh",
		"scripts/check-adapter-conformance.ps1",
	} {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("[adapter-docs] missing adapter conformance script: %s (%v)", rel, err)
		}
	}
}
