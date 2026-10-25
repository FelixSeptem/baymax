Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[host-execution-plan-contract] bounded compile, native delegation, replay and example"
Invoke-NativeStrict -Label "host execution plan suites" -Command {
    go test ./orchestration/planbridge ./orchestration/teams ./orchestration/workflow ./orchestration/scheduler ./integration ./tool/diagnosticsreplay ./tool/contributioncheck -run 'TestHostExecutionPlan|TestHostExecutionPlanOwnershipBoundary' -count=1
}
$pattern = 'github.com/FelixSeptem/baymax/(observability|mcp/)|^[\t ]*go[\t ]+|\.(Resume|Retry|Cancel|Record|AppendEvent)\('
$matches = @(rg -n -- $pattern orchestration/planbridge tool/diagnosticsreplay/host_execution_plan.go)
if ($LASTEXITCODE -eq 0) { throw "[host-execution-plan-contract][ownership] $($matches -join '; ')" }
if ($LASTEXITCODE -ne 1) { throw "[host-execution-plan-contract] ownership scan failed" }

$env:BAYMAX_HOST_EXECUTION_PLAN_FIXTURE = "testdata/host_execution_plan.invalid.json"
$invalidExit = Invoke-NativeStrict -Label "invalid replay fixture" -AllowFailure -Command { go test ./tool/diagnosticsreplay -run TestHostExecutionPlanReplayGoldenAndDrift -count=1 }
Remove-Item Env:BAYMAX_HOST_EXECUTION_PLAN_FIXTURE
if ($invalidExit -eq 0) { throw "[host-execution-plan-contract] invalid replay fixture unexpectedly passed" }

$smoke = @()
Invoke-NativeStrict -Label "host execution plan example" -Command { $script:smoke = @(go run ./examples/host-execution-plan) }
foreach ($marker in @("plan-compiled", "source-owner-preserved", "run-stream-parity", "mailbox-logical-admission", "mailbox-lifecycle-source-owned", "team-mailbox-continuation-owner-configured", "invalid-plan-rejected", "unsupported-capability-rejected", "no-side-effects-before-rejection", "host-commit-required")) {
    if ($smoke -notcontains $marker) { throw "[host-execution-plan-contract] missing marker $marker" }
}
Write-Host "[host-execution-plan-contract] passed"
