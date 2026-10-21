Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[native-event-history-contract] reference-only replay, cursor, retention and ownership"
Invoke-NativeStrict -Label "native event history replay suites" -Command {
    go test ./core/history ./core/types ./core/runner ./orchestration/composer ./orchestration/teams ./orchestration/workflow ./orchestration/mailbox ./orchestration/planbridge ./host/eventadapter ./tool/diagnosticsreplay ./integration -run '^TestNativeHistory|^TestJournal|^TestRunnerNativeHistory|^TestTeamTimelineCanAppend|^TestWorkflowTimelineCanAppend|^TestMailboxHistory|^TestFileMailboxHistory|^TestCallbackAdapterNeverOwnsHistory|^TestRetainedFactCanReachSink' -count=1
}

$file = "tool/diagnosticsreplay/native_event_history.go"
$pattern = '"(net/|database/|os"|github.com/FelixSeptem/baymax/(core/runner|runtime/|observability/|orchestration/|model/|mcp/))|^[\t ]*go[\t ]+|\.(Run|Stream|Resume|Cancel|Retry|Record|AppendEvent|ReadAfter)\('
$matches = @(rg -n -- $pattern $file)
if ($LASTEXITCODE -eq 0) { throw "[native-event-history-contract][ownership] $($matches -join '; ')" }
if ($LASTEXITCODE -ne 1) { throw "[native-event-history-contract] ownership scan failed" }
Write-Host "[native-event-history-contract] passed"
