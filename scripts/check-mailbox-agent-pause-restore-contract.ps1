Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[mailbox-agent-pause-restore-contract] owner continuation and replay evidence"
Invoke-NativeStrict -Label "mailbox pause/restore suites" -Command {
    go test ./orchestration/mailbox ./orchestration/planbridge ./tool/diagnosticsreplay -count=1
}

$files = @(
    "orchestration/planbridge/capability.go",
    "orchestration/planbridge/compile.go",
    "orchestration/planbridge/execute.go",
    "orchestration/planbridge/types.go",
    "orchestration/planbridge/snapshot.go"
)
$pattern = 'mailbox\.NewContinuationWorker|mailbox\.New(File|Memory)Store\(|mailbox\.Consume\(|^[\t ]*go[\t ]+'
$matches = @(rg -n -- $pattern $files)
if ($LASTEXITCODE -eq 0) { throw "[mailbox-agent-pause-restore-contract][ownership] planbridge created a mailbox worker/store/consume loop: $($matches -join '; ')" }
if ($LASTEXITCODE -ne 1) { throw "[mailbox-agent-pause-restore-contract] ownership scan failed" }

Write-Host "[mailbox-agent-pause-restore-contract] passed"
