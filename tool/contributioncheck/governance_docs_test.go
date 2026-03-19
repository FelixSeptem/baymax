package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPre1GovernanceDocsConsistency(t *testing.T) {
	root := repoRoot(t)

	roadmap := mustReadContent(t, filepath.Join(root, "docs", "development-roadmap.md"))
	versioning := mustReadContent(t, filepath.Join(root, "docs", "versioning-and-compatibility.md"))
	readme := mustReadContent(t, filepath.Join(root, "README.md"))

	issues := ValidatePre1GovernanceDocs(roadmap, versioning, readme)
	if len(issues) > 0 {
		t.Fatalf("[pre1-governance] docs consistency issues:\n%s", strings.Join(issues, "\n"))
	}
}

func TestValidatePre1GovernanceDocsDetectsStageConflict(t *testing.T) {
	roadmap := strings.Join([]string{
		"版本阶段口径（延续 0.x）",
		"不做 `1.0.0` / prod-ready 承诺",
		"新增提案准入规则（0.x 阶段）",
		"`Why now`",
		"风险",
		"回滚",
		"文档影响",
		"验证命令",
		"契约一致性",
		"可靠性与安全",
		"质量门禁回归治理",
		"外部接入 DX",
		"长期方向（不进入近期主线）",
		"平台化控制面",
		"跨租户全局调度与控制平面",
		"市场化/托管化 adapter registry 能力",
		"我们将在本季度进入 1.0.0 稳定发布并给出 prod-ready 承诺",
	}, "\n")
	versioning := strings.Join([]string{
		"pre-`1.0.0`",
		"does **not** imply `1.0.0/prod-ready` commitments",
		"Pre-1 Proposal Admission Baseline",
	}, "\n")
	readme := strings.Join([]string{
		"版本阶段快照",
		"`0.x` pre-1 阶段",
		"不做 `1.0.0/prod-ready` 承诺",
	}, "\n")

	issues := ValidatePre1GovernanceDocs(roadmap, versioning, readme)
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "conflicting stable-release claim in roadmap") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected stable-release conflict issue, got: %#v", issues)
	}
}

func mustReadContent(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}
