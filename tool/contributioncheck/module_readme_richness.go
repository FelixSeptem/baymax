package contributioncheck

import (
	"path/filepath"
	"strings"
)

type ModuleReadmeIssue struct {
	Code    string
	Message string
}

var coveredModuleReadmes = []string{
	"a2a/README.md",
	"core/runner/README.md",
	"core/types/README.md",
	"tool/local/README.md",
	"mcp/README.md",
	"model/README.md",
	"context/README.md",
	"orchestration/README.md",
	"runtime/config/README.md",
	"runtime/diagnostics/README.md",
	"runtime/security/README.md",
	"observability/README.md",
	"skill/loader/README.md",
}

var requiredModuleSections = []string{
	"## 功能域",
	"## 架构设计",
	"## 关键入口",
	"## 边界与依赖",
	"## 配置与默认值",
	"## 可观测性与验证",
	"## 扩展点与常见误用",
}

func ValidateCoreModuleReadmeRichness(rootReadme string, moduleReadmes map[string]string) []ModuleReadmeIssue {
	issues := make([]ModuleReadmeIssue, 0)

	for _, rel := range coveredModuleReadmes {
		if !strings.Contains(rootReadme, rel) {
			issues = append(issues, ModuleReadmeIssue{
				Code:    "module-readme-richness.root-link-missing",
				Message: "root README missing module link: " + rel,
			})
		}
		content, ok := moduleReadmes[filepath.ToSlash(rel)]
		if !ok {
			issues = append(issues, ModuleReadmeIssue{
				Code:    "module-readme-richness.readme-missing",
				Message: "covered module README missing: " + rel,
			})
			continue
		}
		for _, section := range requiredModuleSections {
			if !strings.Contains(content, section) {
				issues = append(issues, ModuleReadmeIssue{
					Code:    "module-readme-richness.missing-section",
					Message: rel + " missing section: " + section,
				})
				continue
			}
			sectionBody := sectionBody(content, section)
			if strings.TrimSpace(sectionBody) == "" {
				issues = append(issues, ModuleReadmeIssue{
					Code:    "module-readme-richness.empty-section",
					Message: rel + " has empty section body: " + section,
				})
			}
		}
	}
	return issues
}

func sectionBody(content, section string) string {
	start := strings.Index(content, section)
	if start < 0 {
		return ""
	}
	rest := content[start+len(section):]
	next := strings.Index(rest, "\n## ")
	if next < 0 {
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:next])
}
