Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$eventPath = if ($args.Count -gt 0 -and $args[0]) { $args[0] } elseif ($env:GITHUB_EVENT_PATH) { $env:GITHUB_EVENT_PATH } else { "" }
if (-not $eventPath) {
    throw "[contribution-template] usage: pwsh -File scripts/check-contribution-template.ps1 <event.json> (or set GITHUB_EVENT_PATH)"
}

Write-Host "[contribution-template] validating pull request template completeness"
Invoke-NativeStrict -Label "go run ./cmd/contribution-template-check -event $eventPath" -Command {
    go run ./cmd/contribution-template-check -event $eventPath
}
