package contributioncheck

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type DocumentationNavigationIssue struct {
	Code    string
	Message string
}

var requiredDocumentationPaths = []string{
	"README.md",
	"docs/README.md",
	"docs/architecture/overview.md",
	"docs/architecture/execution-model.md",
	"docs/architecture/design-principles.md",
	"docs/architecture/runtime-lifecycle.md",
	"docs/architecture/data-event-flow.md",
	"docs/architecture/module-ownership.md",
	"docs/guides/getting-started.md",
	"docs/guides/integration.md",
	"docs/guides/testing-replay-gates.md",
	"docs/documentation-ownership.md",
	"docs/api-reference-d1.md",
	"memory/README.md",
	"host/eventadapter/README.md",
	"context/handoff/README.md",
	"context/hostprojection/README.md",
	"integration/README.md",
}

var requiredReaderPathLinks = []string{
	"docs/architecture/overview.md",
	"docs/architecture/execution-model.md",
	"docs/guides/getting-started.md",
	"docs/guides/integration.md",
	"docs/guides/testing-replay-gates.md",
	"docs/api-reference-d1.md",
}

var verificationScriptPattern = regexp.MustCompile(`scripts/[A-Za-z0-9._-]+\.(?:ps1|sh)`)
var markdownLinkPattern = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)(?:\s+["'][^)]*["'])?\)`)

func ValidateDocumentationNavigation(root string, readme, index, sourceMap string) []DocumentationNavigationIssue {
	issues := make([]DocumentationNavigationIssue, 0)
	for _, rel := range requiredDocumentationPaths {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			issues = append(issues, DocumentationNavigationIssue{
				Code:    "documentation-navigation.missing-canonical-path",
				Message: fmt.Sprintf("required documentation path is missing: %s", rel),
			})
		}
	}
	for _, link := range requiredReaderPathLinks {
		if !strings.Contains(readme, link) && !strings.Contains(index, link) {
			issues = append(issues, DocumentationNavigationIssue{
				Code:    "documentation-navigation.reader-path-link-missing",
				Message: fmt.Sprintf("README or docs index is missing reader-path link: %s", link),
			})
		}
	}
	for _, rel := range requiredDocumentationPaths {
		if rel == "README.md" || rel == "docs/README.md" {
			continue
		}
		if !strings.Contains(sourceMap, rel) {
			issues = append(issues, DocumentationNavigationIssue{
				Code:    "documentation-navigation.source-map-path-missing",
				Message: fmt.Sprintf("documentation ownership map is missing canonical path: %s", rel),
			})
		}
	}
	return issues
}

func ValidateStaleDocumentationStatus(documents map[string]string) []DocumentationNavigationIssue {
	issues := make([]DocumentationNavigationIssue, 0)
	for rel, content := range documents {
		for _, marker := range []string{
			"状态：进行中",
			"本 active change",
			"（提案中）",
			"（进行中 change",
			"（进行中提案；",
			"（进行中，",
			"交付门禁进行中",
			"is an active proposal",
			"(in progress)",
			"Active host-context assembly boundary",
			"active host-context bridge",
			"task 6.3's fresh successful suite execution remains pending",
		} {
			if !strings.Contains(content, marker) {
				continue
			}
			issues = append(issues, DocumentationNavigationIssue{
				Code:    "documentation-navigation.stale-status-wording",
				Message: fmt.Sprintf("documentation contains stale status wording %q: %s", marker, rel),
			})
		}
	}
	return issues
}

func ValidateDocumentationSourceMap(root, sourceMap string) []DocumentationNavigationIssue {
	issues := make([]DocumentationNavigationIssue, 0)
	for _, raw := range strings.Split(sourceMap, "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "|") || strings.HasPrefix(line, "| ---") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 6 {
			issues = append(issues, DocumentationNavigationIssue{
				Code:    "documentation-navigation.source-map-row-invalid",
				Message: "source map row must include topic, source, entry, owner and verification: " + line,
			})
			continue
		}
		verification := strings.TrimSpace(cells[len(cells)-2])
		if verification == "" || verification == "—" {
			issues = append(issues, DocumentationNavigationIssue{
				Code:    "documentation-navigation.verification-command-missing",
				Message: "source map row has no verification command: " + line,
			})
		}
		for _, script := range verificationScriptPattern.FindAllString(verification, -1) {
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(script))); err != nil {
				issues = append(issues, DocumentationNavigationIssue{
					Code:    "documentation-navigation.verification-command-missing",
					Message: fmt.Sprintf("source map verification script does not exist: %s", script),
				})
			}
		}
	}
	return issues
}

// ValidateDocumentationLinks checks repository-relative Markdown links without
// treating external URLs or fragment-only references as repository paths.
func ValidateDocumentationLinks(root string, documents []string) []DocumentationNavigationIssue {
	issues := make([]DocumentationNavigationIssue, 0)
	for _, rel := range documents {
		path := filepath.Join(root, filepath.FromSlash(rel))
		content, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, DocumentationNavigationIssue{
				Code:    "documentation-navigation.document-missing",
				Message: fmt.Sprintf("document listed for link validation is missing: %s", rel),
			})
			continue
		}
		for _, match := range markdownLinkPattern.FindAllSubmatch(content, -1) {
			raw := strings.TrimSpace(string(match[1]))
			if raw == "" || strings.HasPrefix(raw, "#") || strings.HasPrefix(raw, "//") {
				continue
			}
			parsed, err := url.Parse(raw)
			if err != nil || parsed.IsAbs() || parsed.Scheme != "" {
				continue
			}
			link := parsed.Path
			if link == "" {
				continue
			}
			link, err = url.PathUnescape(link)
			if err != nil {
				issues = append(issues, DocumentationNavigationIssue{
					Code:    "documentation-navigation.relative-link-malformed",
					Message: fmt.Sprintf("documentation link has malformed escaping in %s: %s", rel, raw),
				})
				continue
			}
			target := filepath.Join(filepath.Dir(path), filepath.FromSlash(link))
			if _, err := os.Stat(target); err != nil {
				issues = append(issues, DocumentationNavigationIssue{
					Code:    "documentation-navigation.relative-link-missing",
					Message: fmt.Sprintf("documentation link target does not exist: %s -> %s", rel, raw),
				})
			}
		}
	}
	return issues
}
