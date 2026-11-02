Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
Write-Host "[execution-graph-contract] owner digest stays separate from topology digest"

Invoke-NativeStrict -Label "execution graph projection suites" -Command {
    go test ./orchestration/graph ./orchestration/workflow ./orchestration/teams ./orchestration/composer ./examples/execution-graph -run '^Test(Compile|Canonical|EquivalentTopology|TopologyDrift|ProjectExecutionGraph|ProjectChildExecutionGraph|OfflineExample)' -count=1
}

$ownerImports = @(rg -n --glob '*.go' --glob '!**/*_test.go' '"github.com/FelixSeptem/baymax/orchestration/(workflow|teams|composer|scheduler|checkpoint|runner)' orchestration/graph)
if ($LASTEXITCODE -eq 0) {
    throw "[execution-graph-contract][ownership] shared graph package imports an owner or executor: $($ownerImports -join '; ')"
}
if ($LASTEXITCODE -ne 1) {
    throw "[execution-graph-contract][ownership] import scan failed"
}

$projectionCalls = @(rg -n --glob 'graph_projection.go' '\.(Run|Stream|Resume|Claim|Save|Commit)\(' orchestration/workflow orchestration/teams orchestration/composer)
if ($LASTEXITCODE -eq 0) {
    throw "[execution-graph-contract][ownership] projection invokes execution or restore owner: $($projectionCalls -join '; ')"
}
if ($LASTEXITCODE -ne 1) {
    throw "[execution-graph-contract][ownership] projection scan failed"
}

Write-Host "[execution-graph-contract] owner digest and execution boundaries passed"
