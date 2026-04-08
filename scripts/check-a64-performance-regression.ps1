Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$enabled = if ($env:BAYMAX_A64_PERF_REGRESSION_ENABLED) {
    $env:BAYMAX_A64_PERF_REGRESSION_ENABLED.Trim().ToLowerInvariant()
}
else {
    "true"
}
if ($enabled -ne "true") {
    Write-Host "[a64-performance-regression] skipped by BAYMAX_A64_PERF_REGRESSION_ENABLED=$enabled"
    exit 0
}

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}

Write-Host "[a64-performance-regression] substep: context production hardening benchmark regression"
$subSteps = @(
    @{
        Name   = "context production hardening benchmark regression"
        Script = "scripts/check-context-production-hardening-benchmark-regression.ps1"
    },
    @{
        Name   = "diagnostics query benchmark regression"
        Script = "scripts/check-diagnostics-query-performance-regression.ps1"
    },
    @{
        Name   = "multi-agent performance benchmark regression"
        Script = "scripts/check-multi-agent-performance-regression.ps1"
    }
)

foreach ($step in $subSteps) {
    $stepName = [string]$step.Name
    $scriptPath = [string]$step.Script
    if ([string]::IsNullOrWhiteSpace($stepName) -or [string]::IsNullOrWhiteSpace($scriptPath)) {
        throw "[a64-performance-regression] invalid substep definition detected"
    }
    if (-not (Test-Path -LiteralPath $scriptPath)) {
        throw "[a64-performance-regression] required substep script missing: $scriptPath"
    }
    Write-Host "[a64-performance-regression] substep: $stepName"
    Invoke-NativeStrict -Label ("pwsh -File " + $scriptPath) -Command {
        pwsh -File $scriptPath
    }
}

Write-Host "[a64-performance-regression] passed"
