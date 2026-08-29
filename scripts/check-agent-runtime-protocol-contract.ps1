Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[agent-runtime-protocol-contract-gate] replay suites"
Invoke-NativeStrict -Label "go test protocol replay suites" -Command {
    go test ./core/types ./core/runner ./orchestration/snapshot ./orchestration/composer ./tool/diagnosticsreplay ./integration ./observability/event ./observability/trace ./examples/agent-modes/internal/runtimeexample -run 'Test(AgentRuntimeProtocol|EvaluateProtocolFixture|ProtocolDescriptor|ProjectProtocol|ConcurrentRunAdmission|RuntimeRecorderParses|ProtocolDecisionAttributes|Checkpoint|Provenance|EventStreamBinding|RealtimeEventStreamBinding|ModeSpec)' -count=1
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

Write-Host "[agent-runtime-protocol-contract-gate] checkpoint_workspace_provenance_surface"
$provenanceMatches = @(rg -n 'CheckpointProvenance|ValidateCheckpointHistory|ValidateWorkspaceIntegrity|agent_runtime_protocol_checkpoint_provenance.v1' core/types orchestration tool examples 2>$null)
if ($LASTEXITCODE -ne 0 -or $provenanceMatches.Count -eq 0) {
    throw "[agent-runtime-protocol-contract-gate][checkpoint_workspace_provenance_surface] required provenance assertions missing"
}

Write-Host "[agent-runtime-protocol-contract-gate] stream_binding_control_plane_absent"
$bindingMatches = @(rg -n --glob '!openspec/changes/archive/**' 'stream[_-]?binding.*(listener|connection[_-]?manager|event[_-]?store|global[_-]?queue|hosted[_-]?service|gateway)' . 2>$null)
if ($LASTEXITCODE -eq 0 -and $bindingMatches.Count -gt 0) {
    throw "[agent-runtime-protocol-contract-gate][stream_binding_control_plane_absent] hosted binding dependency detected"
}
if ($LASTEXITCODE -gt 1) {
    throw "[agent-runtime-protocol-contract-gate] stream binding rg scan failed (exit=$LASTEXITCODE)"
}

Write-Host "[agent-runtime-protocol-contract-gate] passed"
