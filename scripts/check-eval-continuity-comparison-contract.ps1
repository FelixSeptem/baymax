$ErrorActionPreference = "Stop"
$env:GOCACHE = if ($env:BAYMAX_CONTINUITY_GOCACHE) { $env:BAYMAX_CONTINUITY_GOCACHE } else { Join-Path (Get-Location) ".gocache-eval-continuity" }
go test ./runtime/evalcontract ./tool/diagnosticsreplay -run 'Test(Continuity|ReplayContractContinuity)' -count=1
if ($LASTEXITCODE -ne 0) { throw "eval continuity contract tests failed" }
if (rg -n -g 'continuity*.go' 'os/exec|os/Chdir|git |workspace mutation|transcript body|reasoning body' runtime/evalcontract tool/diagnosticsreplay) { throw "eval continuity comparator contains forbidden side effects or body persistence" }
Write-Host "[eval-continuity-comparison-contract] passed"
