Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[quality-gate] go test ./..."
go test ./...

$cgoEnabled = (go env CGO_ENABLED).Trim()
if ($cgoEnabled -ne "1") {
    throw "[quality-gate] go test -race requires CGO_ENABLED=1"
}

$pkgs = go list ./... | Where-Object { $_ -notmatch "/examples/" }
if (-not $pkgs -or $pkgs.Count -eq 0) {
    throw "[quality-gate] no packages found for race tests"
}

Write-Host "[quality-gate] go test -race (exclude examples packages)"
go test -race @pkgs

Write-Host "[quality-gate] done"
