package contributioncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHostContextProjectionGateScriptsAreEquivalentAndBlocking(t *testing.T) {
	root := repoRoot(t)
	for _, relative := range []string{
		filepath.Join("scripts", "check-host-context-assembly-handoff-contract.sh"),
		filepath.Join("scripts", "check-host-context-assembly-handoff-contract.ps1"),
	} {
		raw, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("read %s: %v", filepath.ToSlash(relative), err)
		}
		text := string(raw)
		for _, token := range []string{
			"host_context_projection",
			"TestReplayHostContextProjection",
			"TestRunAndStreamNormalizeProjectionBudgetHandoffAndRejectionEquivalently",
			"TestEquivalentRunStreamProjectedHandoffsRestoreToSameNormalizedOutcome",
			"TestProjectedHandoffDoesNotPersistResolvedBody",
			"TestHostContextProjectionOwnershipBoundary",
			"TestOfflineExampleEmitsContractMarkers",
			"TestComposerResumeChildContinuationResolvesBoundContextBeforeExecution",
			"TestComposerResumeChildContinuationRechecksSchedulerFenceAfterResolver",
			"TestTeamContinuationResolvesContextOnlyForBoundPendingMember",
			"TestTeamContinuationRejectsForeignMemberProjectionBeforeResolverOrClaim",
			"TestTeamContextResumeRechecksDeadlineAfterResolver",
			"TestTeamContextResumeRunStreamParity",
			"TestTeamContextResumeReopensFileOwnerWithBoundMember",
			"TestTeamPauseRejectsRemoteMemberContextBinding",
			"TestTeamVoteContinuationResolvesBoundPendingMemberWithoutRerunningSettledVote",
			"TestWorkflowContinuationRejectsForeignStepProjectionBeforeResolverOrClaim",
			"TestWorkflowContinuationResolvesBoundContextOnlyForPendingStep",
			"TestWorkflowContextResumeRunStreamParity",
			"TestWorkflowContextResumeRechecksExpiryAfterResolver",
			"host_context.workflow_cross_step_rejected",
			"TestCapabilityManifestDeclaresContextDimensionsIndependently",
			"unavailable",
		} {
			if !strings.Contains(text, token) {
				t.Fatalf("%s missing required host-context gate token %q", filepath.ToSlash(relative), token)
			}
		}
	}
}

func TestQualityGateIncludesHostContextProjectionGate(t *testing.T) {
	root := repoRoot(t)
	for _, relative := range []string{filepath.Join("scripts", "check-quality-gate.sh"), filepath.Join("scripts", "check-quality-gate.ps1")} {
		raw, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("read %s: %v", filepath.ToSlash(relative), err)
		}
		if !strings.Contains(string(raw), "check-host-context-assembly-handoff-contract") {
			t.Fatalf("%s does not invoke host-context contract gate", filepath.ToSlash(relative))
		}
	}
}

func TestQualityGateKeepsHostContextLintReachable(t *testing.T) {
	root := repoRoot(t)
	for _, relative := range []string{filepath.Join("scripts", "check-quality-gate.sh"), filepath.Join("scripts", "check-quality-gate.ps1")} {
		raw, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("read %s: %v", filepath.ToSlash(relative), err)
		}
		if !strings.Contains(string(raw), "golangci-lint run --config .golangci.yml ./...") {
			t.Fatalf("%s must lint explicit module packages instead of the Go-file-free repository root", filepath.ToSlash(relative))
		}
	}
	psRaw, err := os.ReadFile(filepath.Join(root, "scripts", "check-quality-gate.ps1"))
	if err != nil {
		t.Fatalf("read PowerShell quality gate: %v", err)
	}
	if !strings.Contains(string(psRaw), "-DefaultValue 1800") {
		t.Fatal("PowerShell quality gate must retain enough default total time for full lint after the host-context contract suite")
	}
}

func TestHostContextProjectionOwnershipBoundary(t *testing.T) {
	root := repoRoot(t)
	files := []string{
		filepath.Join("context", "hostprojection", "projection.go"),
		filepath.Join("context", "assembler", "assembler.go"),
		filepath.Join("context", "assembler", "handoff.go"),
	}
	for _, relative := range files {
		raw, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("read %s: %v", filepath.ToSlash(relative), err)
		}
		if violation := hostContextOwnershipViolation(string(raw)); violation != "" {
			t.Fatalf("%s violates host-context ownership boundary: %s", filepath.ToSlash(relative), violation)
		}
	}
	for name, source := range map[string]string{
		"provider sdk":       `import "github.com/openai/openai-go"`,
		"budget writeback":   `func bridge() { budget.WriteBack() }`,
		"direct diagnostics": `event.RuntimeRecorder{}.Record()`,
		"parallel owner":     `go resumeProjectedContext()`,
	} {
		if hostContextOwnershipViolation(source) == "" {
			t.Fatalf("deliberate %s violation was not detected", name)
		}
	}
}

func hostContextOwnershipViolation(source string) string {
	lower := strings.ToLower(source)
	for token, violation := range map[string]string{
		"github.com/openai/":      "provider_sdk_dependency",
		"github.com/anthropic/":   "provider_sdk_dependency",
		"google.golang.org/genai": "provider_sdk_dependency",
		"budget.writeback":        "budget_writeback",
		"budgetledger":            "second_budget_ledger",
		"runtimerecorder":         "direct_diagnostics_write",
		"go resumeprojected":      "parallel_recovery_owner",
		"newterminal":             "parallel_terminal_owner",
	} {
		if strings.Contains(lower, token) {
			return violation
		}
	}
	return ""
}
