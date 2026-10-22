Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
$env:GOTOOLCHAIN = "local"
$env:GOCACHE = Join-Path $repoRoot ".gocache-deployment-scope"

Write-Host "[durable-store-deployment-scope] local ownership, fail-closed admission and read-only replay"
Invoke-NativeStrict -Label "deployment scope contract suites" -Command {
    go test ./core/types ./core/checkpoint ./core/history ./core/runner ./orchestration/composer ./orchestration/workflow ./orchestration/mailbox ./orchestration/planbridge ./tool/diagnosticsreplay -run 'TestDeploymentScope|TestCheckpointClaimRejectsUnsupportedDeploymentBeforeMutation|TestHostExecutionPlanRejectsUnsupportedDeploymentBeforeActivation|TestReplayDeploymentScopeFixture' -count=1
}

$files = @("core/types/deployment_scope.go", "tool/diagnosticsreplay/deployment_scope.go")
foreach ($file in $files) {
    $matches = @(rg -n -- '(Run|Stream|Resume|Retry|Append|ClaimResume)\(' $file)
    if ($LASTEXITCODE -eq 0) { throw "[durable-store-deployment-scope][ownership] $file contains source execution or mutation" }
    if ($LASTEXITCODE -ne 1) { throw "[durable-store-deployment-scope] ownership scan failed for $file" }
}
Write-Host "[durable-store-deployment-scope] passed"
