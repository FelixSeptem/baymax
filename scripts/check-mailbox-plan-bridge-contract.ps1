Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[mailbox-plan-bridge-contract] owner delegation, lifecycle, parity and reference-only replay"
Invoke-NativeStrict -Label "mailbox plan bridge suites" -Command {
    go test ./orchestration/planbridge ./orchestration/mailbox ./integration ./tool/diagnosticsreplay -run 'Test(MailboxPlan|HostPlanOutcome|HostExecutionPlanMailboxAsync)' -count=1
}
$pattern = 'github.com/FelixSeptem/baymax/(observability|mcp/)|^[\t ]*go[\t ]+|\.(Run|Stream|Resume|Retry|Cancel|Record|AppendEvent)\('
$matches = @(rg -n -- $pattern tool/diagnosticsreplay/mailbox_plan_bridge.go)
if ($LASTEXITCODE -eq 0) { throw "[mailbox-plan-bridge-contract] replay owns an executable surface: $($matches -join '; ')" }
if ($LASTEXITCODE -ne 1) { throw "[mailbox-plan-bridge-contract] ownership scan failed" }

$env:BAYMAX_MAILBOX_PLAN_BRIDGE_FIXTURE = "testdata/mailbox_plan_bridge.invalid.json"
$invalidExit = Invoke-NativeStrict -Label "invalid mailbox replay fixture" -AllowFailure -Command { go test ./tool/diagnosticsreplay -run TestMailboxPlanBridgeReplayGoldenAndDrift -count=1 }
Remove-Item Env:BAYMAX_MAILBOX_PLAN_BRIDGE_FIXTURE
if ($invalidExit -eq 0) { throw "[mailbox-plan-bridge-contract] invalid replay fixture unexpectedly passed" }

Write-Host "[mailbox-plan-bridge-contract] passed"
