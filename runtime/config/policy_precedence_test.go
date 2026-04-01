package config

import "testing"

func TestEvaluateRuntimePolicyDecisionEmptyCandidates(t *testing.T) {
	got, err := EvaluateRuntimePolicyDecision(DefaultConfig().Runtime.Policy, nil)
	if err != nil {
		t.Fatalf("EvaluateRuntimePolicyDecision failed: %v", err)
	}
	if got.Version != RuntimePolicyPrecedenceVersionPolicyStackV1 {
		t.Fatalf("version=%q, want %q", got.Version, RuntimePolicyPrecedenceVersionPolicyStackV1)
	}
	if got.WinnerStage != "" {
		t.Fatalf("winner_stage=%q, want empty", got.WinnerStage)
	}
	if got.DenySource != "" {
		t.Fatalf("deny_source=%q, want empty", got.DenySource)
	}
	if len(got.PolicyDecisionPath) != 0 {
		t.Fatalf("policy_decision_path len=%d, want 0", len(got.PolicyDecisionPath))
	}
}

func TestEvaluateRuntimePolicyDecisionDenyPrecedence(t *testing.T) {
	got, err := EvaluateRuntimePolicyDecision(DefaultConfig().Runtime.Policy, []RuntimePolicyCandidate{
		{Stage: RuntimePolicyStageReadinessAdmission, Code: ReadinessAdmissionCodeBlocked, Source: "readiness.admission", Decision: RuntimePolicyDecisionDeny},
		{Stage: RuntimePolicyStageActionGate, Code: "action.gate.allow", Source: "action_gate.rule", Decision: RuntimePolicyDecisionAllow},
		{Stage: RuntimePolicyStageSecurityS2, Code: "security.permission_denied", Source: "security_s2.guard", Decision: RuntimePolicyDecisionDeny},
	})
	if err != nil {
		t.Fatalf("EvaluateRuntimePolicyDecision failed: %v", err)
	}
	if got.WinnerStage != RuntimePolicyStageSecurityS2 {
		t.Fatalf("winner_stage=%q, want %q", got.WinnerStage, RuntimePolicyStageSecurityS2)
	}
	if got.DenySource != "security_s2.guard" {
		t.Fatalf("deny_source=%q, want security_s2.guard", got.DenySource)
	}
	if len(got.PolicyDecisionPath) != 3 {
		t.Fatalf("policy_decision_path len=%d, want 3", len(got.PolicyDecisionPath))
	}
	if got.PolicyDecisionPath[0].Stage != RuntimePolicyStageActionGate ||
		got.PolicyDecisionPath[1].Stage != RuntimePolicyStageSecurityS2 ||
		got.PolicyDecisionPath[2].Stage != RuntimePolicyStageReadinessAdmission {
		t.Fatalf("unexpected stage order: %#v", got.PolicyDecisionPath)
	}
}

func TestEvaluateRuntimePolicyDecisionDenyOverridesAllow(t *testing.T) {
	got, err := EvaluateRuntimePolicyDecision(DefaultConfig().Runtime.Policy, []RuntimePolicyCandidate{
		{Stage: RuntimePolicyStageActionGate, Code: "action.gate.allow", Source: "action_gate.rule", Decision: RuntimePolicyDecisionAllow},
		{Stage: RuntimePolicyStageReadinessAdmission, Code: ReadinessAdmissionCodeBlocked, Source: "readiness.admission", Decision: RuntimePolicyDecisionDeny},
	})
	if err != nil {
		t.Fatalf("EvaluateRuntimePolicyDecision failed: %v", err)
	}
	if got.WinnerStage != RuntimePolicyStageReadinessAdmission {
		t.Fatalf("winner_stage=%q, want %q", got.WinnerStage, RuntimePolicyStageReadinessAdmission)
	}
	if got.DenySource != "readiness.admission" {
		t.Fatalf("deny_source=%q, want readiness.admission", got.DenySource)
	}
}

