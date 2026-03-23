Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

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

function Invoke-RequiredStep {
    param(
        [Parameter(Mandatory = $true)][string]$StepLabel,
        [Parameter(Mandatory = $true)][scriptblock]$Command
    )

    Write-Host $StepLabel
    [void](Invoke-NativeStrict -Label $StepLabel -Command $Command)
}

Invoke-RequiredStep -StepLabel "[quality-gate] repo hygiene" -Command {
    pwsh -File scripts/check-repo-hygiene.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] docs consistency" -Command {
    pwsh -File scripts/check-docs-consistency.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] canonical mailbox entrypoints" -Command {
    pwsh -File scripts/check-canonical-mailbox-entrypoints.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] multi-agent shared contract suites" -Command {
    pwsh -File scripts/check-multi-agent-shared-contract.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] mailbox runtime wiring regression" -Command {
    go test ./integration -run '^TestComposerContractMailboxRuntimeWiring' -count=1
}

Invoke-RequiredStep -StepLabel "[quality-gate] adapter conformance" -Command {
    pwsh -File scripts/check-adapter-conformance.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] adapter manifest contract" -Command {
    pwsh -File scripts/check-adapter-manifest-contract.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] adapter capability negotiation contract" -Command {
    pwsh -File scripts/check-adapter-capability-contract.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] adapter contract replay" -Command {
    pwsh -File scripts/check-adapter-contract-replay.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] adapter scaffold drift" -Command {
    pwsh -File scripts/check-adapter-scaffold-drift.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] go test ./..." -Command {
    go test ./...
}

$cgoEnabled = ((Invoke-NativeCaptureStrict -Label "go env CGO_ENABLED" -Command {
            go env CGO_ENABLED
        }) | Select-Object -First 1).Trim()
if ($cgoEnabled -ne "1") {
    throw "[quality-gate] go test -race requires CGO_ENABLED=1"
}

$pkgs = (Invoke-NativeCaptureStrict -Label "go list ./..." -Command {
        go list ./...
    }) | Where-Object { $_ -notmatch "/examples/" }
if (-not $pkgs -or $pkgs.Count -eq 0) {
    throw "[quality-gate] no packages found for race tests"
}

Invoke-RequiredStep -StepLabel "[quality-gate] go test -race (exclude examples packages)" -Command {
    go test -race @pkgs
}

$lintConfig = ".golangci.yml"
Invoke-RequiredStep -StepLabel "[quality-gate] golangci-lint --config $lintConfig" -Command {
    golangci-lint run --config $lintConfig
}

Invoke-RequiredStep -StepLabel "[quality-gate] CA4 benchmark regression" -Command {
    pwsh -File scripts/check-ca4-benchmark-regression.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] multi-agent mainline benchmark regression" -Command {
    pwsh -File scripts/check-multi-agent-performance-regression.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] full-chain example smoke" -Command {
    pwsh -File scripts/check-full-chain-example-smoke.ps1
}

$scanMode = if ($env:BAYMAX_SECURITY_SCAN_MODE) { $env:BAYMAX_SECURITY_SCAN_MODE.Trim().ToLowerInvariant() } else { "strict" }
$govulncheckEnabled = if ($env:BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED) { $env:BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED.Trim().ToLowerInvariant() } else { "true" }
if ($govulncheckEnabled -eq "true") {
    Write-Host "[quality-gate] govulncheck (mode=$scanMode)"

    # The only governance exception path: warn mode allows vulnerability findings without unblocking other strict checks.
    $allowWarn = ($scanMode -eq "warn")
    if (Get-Command govulncheck -ErrorAction SilentlyContinue) {
        $exitCode = Invoke-NativeStrict -Label "govulncheck ./..." -AllowFailure:$allowWarn -Command {
            govulncheck ./...
        }
    }
    else {
        $exitCode = Invoke-NativeStrict -Label "go run golang.org/x/vuln/cmd/govulncheck@latest ./..." -AllowFailure:$allowWarn -Command {
            go run golang.org/x/vuln/cmd/govulncheck@latest ./...
        }
    }

    if ($allowWarn -and $exitCode -ne 0) {
        Write-Warning "[quality-gate] govulncheck found issues but mode=warn; continue"
    }
}
else {
    Write-Host "[quality-gate] govulncheck disabled by BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED"
}

Write-Host "[quality-gate] done"
