$ErrorActionPreference = "Stop"
Write-Host "[terminal-outcome-contract-gate] focused contract suites"
go test ./core/types ./tool/local ./orchestration/composer ./tool/diagnosticsreplay ./integration -run 'TerminalOutcome|RecoveryErrorProjects|ReplayFixture' -count=1

Write-Host "[terminal-outcome-contract-gate] source ownership assertions"
$projection = Get-ChildItem core/runner,core/types,tool/local -Recurse -Filter *.go | Select-String -Pattern 'retry exhausted|resume may be possible' | Where-Object { $_.Path -match 'terminal_outcome_projection|terminal_outcome\.go' }
if ($projection) { throw "projection must not synthesize retry/resume from free-form messages" }
$rewind = Get-ChildItem core/runner,core/types,orchestration -Recurse -Filter *.go | Select-String -Pattern 'terminal.*working|working.*terminal' | Where-Object { $_.Path -match 'TerminalOutcome|terminal_outcome' }
if ($rewind) { throw "terminal outcome must not transition back to working" }

Write-Host "[terminal-outcome-contract-gate] done"
