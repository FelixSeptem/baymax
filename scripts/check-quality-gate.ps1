Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

function Test-WritableDirectory {
    param(
        [Parameter(Mandatory = $false)][string]$Path
    )
    if ([string]::IsNullOrWhiteSpace($Path)) {
        return $false
    }
    try {
        if (-not (Test-Path -LiteralPath $Path)) {
            New-Item -ItemType Directory -Path $Path -Force | Out-Null
        }
        $probe = Join-Path $Path ("._write_probe_" + [Guid]::NewGuid().ToString("N"))
        [System.IO.File]::WriteAllText($probe, "ok")
        Remove-Item -LiteralPath $probe -Force -ErrorAction SilentlyContinue
        return $true
    }
    catch {
        return $false
    }
}

function Ensure-WritableCacheEnv {
    param(
        [Parameter(Mandatory = $true)][string]$EnvName,
        [Parameter(Mandatory = $true)][string]$FallbackPath
    )
    $current = [Environment]::GetEnvironmentVariable($EnvName)
    if (Test-WritableDirectory -Path $current) {
        return
    }
    if (-not (Test-WritableDirectory -Path $FallbackPath)) {
        throw "[quality-gate] unable to prepare writable cache directory for $EnvName at $FallbackPath"
    }
    Set-Item -Path ("Env:" + $EnvName) -Value $FallbackPath
}

Ensure-WritableCacheEnv -EnvName "GOCACHE" -FallbackPath (Join-Path $repoRoot ".gocache")
Ensure-WritableCacheEnv -EnvName "GOLANGCI_LINT_CACHE" -FallbackPath (Join-Path $repoRoot ".gocache/golangci-lint")

if (-not $env:GOPROXY) {
    $env:GOPROXY = "https://proxy.golang.org,direct"
}
if (-not $env:GOSUMDB) {
    $env:GOSUMDB = "sum.golang.org"
}
if (-not $env:GOVULNDB) {
    $env:GOVULNDB = "https://vuln.go.dev"
}
if (-not $env:CGO_ENABLED) {
    $env:CGO_ENABLED = "1"
}
if ($env:GODEBUG) {
    if ($env:GODEBUG -notmatch "(^|,)goindex=") {
        $env:GODEBUG = "$($env:GODEBUG),goindex=0"
    }
}
else {
    $env:GODEBUG = "goindex=0"
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

Invoke-RequiredStep -StepLabel "[quality-gate] runtime readiness + explainability + version governance contract suites" -Command {
    go test ./runtime/config ./runtime/diagnostics ./observability/event ./orchestration/composer ./integration -run 'Test(RuntimeReadiness|ReadinessAdmission|ArbitrationVersionGovernanceContract|StoreRunReadiness|StoreRunArbitrationVersionGovernance|RuntimeRecorderA40ParserCompatibilityAdditiveNullableDefault|RuntimeRecorderA49ParserCompatibilityAdditiveNullableDefault|RuntimeRecorderA50ParserCompatibilityAdditiveNullableDefault|RuntimeRecorderParsesA50ArbitrationVersionGovernanceFields|ComposerReadiness)' -count=1
}

Invoke-RequiredStep -StepLabel "[quality-gate] diagnostics cardinality contract suites" -Command {
    go test ./runtime/config ./runtime/diagnostics ./observability/event ./integration -run 'Test(DiagnosticsCardinality|ManagerDiagnosticsCardinality|StoreRunCardinality|CardinalityListGovernance|RuntimeRecorderA45ParserCompatibilityAdditiveNullableDefault|DiagnosticsCardinalityContract)' -count=1
}

Invoke-RequiredStep -StepLabel "[quality-gate] adapter-health contract suites" -Command {
    go test ./adapter/health ./runtime/config ./runtime/diagnostics ./observability/event ./integration/adapterconformance -run 'Test(RunnerProbe|AdapterHealthConfig|ManagerAdapterHealth|ManagerReadinessPreflightAdapterHealth|StoreRunReadinessAdditiveFieldsPersistAndReplayIdempotent|RuntimeRecorderA14ParserCompatibilityAdditiveNullableDefault|RuntimeRecorderA46ParserCompatibilityAdditiveNullableDefault|AdapterConformanceHealth(Matrix|Governance))' -count=1
}

Invoke-RequiredStep -StepLabel "[quality-gate] mailbox runtime wiring regression" -Command {
    go test ./integration -run '^TestComposerContractMailboxRuntimeWiring' -count=1
}

Invoke-RequiredStep -StepLabel "[quality-gate] timeout resolution contract suites" -Command {
    go test ./integration -run '^TestTimeoutResolutionContract' -count=1
}

Invoke-RequiredStep -StepLabel "[quality-gate] readiness-timeout-health replay fixture suites" -Command {
    go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractCompositeFixture|ReplayContractPrimaryReasonArbitrationFixture|ReadinessTimeoutHealthReplayContract|PrimaryReasonArbitrationReplayContract)' -count=1
}

