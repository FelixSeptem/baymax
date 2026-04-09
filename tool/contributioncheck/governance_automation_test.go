package contributioncheck

import (
	"reflect"
	"strings"
	"testing"
)

func TestValidateRoadmapStatusConsistencyPass(t *testing.T) {
	active := []string{
		"introduce-governance-automation-and-consistency-gate-contract-a70",
		"introduce-real-runtime-agent-mode-examples-contract-a71",
	}
	archived := []string{
		"introduce-context-compression-production-hardening-contract-a69",
	}
	roadmap := strings.Join([]string{
		"## 当前状态（以代码与 OpenSpec 为准）",
		"- 已归档：",
		"  - `introduce-context-compression-production-hardening-contract-a69`",
		"- 进行中：",
		"  - `introduce-governance-automation-and-consistency-gate-contract-a70`",
		"  - `introduce-real-runtime-agent-mode-examples-contract-a71`",
		"## 版本阶段口径（延续 0.x）",
	}, "\n")

	issues := ValidateRoadmapStatusConsistency(active, archived, roadmap)
	if len(issues) != 0 {
		t.Fatalf("expected no drift issues, got %#v", issues)
	}
}

func TestValidateRoadmapStatusConsistencyDetectsDrift(t *testing.T) {
	active := []string{
		"introduce-governance-automation-and-consistency-gate-contract-a70",
	}
	archived := []string{
		"introduce-context-compression-production-hardening-contract-a69",
	}
	roadmap := strings.Join([]string{
		"## 当前状态（以代码与 OpenSpec 为准）",
		"- 已归档：",
		"  - `introduce-governance-automation-and-consistency-gate-contract-a70`",
		"  - `introduce-unknown-change-a72`",
		"- 进行中：",
		"  - `introduce-context-compression-production-hardening-contract-a69`",
		"## 版本阶段口径（延续 0.x）",
	}, "\n")

	gotA := ValidateRoadmapStatusConsistency(active, archived, roadmap)
	gotB := ValidateRoadmapStatusConsistency(active, archived, roadmap)
	if !reflect.DeepEqual(gotA, gotB) {
		t.Fatalf("issues must be deterministic:\nA=%#v\nB=%#v", gotA, gotB)
	}
	if len(gotA) == 0 {
		t.Fatal("expected drift issues")
	}

	requiredMessages := []string{
		"roadmap missing in-progress change from openspec list: introduce-governance-automation-and-consistency-gate-contract-a70",
		"roadmap marks archived change as in-progress: introduce-context-compression-production-hardening-contract-a69",
		"roadmap marks active change as archived: introduce-governance-automation-and-consistency-gate-contract-a70",
		"roadmap archived entry is not present in archive index: introduce-unknown-change-a72",
	}
	for _, expected := range requiredMessages {
		found := false
		for _, issue := range gotA {
			if issue.Code == GovernanceCodeRoadmapStatusDrift && issue.Message == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected drift message %q, got %#v", expected, gotA)
		}
	}
}

func TestValidateProposalExampleImpactDeclarationPass(t *testing.T) {
	proposal := strings.Join([]string{
		"## Why",
		"some change",
		"## Example Impact Assessment",
		"- 新增示例",
		"## What Changes",
		"details",
	}, "\n")

	issues := ValidateProposalExampleImpactDeclaration("introduce-foo-contract-a71", proposal)
	if len(issues) != 0 {
		t.Fatalf("expected no declaration issues, got %#v", issues)
	}
}

func TestValidateProposalExampleImpactDeclarationAllowsNoExampleWithReasonSuffix(t *testing.T) {
	proposal := strings.Join([]string{
		"## Why",
		"some change",
		"## Example Impact Assessment",
		"- 无需示例变更（附理由）：仅调整文档与门禁映射",
		"## What Changes",
		"details",
	}, "\n")

	issues := ValidateProposalExampleImpactDeclaration("introduce-foo-contract-a71", proposal)
	if len(issues) != 0 {
		t.Fatalf("expected no declaration issues, got %#v", issues)
	}
}

func TestValidateProposalExampleImpactDeclarationMissing(t *testing.T) {
	proposal := strings.Join([]string{
		"## Why",
		"some change",
		"## What Changes",
		"details",
	}, "\n")

	issues := ValidateProposalExampleImpactDeclaration("introduce-foo-contract-a71", proposal)
	if len(issues) != 1 {
		t.Fatalf("expected one issue, got %#v", issues)
	}
	if issues[0].Code != GovernanceCodeMissingExampleImpactDeclaration {
		t.Fatalf("unexpected code: %#v", issues)
	}
}

func TestValidateProposalExampleImpactDeclarationInvalidValue(t *testing.T) {
	proposal := strings.Join([]string{
		"## Why",
		"some change",
		"## Example Impact Assessment",
		"- 不涉及示例",
		"## What Changes",
		"details",
	}, "\n")

	issues := ValidateProposalExampleImpactDeclaration("introduce-foo-contract-a71", proposal)
	if len(issues) != 1 {
		t.Fatalf("expected one issue, got %#v", issues)
	}
	if issues[0].Code != GovernanceCodeInvalidExampleImpactValue {
		t.Fatalf("unexpected code: %#v", issues)
	}
}

func TestValidateProposalExampleImpactDeclarationSkipsLegacyChange(t *testing.T) {
	proposal := strings.Join([]string{
		"## Why",
		"legacy proposal without declaration",
	}, "\n")

	issues := ValidateProposalExampleImpactDeclaration("introduce-something-a69", proposal)
	if len(issues) != 0 {
		t.Fatalf("legacy change must be skipped, got %#v", issues)
	}
}

func TestValidateProposalExampleImpactDeclarationsDeterministicOrder(t *testing.T) {
	changes := map[string]string{
		"introduce-zeta-contract-a72": strings.Join([]string{
			"## Example Impact Assessment",
			"- 非法值",
		}, "\n"),
		"introduce-alpha-contract-a71": strings.Join([]string{
			"## Why",
			"missing section content",
		}, "\n"),
	}

	issues := ValidateProposalExampleImpactDeclarations(changes)
	if len(issues) != 2 {
		t.Fatalf("expected two issues, got %#v", issues)
	}
	if issues[0].Code != GovernanceCodeMissingExampleImpactDeclaration {
		t.Fatalf("first issue should be missing declaration for sorted change name, got %#v", issues)
	}
	if issues[1].Code != GovernanceCodeInvalidExampleImpactValue {
		t.Fatalf("second issue should be invalid value for sorted change name, got %#v", issues)
	}
}