func TestEvaluateRuntimePolicyDecisionSameStageTieBreakDeterministic(t *testing.T) {
	t.Run("code-lexical-order", func(t *testing.T) {
		got, err := EvaluateRuntimePolicyDecision(DefaultConfig().Runtime.Policy, []RuntimePolicyCandidate{
			{Stage: RuntimePolicyStageSandboxEgress, Code: "sandbox.egress.zzz", Source: "sandbox_egress", Decision: RuntimePolicyDecisionDeny},
			{Stage: RuntimePolicyStageSandboxEgress, Code: "sandbox.egress.aaa", Source: "sandbox_egress", Decision: RuntimePolicyDecisionDeny},
		})
		if err != nil {
			t.Fatalf("EvaluateRuntimePolicyDecision failed: %v", err)
		}
		if got.WinnerStage != RuntimePolicyStageSandboxEgress {
			t.Fatalf("winner_stage=%q, want %q", got.WinnerStage, RuntimePolicyStageSandboxEgress)
		}
		if got.DenySource != "sandbox_egress" {
			t.Fatalf("deny_source=%q, want sandbox_egress", got.DenySource)
		}
		if got.TieBreakReason != RuntimePolicyTieBreakerModeLexicalCodeThenSourceOrder {
			t.Fatalf("tie_break_reason=%q, want %q", got.TieBreakReason, RuntimePolicyTieBreakerModeLexicalCodeThenSourceOrder)
		}
		if len(got.PolicyDecisionPath) != 2 || got.PolicyDecisionPath[0].Code != "sandbox.egress.aaa" {
			t.Fatalf("policy_decision_path=%#v, want lexical winner first", got.PolicyDecisionPath)
		}
	})

	t.Run("source-order", func(t *testing.T) {
		got, err := EvaluateRuntimePolicyDecision(DefaultConfig().Runtime.Policy, []RuntimePolicyCandidate{
			{Stage: RuntimePolicyStageSandboxEgress, Code: "sandbox.egress.same", Source: RuntimePolicyStageSandboxEgress, Decision: RuntimePolicyDecisionDeny},
			{Stage: RuntimePolicyStageSandboxEgress, Code: "sandbox.egress.same", Source: RuntimePolicyStageSecurityS2, Decision: RuntimePolicyDecisionDeny},
		})
		if err != nil {
			t.Fatalf("EvaluateRuntimePolicyDecision failed: %v", err)
		}
		if got.DenySource != RuntimePolicyStageSecurityS2 {
			t.Fatalf("deny_source=%q, want %q by source_order", got.DenySource, RuntimePolicyStageSecurityS2)
		}
		if got.TieBreakReason != RuntimePolicyTieBreakerModeLexicalCodeThenSourceOrder {
			t.Fatalf("tie_break_reason=%q, want %q", got.TieBreakReason, RuntimePolicyTieBreakerModeLexicalCodeThenSourceOrder)
		}
	})
}

func TestEvaluateRuntimePolicyDecisionVersionSwitch(t *testing.T) {
	cfg := DefaultConfig().Runtime.Policy
	cfg.Precedence.Version = RuntimePolicyPrecedenceVersionPolicyStackV1
	if _, err := EvaluateRuntimePolicyDecision(cfg, []RuntimePolicyCandidate{
		{Stage: RuntimePolicyStageActionGate, Code: "action.gate.allow", Source: "action_gate", Decision: RuntimePolicyDecisionAllow},
	}); err != nil {
		t.Fatalf("expected v1 evaluation success, got %v", err)
	}

	cfg.Precedence.Version = "policy_stack.v2"
	if _, err := EvaluateRuntimePolicyDecision(cfg, []RuntimePolicyCandidate{
		{Stage: RuntimePolicyStageActionGate, Code: "action.gate.allow", Source: "action_gate", Decision: RuntimePolicyDecisionAllow},
	}); err == nil {
		t.Fatal("expected unsupported version error")
	}
}

func TestEvaluateRuntimePolicyDecisionRejectsUnsupportedCandidateStage(t *testing.T) {
	_, err := EvaluateRuntimePolicyDecision(DefaultConfig().Runtime.Policy, []RuntimePolicyCandidate{
		{Stage: "unknown_stage", Code: "unknown.deny", Source: "unknown", Decision: RuntimePolicyDecisionDeny},
	})
	if err == nil {
		t.Fatal("expected unsupported stage error")
	}
}

func TestEvaluateRuntimePolicyDecisionExplainabilityDisabled(t *testing.T) {
	cfg := DefaultConfig().Runtime.Policy
	cfg.Explainability.Enabled = false
	got, err := EvaluateRuntimePolicyDecision(cfg, []RuntimePolicyCandidate{
		{Stage: RuntimePolicyStageSandboxEgress, Code: "sandbox.egress.b", Source: RuntimePolicyStageSandboxEgress, Decision: RuntimePolicyDecisionDeny},
		{Stage: RuntimePolicyStageSandboxEgress, Code: "sandbox.egress.a", Source: RuntimePolicyStageSecurityS2, Decision: RuntimePolicyDecisionDeny},
	})
	if err != nil {
		t.Fatalf("EvaluateRuntimePolicyDecision failed: %v", err)
	}
	if got.WinnerStage != RuntimePolicyStageSandboxEgress {
		t.Fatalf("winner_stage=%q, want %q", got.WinnerStage, RuntimePolicyStageSandboxEgress)
	}
	if got.DenySource == "" {
		t.Fatal("deny_source should remain populated even when explainability is disabled")
	}
	if len(got.PolicyDecisionPath) != 0 {
		t.Fatalf("policy_decision_path len=%d, want 0 when explainability disabled", len(got.PolicyDecisionPath))
	}
	if got.TieBreakReason != "" {
		t.Fatalf("tie_break_reason=%q, want empty when explainability disabled", got.TieBreakReason)
	}
}
