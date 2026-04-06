package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCoreModuleReadmeRichnessBaseline(t *testing.T) {
	root := repoRoot(t)
	rootReadme := mustReadContent(t, filepath.Join(root, "README.md"))

	moduleReadmes := map[string]string{}
	for _, rel := range coveredModuleReadmes {
		path := filepath.Join(root, filepath.FromSlash(rel))
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read module readme %s: %v", rel, err)
		}
		moduleReadmes[filepath.ToSlash(rel)] = string(raw)
	}

	issues := ValidateCoreModuleReadmeRichness(rootReadme, moduleReadmes)
	if len(issues) > 0 {
		lines := make([]string, 0, len(issues))
		for _, issue := range issues {
			lines = append(lines, "["+issue.Code+"] "+issue.Message)
		}
		t.Fatalf("[module-readme-richness] docs conflict:\n%s", strings.Join(lines, "\n"))
	}
}

func TestValidateCoreModuleReadmeRichnessDetectsMissingSection(t *testing.T) {
	rootReadme := strings.Join([]string{
		"docs",
		"a2a/README.md",
		"core/runner/README.md",
	}, "\n")

	moduleReadmes := map[string]string{
		"a2a/README.md": strings.Join([]string{
			"# a2a",
			"## 功能域",
			"- ok",
			"## 架构设计",
			"- ok",
			"## 关键入口",
			"- ok",
			"## 边界与依赖",
			"- ok",
			"## 配置与默认值",
			"N/A",
			"## 可观测性与验证",
			"N/A",
		}, "\n"),
	}

	issues := ValidateCoreModuleReadmeRichness(rootReadme, moduleReadmes)
	if len(issues) == 0 {
		t.Fatal("expected missing section and missing module issues")
	}

	hasMissingSection := false
	hasMissingReadme := false
	for _, issue := range issues {
		if issue.Code == "module-readme-richness.missing-section" {
			hasMissingSection = true
		}
		if issue.Code == "module-readme-richness.readme-missing" {
			hasMissingReadme = true
		}
	}
	if !hasMissingSection {
		t.Fatalf("expected missing-section issue, got %#v", issues)
	}
	if !hasMissingReadme {
		t.Fatalf("expected readme-missing issue, got %#v", issues)
	}
}

func TestValidateCoreModuleReadmeRichnessDetectsCanonicalPathDrift(t *testing.T) {
	rootReadme := strings.Join([]string{
		"docs/development-roadmap.md",
		"docs/runtime-module-boundaries.md",
		"docs/mainline-contract-test-index.md",
		"docs/runtime-config-diagnostics.md",
		// Intentionally missing docs/runtime-harness-architecture.md
		"a2a/README.md",
		"core/runner/README.md",
		"core/types/README.md",
		"tool/local/README.md",
		"mcp/README.md",
		"model/README.md",
		"context/README.md",
		"orchestration/README.md",
		"adapter/README.md",
		"runtime/config/README.md",
		"runtime/diagnostics/README.md",
		"runtime/security/README.md",
		"observability/README.md",
		"skill/loader/README.md",
	}, "\n")

	moduleReadmes := map[string]string{}
	for _, rel := range coveredModuleReadmes {
		moduleReadmes[filepath.ToSlash(rel)] = strings.Join([]string{
			"# module",
			"## 功能域",
			"- ok",
			"## 架构设计",
			"- ok",
			"## 关键入口",
			"- ok",
			"## 边界与依赖",
			"- ok",
			"## 配置与默认值",
			"- ok",
			"## 可观测性与验证",
			"- ok",
			"## 扩展点与常见误用",
			"- ok",
			// Intentionally missing canonical runtime harness doc link
		}, "\n")
	}

	issues := ValidateCoreModuleReadmeRichness(rootReadme, moduleReadmes)
	if len(issues) == 0 {
		t.Fatal("expected canonical-doc issues")
	}

	hasRootCanonicalIssue := false
	hasModuleCanonicalIssue := false
	for _, issue := range issues {
		if issue.Code == "module-readme-richness.root-canonical-doc-missing" {
			hasRootCanonicalIssue = true
		}
		if issue.Code == "module-readme-richness.module-canonical-doc-missing" {
			hasModuleCanonicalIssue = true
		}
	}
	if !hasRootCanonicalIssue {
		t.Fatalf("expected root canonical-doc issue, got %#v", issues)
	}
	if !hasModuleCanonicalIssue {
		t.Fatalf("expected module canonical-doc issue, got %#v", issues)
	}
}
