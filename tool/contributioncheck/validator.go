package contributioncheck

import (
	"regexp"
	"strings"
)

type Violation struct {
	Code    string
	Message string
}

var (
	checkedBoxPattern = regexp.MustCompile(`(?i)^\s*-\s*\[(x)\]\s*(.+?)\s*$`)
)

type requiredSection struct {
	needle  string
	code    string
	message string
}

var requiredSections = []requiredSection{
	{
		needle:  "## 摘要 Summary",
		code:    "missing_section_summary",
		message: "missing required section: 摘要 Summary",
	},
	{
		needle:  "## 变更内容 Changes",
		code:    "missing_section_changes",
		message: "missing required section: 变更内容 Changes",
	},
	{
		needle:  "## 验证 Validation",
		code:    "missing_section_validation",
		message: "missing required section: 验证 Validation",
	},
	{
		needle:  "## 文档影响 Documentation",
		code:    "missing_section_documentation",
		message: "missing required section: 文档影响 Documentation",
	},
	{
		needle:  "## 变更影响 Impact",
		code:    "missing_section_impact",
		message: "missing required section: 变更影响 Impact",
	},
}

type requiredCheckbox struct {
	label   string
	code    string
	message string
}

var requiredCheckedBoxes = []requiredCheckbox{
	{
		label:   "`go test ./...`",
		code:    "missing_checkbox_go_test",
		message: "missing checked checkbox: `go test ./...`",
	},
	{
		label:   "`go test -race ./...`",
		code:    "missing_checkbox_go_test_race",
		message: "missing checked checkbox: `go test -race ./...`",
	},
	{
		label:   "`golangci-lint run --config .golangci.yml`",
		code:    "missing_checkbox_golangci_lint",
		message: "missing checked checkbox: `golangci-lint run --config .golangci.yml`",
	},
	{
		label:   "我确认以上检查已执行，或在本 PR 中说明了未执行原因",
		code:    "missing_checkbox_validation_ack",
		message: "missing checked checkbox: validation acknowledgement",
	},
}

var migrationImpactChoices = []string{
	"无迁移影响（No migration impact）",
	"有迁移影响，已在 `CHANGELOG.md` 或 PR 说明中给出迁移指引",
}

func ValidatePullRequestBody(body string) []Violation {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return []Violation{{
			Code:    "empty_pr_body",
			Message: "pull request body is empty",
		}}
	}

	violations := make([]Violation, 0)

	for _, section := range requiredSections {
		if !strings.Contains(body, section.needle) {
			violations = append(violations, Violation{
				Code:    section.code,
				Message: section.message,
			})
		}
	}

	checkedLabels := checkedCheckboxLabels(body)
	for _, checkbox := range requiredCheckedBoxes {
		if _, ok := checkedLabels[normalizeLabel(checkbox.label)]; !ok {
			violations = append(violations, Violation{
				Code:    checkbox.code,
				Message: checkbox.message,
			})
		}
	}

	if !hasCheckedAny(checkedLabels, migrationImpactChoices) {
		violations = append(violations, Violation{
			Code:    "missing_checkbox_migration_impact_choice",
			Message: "missing checked checkbox: migration impact choice",
		})
	}

	return violations
}

func checkedCheckboxLabels(body string) map[string]struct{} {
	labels := make(map[string]struct{})
	for _, line := range strings.Split(body, "\n") {
		match := checkedBoxPattern.FindStringSubmatch(line)
		if len(match) != 3 {
			continue
		}
		labels[normalizeLabel(match[2])] = struct{}{}
	}
	return labels
}

func hasCheckedAny(checkedLabels map[string]struct{}, options []string) bool {
	for _, option := range options {
		if _, ok := checkedLabels[normalizeLabel(option)]; ok {
			return true
		}
	}
	return false
}

func normalizeLabel(raw string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(raw))), " ")
}