Invoke-RequiredStep -StepLabel "[quality-gate] react contract suites" -Command {
    pwsh -File scripts/check-react-contract.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] security sandbox contract suites" -Command {
    pwsh -File scripts/check-security-sandbox-contract.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] sandbox rollout governance contract suites" -Command {
    pwsh -File scripts/check-sandbox-rollout-governance-contract.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] sandbox egress + adapter allowlist contract suites" -Command {
    pwsh -File scripts/check-sandbox-egress-allowlist-contract.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] policy precedence contract suites" -Command {
    pwsh -File scripts/check-policy-precedence-contract.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] runtime budget admission contract suites" -Command {
    pwsh -File scripts/check-runtime-budget-admission-contract.ps1
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

Invoke-RequiredStep -StepLabel "[quality-gate] sandbox adapter conformance contract" -Command {
    pwsh -File scripts/check-sandbox-adapter-conformance-contract.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] memory contract conformance" -Command {
    pwsh -File scripts/check-memory-contract-conformance.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] memory scope and search governance contract" -Command {
    pwsh -File scripts/check-memory-scope-and-search-contract.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] observability export and diagnostics bundle contract" -Command {
    pwsh -File scripts/check-observability-export-and-bundle-contract.ps1
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

Invoke-RequiredStep -StepLabel "[quality-gate] go test -race ./..." -Command {
    go test -race ./...
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

Invoke-RequiredStep -StepLabel "[quality-gate] diagnostics query benchmark regression" -Command {
    pwsh -File scripts/check-diagnostics-query-performance-regression.ps1
}

Invoke-RequiredStep -StepLabel "[quality-gate] full-chain example smoke" -Command {
    pwsh -File scripts/check-full-chain-example-smoke.ps1
}

$scanMode = if ($env:BAYMAX_SECURITY_SCAN_MODE) { $env:BAYMAX_SECURITY_SCAN_MODE.Trim().ToLowerInvariant() } else { "strict" }
$govulncheckEnabled = if ($env:BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED) { $env:BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED.Trim().ToLowerInvariant() } else { "true" }
if ($govulncheckEnabled -eq "true") {
    Write-Host "[quality-gate] govulncheck (mode=$scanMode)"

    $proxyVars = @("HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy", "GIT_HTTP_PROXY", "GIT_HTTPS_PROXY")
    $savedProxy = @{}
    $needsDirect = $false
    foreach ($name in $proxyVars) {
        $value = [Environment]::GetEnvironmentVariable($name)
        if ([string]::IsNullOrWhiteSpace($value)) {
            continue
        }
        $savedProxy[$name] = $value
        $lower = $value.Trim().ToLowerInvariant()
        if ($lower.Contains("127.0.0.1:9") -or $lower.Contains("localhost:9") -or $lower.Contains("[::1]:9")) {
            $needsDirect = $true
        }
    }
    if ($needsDirect) {
        Write-Host "[quality-gate] detected placeholder proxy for govulncheck; run with direct connection"
        foreach ($name in $proxyVars) {
            Remove-Item -Path ("Env:" + $name) -ErrorAction SilentlyContinue
        }
    }

    # The only governance exception path: warn mode allows vulnerability findings without unblocking other strict checks.
    $allowWarn = ($scanMode -eq "warn")
    try {
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
    finally {
        if ($needsDirect) {
            foreach ($name in $savedProxy.Keys) {
                Set-Item -Path ("Env:" + $name) -Value $savedProxy[$name]
            }
        }
    }
}
else {
    Write-Host "[quality-gate] govulncheck disabled by BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED"
}

Write-Host "[quality-gate] done"
