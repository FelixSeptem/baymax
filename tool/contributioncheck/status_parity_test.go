package contributioncheck

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseStatusParityDocsConsistency(t *testing.T) {
	root := repoRoot(t)
	active, archived, err := LoadOpenSpecStatusAuthority(root)
	if err != nil {
		t.Fatalf("load openspec status authority: %v", err)
	}

	roadmap := mustReadContent(t, filepath.Join(root, "docs", "development-roadmap.md"))
	readme := mustReadContent(t, filepath.Join(root, "README.md"))

	issues := ValidateStatusParity(active, archived, roadmap, readme)
	if len(issues) > 0 {
		lines := make([]string, 0, len(issues))
		for _, issue := range issues {
			lines = append(lines, "["+issue.Code+"] "+issue.Message)
		}
		t.Fatalf("[status-parity] docs conflict:\n%s", strings.Join(lines, "\n"))
	}
}

func TestValidateStatusParityDetectsConflict(t *testing.T) {
	active := []string{"change-foo-a25"}
	archived := []string{"change-bar-a24"}
	roadmap := strings.Join([]string{
		"- 进行中：",
		"  - `change-bar-a24`",
	}, "\n")
	readme := strings.Join([]string{
		"- `change-bar-a24`（bar）进行中。",
		"- `change-foo-a25`（foo）已归档并稳定。",
	}, "\n")

	issues := ValidateStatusParity(active, archived, roadmap, readme)
	if len(issues) == 0 {
		t.Fatal("expected conflict issues")
	}

	requiredCodes := map[string]bool{
		"status-parity.snapshot-missing-active-change": false,
		"status-parity.active-vs-archived-mismatch":    false,
		"status-parity.stale-snapshot-mismatch":        false,
	}
	for _, issue := range issues {
		if _, ok := requiredCodes[issue.Code]; ok {
			requiredCodes[issue.Code] = true
		}
	}
	for code, seen := range requiredCodes {
		if !seen {
			t.Fatalf("expected issue code %s, got %#v", code, issues)
		}
	}
}

func TestValidateStatusParitySupportsSlugSnapshotFormat(t *testing.T) {
	active := []string{"introduce-foo-contract-a25"}
	archived := []string{"introduce-bar-contract-a24"}
	roadmap := strings.Join([]string{
		"- 进行中：",
		"  - `introduce-foo-contract-a25`",
	}, "\n")
	readme := strings.Join([]string{
		"- `introduce-foo-contract-a25`（foo）进行中。",
		"- `introduce-bar-contract-a24`（bar）已归档并稳定。",
	}, "\n")

	issues := ValidateStatusParity(active, archived, roadmap, readme)
	if len(issues) > 0 {
		t.Fatalf("expected no parity issues for slug snapshot format, got %#v", issues)
	}
}
