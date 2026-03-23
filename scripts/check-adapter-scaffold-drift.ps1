Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}

Write-Host "[adapter-scaffold-drift] running fixture drift validation"
try {
    Invoke-NativeStrict -Label "go test ./adapter/scaffold -run '^TestScaffoldDriftFixtures$' -count=1" -Command {
        go test ./adapter/scaffold -run '^TestScaffoldDriftFixtures$' -count=1
    }
}
catch {
    throw "[adapter-scaffold-drift][fixture-mismatch] generated scaffold output diverged from committed fixtures: $($_.Exception.Message)"
}

Write-Host "[adapter-scaffold-drift] passed"
