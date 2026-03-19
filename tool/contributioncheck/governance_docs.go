package contributioncheck

import "strings"

func ValidatePre1GovernanceDocs(roadmap, versioning, readme string) []string {
	issues := make([]string, 0)

	roadmapMarkers := []string{
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
	}
	for _, marker := range roadmapMarkers {
		if !strings.Contains(roadmap, marker) {
			issues = append(issues, "roadmap missing marker: "+marker)
		}
	}

	versioningMarkers := []string{
		"pre-`1.0.0`",
		"does **not** imply `1.0.0/prod-ready` commitments",
		"Pre-1 Proposal Admission Baseline",
	}
	for _, marker := range versioningMarkers {
		if !strings.Contains(versioning, marker) {
			issues = append(issues, "versioning missing marker: "+marker)
		}
	}

	readmeMarkers := []string{
		"版本阶段快照",
		"`0.x` pre-1 阶段",
		"不做 `1.0.0/prod-ready` 承诺",
	}
	for _, marker := range readmeMarkers {
		if !strings.Contains(readme, marker) {
			issues = append(issues, "readme missing marker: "+marker)
		}
	}

	if containsStableReleaseClaim(roadmap) {
		issues = append(issues, "conflicting stable-release claim in roadmap")
	}
	if containsStableReleaseClaim(readme) {
		issues = append(issues, "conflicting stable-release claim in readme")
	}

	return issues
}

func containsStableReleaseClaim(in string) bool {
	for _, raw := range strings.Split(in, "\n") {
		line := strings.ToLower(strings.TrimSpace(raw))
		if line == "" {
			continue
		}
		if !strings.Contains(line, "1.0") &&
			!strings.Contains(line, "prod-ready") &&
			!strings.Contains(line, "stable release") {
			continue
		}
		if strings.Contains(line, "不做") ||
			strings.Contains(line, "does not") ||
			strings.Contains(line, "not imply") ||
			strings.Contains(line, "非") {
			continue
		}
		return true
	}
	return false
}
