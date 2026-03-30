#!/usr/bin/env bash
set -euo pipefail

echo "[sandbox-rollout-governance-gate] runtime rollout config/readiness contracts"
go test ./runtime/config -run 'Test(SandboxRolloutPhaseTransitionValidation|ManagerSandboxRolloutGovernanceRecordRunAutoFreeze|ManagerSandboxRolloutUnfreezeRequiresCooldownAndToken|ManagerSandboxCapacityActionDeterministicFromQueueAndInflight|ManagerReadinessPreflightSandboxRolloutFrozenFinding|ManagerReadinessPreflightSandboxRolloutHealthBudgetBreachedFinding|ManagerReadinessPreflightSandboxCapacityStrictMapping|ManagerReadinessAdmissionSandboxRolloutFrozenDeny|ManagerReadinessAdmissionSandboxCapacityPolicyMapping)' -count=1

echo "[sandbox-rollout-governance-gate] composer run/stream rollout parity contracts"
go test ./orchestration/composer -run 'TestComposerReadinessAdmissionSandbox(RolloutFrozenRunAndStreamEquivalent|CapacityThrottlePolicyParity|RolloutTimelineReasonParity)' -count=1

echo "[sandbox-rollout-governance-gate] runtime recorder a52 additive contracts"
go test ./observability/event -run 'TestRuntimeRecorder(A52ParserCompatibilityAdditiveNullableDefault|ParsesA52RolloutGovernanceFields)' -count=1

echo "[sandbox-rollout-governance-gate] diagnostics replay a52 fixture drift contracts"
go test ./tool/diagnosticsreplay -run 'TestReplayContract(PrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|PrimaryReasonArbitrationFixtureDriftClassification|ArbitrationMixedA51A52Compatibility)' -count=1
