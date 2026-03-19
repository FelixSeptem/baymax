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
