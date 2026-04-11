package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestAgentModeP0RegressionPaths(t *testing.T) {
	cases := []struct {
		name string
		mode string
		must []string
	}{
		{
			name: "positive-rag-minimal",
			mode: "rag-hybrid-retrieval/minimal",
			must: []string{
				"result.final_answer=rag-hybrid-retrieval/minimal",
				"phase=P0",
				"classification=rag.hybrid_retrieval",
				"fallback=direct_answer",
				"verification.semantic.marker.retrieval_candidates_built=ok",
			},
		},
		{
			name: "degraded-hitl-production",
			mode: "hitl-governed-checkpoint/production-ish",
			must: []string{
				"result.final_answer=hitl-governed-checkpoint/production-ish",
				"phase=P0",
				"decision=timeout",
				"governance=allow_with_recovery",
				"verification.semantic.marker.governance_hitl_gate_enforced=ok",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			out := runAgentModeExample(t, tc.mode)
			for _, token := range tc.must {
				if !strings.Contains(out, token) {
					t.Fatalf("mode %s missing token %q\noutput:\n%s", tc.mode, token, out)
				}
			}
		})
	}
}

func TestAgentModeP1RegressionPaths(t *testing.T) {
	cases := []struct {
		name string
		mode string
		must []string
	}{
		{
			name: "positive-policy-minimal",
			mode: "policy-budget-admission/minimal",
			must: []string{
				"result.final_answer=policy-budget-admission/minimal",
				"phase=P1",
				"admission=admit",
				"reason=within_budget",
				"verification.semantic.marker.decision_trace_recorded=ok",
			},
		},
		{
			name: "failure-workflow-production",
			mode: "workflow-branch-retry-failfast/production-ish",
			must: []string{
				"result.final_answer=workflow-branch-retry-failfast/production-ish",
				"phase=P1",
				"failfast=true",
				"governance=deny",
				"verification.semantic.marker.governance_workflow_gate_enforced=ok",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			out := runAgentModeExample(t, tc.mode)
			for _, token := range tc.must {
				if !strings.Contains(out, token) {
					t.Fatalf("mode %s missing token %q\noutput:\n%s", tc.mode, token, out)
				}
			}
		})
	}
}

func TestAgentModeP2RegressionPaths(t *testing.T) {
	cases := []struct {
		name string
		mode string
		must []string
	}{
		{
			name: "positive-adapter-minimal",
			mode: "adapter-onboarding-manifest-capability/minimal",
			must: []string{
				"result.final_answer=adapter-onboarding-manifest-capability/minimal",
				"phase=P2",
				"fallback=full-capability",
				"governance=n/a",
				"verification.semantic.marker.adapter_capability_negotiated=ok",
			},
		},
		{
			name: "degraded-config-production",
			mode: "config-hot-reload-rollback/production-ish",
			must: []string{
				"result.final_answer=config-hot-reload-rollback/production-ish",
				"phase=P2",
				"validation=rolled_back",
				"fail_fast=true",
				"governance=quarantine_release",
				"verification.semantic.marker.config_invalid_failfast=ok",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			out := runAgentModeExample(t, tc.mode)
			for _, token := range tc.must {
				if !strings.Contains(out, token) {
					t.Fatalf("mode %s missing token %q\noutput:\n%s", tc.mode, token, out)
				}
			}
		})
	}
}

func runAgentModeExample(t *testing.T, mode string) string {
	t.Helper()
	root := repoRootForIntegration(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", "./examples/agent-modes/"+mode)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), ensureAgentModeGoCacheEnv(root)...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("go run timeout for mode %s\noutput:\n%s", mode, string(output))
	}
	if err != nil {
		t.Fatalf("go run failed for mode %s: %v\noutput:\n%s", mode, err, string(output))
	}
	return string(output)
}

func ensureAgentModeGoCacheEnv(root string) []string {
	if strings.TrimSpace(os.Getenv("GOCACHE")) != "" {
		return nil
	}
	cachePath := filepath.Join(root, ".tmp", "go-cache-agent-mode-local")
	_ = os.MkdirAll(cachePath, 0o755)
	return []string{"GOCACHE=" + cachePath}
}

func repoRootForIntegration(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve runtime caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), ".."))
}
