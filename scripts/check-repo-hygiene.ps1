Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$pattern = '(\.go\.[0-9]+$|\.tmp$|\.bak$|~$)'
$tracked = git ls-files
$matches = @($tracked | Where-Object { $_ -match $pattern })

if ($matches.Count -gt 0) {
    Write-Error ("[repo-hygiene] found banned temporary/backup artifacts: " + ($matches -join ", "))
}

Write-Host "[repo-hygiene] passed"
