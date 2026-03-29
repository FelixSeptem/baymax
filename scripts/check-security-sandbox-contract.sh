#!/usr/bin/env bash
set -euo pipefail

echo "[security-sandbox-gate] sandbox executor and dispatcher contracts"
go test ./core/types ./tool/local ./mcp/stdio -run 'Test(NormalizeSandboxExecSpec|DispatcherSandbox|OfficialClientSandbox)' -count=1

echo "[security-sandbox-gate] sandbox runtime config, readiness, diagnostics contracts"
go test ./runtime/config ./runtime/diagnostics ./observability/event -run 'Test(SecuritySandbox|ValidateRejectsInvalidSandbox|ManagerReadinessPreflightSandbox|ManagerReadinessAdmissionSandbox|ArbitratePrimaryReasonSandbox|StoreRunSandbox|RuntimeRecorderA51)' -count=1

echo "[security-sandbox-gate] sandbox runner, replay, integration parity contracts"
go test ./core/runner ./tool/diagnosticsreplay ./integration -run 'Test(WithSandboxExecutor|RunSandbox|SecurityEventContractSandbox|SecurityDeliveryContractSandbox|ReplayContractPrimaryReasonArbitrationFixture|PrimaryReasonArbitrationReplayContract|SandboxExecutionIsolationContract)' -count=1

echo "[security-sandbox-gate] sandbox executor conformance harness"
bash scripts/check-sandbox-executor-conformance.sh
