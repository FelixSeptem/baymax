Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[security-sandbox-gate] sandbox executor and dispatcher contracts"
Invoke-NativeStrict -Label "go test ./core/types ./tool/local ./mcp/stdio -run 'Test(NormalizeSandboxExecSpec|DispatcherSandbox|OfficialClientSandbox)' -count=1" -Command {
    go test ./core/types ./tool/local ./mcp/stdio -run 'Test(NormalizeSandboxExecSpec|DispatcherSandbox|OfficialClientSandbox)' -count=1
}

Write-Host "[security-sandbox-gate] sandbox runtime config, readiness, diagnostics contracts"
Invoke-NativeStrict -Label "go test ./runtime/config ./runtime/diagnostics ./observability/event -run 'Test(SecuritySandbox|ValidateRejectsInvalidSandbox|ManagerReadinessPreflightSandbox|ManagerReadinessAdmissionSandbox|ArbitratePrimaryReasonSandbox|StoreRunSandbox|RuntimeRecorderA51)' -count=1" -Command {
    go test ./runtime/config ./runtime/diagnostics ./observability/event -run 'Test(SecuritySandbox|ValidateRejectsInvalidSandbox|ManagerReadinessPreflightSandbox|ManagerReadinessAdmissionSandbox|ArbitratePrimaryReasonSandbox|StoreRunSandbox|RuntimeRecorderA51)' -count=1
}

Write-Host "[security-sandbox-gate] sandbox runner, replay, integration parity contracts"
Invoke-NativeStrict -Label "go test ./core/runner ./tool/diagnosticsreplay ./integration -run 'Test(WithSandboxExecutor|RunSandbox|SecurityEventContractSandbox|SecurityDeliveryContractSandbox|ReplayContractPrimaryReasonArbitrationFixture|PrimaryReasonArbitrationReplayContract|SandboxExecutionIsolationContract)' -count=1" -Command {
    go test ./core/runner ./tool/diagnosticsreplay ./integration -run 'Test(WithSandboxExecutor|RunSandbox|SecurityEventContractSandbox|SecurityDeliveryContractSandbox|ReplayContractPrimaryReasonArbitrationFixture|PrimaryReasonArbitrationReplayContract|SandboxExecutionIsolationContract)' -count=1
}

Write-Host "[security-sandbox-gate] sandbox executor conformance harness"
Invoke-NativeStrict -Label "pwsh -File scripts/check-sandbox-executor-conformance.ps1" -Command {
    pwsh -File scripts/check-sandbox-executor-conformance.ps1
}
