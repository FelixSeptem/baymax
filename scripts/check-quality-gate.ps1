Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}
if (-not $env:GOLANGCI_LINT_CACHE) {
    $env:GOLANGCI_LINT_CACHE = Join-Path $repoRoot ".gocache/golangci-lint"
}
if (-not $env:CGO_ENABLED) {
    $env:CGO_ENABLED = "1"
}

Write-Host "[quality-gate] repo hygiene"
pwsh -File scripts/check-repo-hygiene.ps1

Write-Host "[quality-gate] docs consistency"
try {
    pwsh -File scripts/check-docs-consistency.ps1
}
catch {
    throw "[quality-gate][docs-consistency] docs consistency failed (adapter mapping or pre1 governance drift): $($_.Exception.Message)"
}

Write-Host "[quality-gate] adapter conformance"
try {
    pwsh -File scripts/check-adapter-conformance.ps1
}
catch {
    throw "[quality-gate][adapter-conformance] adapter conformance harness failed: $($_.Exception.Message)"
}

Write-Host "[quality-gate] adapter manifest contract"
try {
    pwsh -File scripts/check-adapter-manifest-contract.ps1
}
catch {
    throw "[quality-gate][adapter-manifest-contract] adapter manifest contract check failed: $($_.Exception.Message)"
}

Write-Host "[quality-gate] adapter capability negotiation contract"
try {
    pwsh -File scripts/check-adapter-capability-contract.ps1
}
catch {
    throw "[quality-gate][adapter-capability-contract] adapter capability negotiation contract check failed: $($_.Exception.Message)"
}

Write-Host "[quality-gate] adapter contract replay"
try {
    pwsh -File scripts/check-adapter-contract-replay.ps1
}
catch {
    throw "[quality-gate][adapter-contract-replay] adapter contract replay check failed: $($_.Exception.Message)"
}

Write-Host "[quality-gate] adapter scaffold drift"
try {
    pwsh -File scripts/check-adapter-scaffold-drift.ps1
}
catch {
    throw "[quality-gate][adapter-scaffold-drift] adapter scaffold drift check failed: $($_.Exception.Message)"
}

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

Write-Host "[quality-gate] CA4 benchmark regression"
pwsh -File scripts/check-ca4-benchmark-regression.ps1

Write-Host "[quality-gate] multi-agent mainline benchmark regression"
pwsh -File scripts/check-multi-agent-performance-regression.ps1

Write-Host "[quality-gate] full-chain example smoke"
pwsh -File scripts/check-full-chain-example-smoke.ps1

$scanMode = if ($env:BAYMAX_SECURITY_SCAN_MODE) { $env:BAYMAX_SECURITY_SCAN_MODE.Trim().ToLowerInvariant() } else { "strict" }
$govulncheckEnabled = if ($env:BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED) { $env:BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED.Trim().ToLowerInvariant() } else { "true" }
if ($govulncheckEnabled -eq "true") {
    Write-Host "[quality-gate] govulncheck (mode=$scanMode)"
    if (Get-Command govulncheck -ErrorAction SilentlyContinue) {
        govulncheck ./...
    }
    else {
        go run golang.org/x/vuln/cmd/govulncheck@latest ./...
    }
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
