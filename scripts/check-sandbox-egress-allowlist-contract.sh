#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$(pwd)/.gocache"
fi

echo "[sandbox-egress-allowlist-gate] runtime config + readiness + admission contracts"
go test ./runtime/config -run 'Test(SecuritySandboxEgress|AdapterAllowlist|SandboxEgressReadiness|ManagerReadinessPreflightSandboxEgress|ManagerReadinessPreflightAdapterAllowlist|ManagerReadinessAdmissionAdapterAllowlist|ArbitratePrimaryReasonSandboxEgress|ArbitratePrimaryReasonAdapterAllowlist)' -count=1

echo "[sandbox-egress-allowlist-gate] adapter manifest allowlist activation contracts"
go test ./adapter/manifest -run 'Test(ParseManifestAllowlist|ActivateManifestAllowlist)' -count=1

echo "[sandbox-egress-allowlist-gate] sandbox adapter conformance egress + allowlist matrix"
go test ./integration/adapterconformance -run 'TestSandboxAdapterConformance(EgressPolicyMatrix|EgressSelectorOverridePrecedence|AllowlistActivationMatrix|AllowlistTaxonomyDriftClassification|CanonicalDriftClasses)' -count=1

echo "[sandbox-egress-allowlist-gate] diagnostics additive fields and run/stream parity"
go test ./runtime/diagnostics ./observability/event ./core/runner ./integration -run 'Test(StoreRunSandboxEgressAdditiveFieldsPersistAndReplayIdempotent|RuntimeRecorderParsesSandboxEgressAdditiveFields|RunSandboxEgressAdditiveFieldsPropagateToRunFinishedPayload|RuntimeReadinessAdmissionContractAdapterAllowlistMissingEntryRunStreamParity)' -count=1

echo "[sandbox-egress-allowlist-gate] replay fixture + drift classification"
go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractPrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|ReplayContractPrimaryReasonArbitrationFixtureDriftClassification|ReplayContractArbitrationMixedSandboxRolloutMemoryReactSandboxEgressCompatibility|ReplayContractSandboxEgressAllowlistFixture|PrimaryReasonArbitrationReplayContractFixtureSuite|PrimaryReasonArbitrationReplayContractDriftGuardFailFast)' -count=1
