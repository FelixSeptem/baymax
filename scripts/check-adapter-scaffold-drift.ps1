Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}

Write-Host "[adapter-scaffold-drift] running fixture drift validation"
try {
    go test ./adapter/scaffold -run '^TestScaffoldDriftFixtures$' -count=1
}
catch {
    throw "[adapter-scaffold-drift][fixture-mismatch] generated scaffold output diverged from committed fixtures: $($_.Exception.Message)"
}

if ($LASTEXITCODE -ne 0) {
    throw "[adapter-scaffold-drift][fixture-mismatch] generated scaffold output diverged from committed fixtures"
}

Write-Host "[adapter-scaffold-drift] passed"
