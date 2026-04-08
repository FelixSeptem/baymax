Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$pattern = '(\.go\.[0-9]+$|\.tmp$|\.bak$|~$)'
$tracked = Invoke-NativeCaptureStrict -Label "git ls-files" -Command { git ls-files }
$untracked = Invoke-NativeCaptureStrict -Label "git ls-files --others --exclude-standard" -Command { git ls-files --others --exclude-standard }
$matches = @($tracked + $untracked | Where-Object { $_ -match $pattern } | Sort-Object -Unique)

if ($matches.Count -gt 0) {
    Write-Error ("[repo-hygiene] found banned temporary/backup artifacts (tracked or untracked): " + ($matches -join ", "))
}

Write-Host "[repo-hygiene] passed"
