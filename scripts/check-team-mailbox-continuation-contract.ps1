Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
$env:GOCACHE = Join-Path $repoRoot ".gocache-team-mailbox-continuation-gate"

Write-Host "[team-mailbox-continuation-contract] owner restore, lifecycle races, and owner boundaries"
Invoke-NativeStrict -Label "team/mailbox continuation suites" -Command {
    go test ./orchestration/teams ./orchestration/mailbox ./orchestration/planbridge -count=1
}

# Continuation implementation must remain inside existing team/mailbox owners.
# It may not introduce a second mailbox worker, queue, scheduler, or direct-run fallback.
$files = @(
    "orchestration/mailbox/continuation.go",
    "orchestration/teams/continuation.go"
)
$pattern = '^[\t ]*go[\t ]+|\b(NewMemoryStore|NewFileStore|Publish|Consume)\('
$matches = @(rg -n -- $pattern $files)
if ($LASTEXITCODE -eq 0) { throw "[team-mailbox-continuation-contract][ownership] $($matches -join '; ')" }
if ($LASTEXITCODE -ne 1) { throw "[team-mailbox-continuation-contract] ownership scan failed" }

Write-Host "[team-mailbox-continuation-contract] passed"
