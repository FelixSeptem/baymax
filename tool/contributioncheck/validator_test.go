package contributioncheck

import (
	"reflect"
	"testing"
)

func TestValidatePullRequestBodyPass(t *testing.T) {
	body := `## 摘要 Summary（必填）
English summary is acceptable.

## 变更内容 Changes（必填）
- update governance policy docs

## 验证 Validation（必填）
- [x] ` + "`go test ./...`" + `
- [X] ` + "`go test -race ./...`" + `
- [x] ` + "`golangci-lint run --config .golangci.yml`" + `
- [x] 我确认以上检查已执行，或在本 PR 中说明了未执行原因

## 文档影响 Documentation（必填）
updated docs/versioning-and-compatibility.md

## 变更影响 Impact（必填）
- [x] 无迁移影响（No migration impact）
- [ ] 有迁移影响，已在 ` + "`CHANGELOG.md`" + ` 或 PR 说明中给出迁移指引
`

	violations := ValidatePullRequestBody(body)
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got: %+v", violations)
	}
}

func TestValidatePullRequestBodyDeterministicFailures(t *testing.T) {
	body := `## 摘要 Summary（必填）
only summary provided
`

	violations := ValidatePullRequestBody(body)
	gotCodes := make([]string, 0, len(violations))
	for _, v := range violations {
		gotCodes = append(gotCodes, v.Code)
	}
	wantCodes := []string{
		"missing_section_changes",
		"missing_section_validation",
		"missing_section_documentation",
		"missing_section_impact",
		"missing_checkbox_go_test",
		"missing_checkbox_go_test_race",
		"missing_checkbox_golangci_lint",
		"missing_checkbox_validation_ack",
		"missing_checkbox_migration_impact_choice",
	}

	if !reflect.DeepEqual(gotCodes, wantCodes) {
		t.Fatalf("codes mismatch:\n got: %v\nwant: %v", gotCodes, wantCodes)
	}
}

func TestValidatePullRequestBodyEmpty(t *testing.T) {
	violations := ValidatePullRequestBody(" \n\t ")
	if len(violations) != 1 {
		t.Fatalf("expected exactly one violation, got %d", len(violations))
	}
	if violations[0].Code != "empty_pr_body" {
		t.Fatalf("code = %q, want empty_pr_body", violations[0].Code)
	}
}
