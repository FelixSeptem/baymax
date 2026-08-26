Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[agent-runtime-protocol-contract-gate] replay suites"
Invoke-NativeStrict -Label "go test protocol replay suites" -Command {
    go test ./core/types ./tool/diagnosticsreplay ./integration ./observability/event ./observability/trace -run 'Test(AgentRuntimeProtocol|EvaluateProtocolFixture|ProtocolDescriptor|ProjectProtocol|ConcurrentRunAdmission|RuntimeRecorderParses|ProtocolDecisionAttributes)' -count=1
}

Write-Host "[agent-runtime-protocol-contract-gate] capability context action admission Run/Stream parity"
$projectionMatches = @(rg -n 'ProtocolDescriptor|ProtocolSessionContext|ProtocolRunAdmission|TestRunAndStreamProtocolProjectionSemanticEquivalence' core/types orchestration a2a integration 2>$null)
if ($LASTEXITCODE -ne 0 -or $projectionMatches.Count -eq 0) {
    throw "[agent-runtime-protocol-contract-gate][projection_surface] required capability/context/action/admission/Run/Stream assertions missing"
}

Write-Host "[agent-runtime-protocol-contract-gate] control_plane_absent"
$matches = @(rg -n --glob '!openspec/changes/archive/**' --glob '!openspec/changes/introduce-agent-runtime-protocol-contract/**' --glob '!openspec/specs/agent-runtime-protocol-contract/**' --glob '!scripts/check-agent-runtime-protocol-contract.*' 'agent_runtime_protocol.*(hosted|control[_-]?plane|gateway|session[_-]?service|artifact[_-]?store)' . 2>$null)
if ($LASTEXITCODE -eq 0 -and $matches.Count -gt 0) {
    throw "[agent-runtime-protocol-contract-gate][control_plane_absent] hosted protocol dependency detected"
}
if ($LASTEXITCODE -gt 1) {
    throw "[agent-runtime-protocol-contract-gate] rg scan failed (exit=$LASTEXITCODE)"
}

Write-Host "[agent-runtime-protocol-contract-gate] passed"
