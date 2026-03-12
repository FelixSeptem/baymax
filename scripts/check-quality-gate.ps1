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

$lintConfig = ".golangci.yml"
Write-Host "[quality-gate] golangci-lint --config $lintConfig"
golangci-lint run --config $lintConfig

$scanMode = if ($env:BAYMAX_SECURITY_SCAN_MODE) { $env:BAYMAX_SECURITY_SCAN_MODE.Trim().ToLowerInvariant() } else { "strict" }
$govulncheckEnabled = if ($env:BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED) { $env:BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED.Trim().ToLowerInvariant() } else { "true" }
if ($govulncheckEnabled -eq "true") {
    Write-Host "[quality-gate] govulncheck (mode=$scanMode)"
    go run golang.org/x/vuln/cmd/govulncheck@latest ./...
    if ($LASTEXITCODE -ne 0) {
        if ($scanMode -eq "warn") {
            Write-Warning "[quality-gate] govulncheck found issues but mode=warn; continue"
        }
        else {
            throw "[quality-gate] govulncheck found issues; mode=strict fails"
        }
    }
}
else {
    Write-Host "[quality-gate] govulncheck disabled by BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED"
}

Write-Host "[quality-gate] done"
