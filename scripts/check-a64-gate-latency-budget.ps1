Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

function Get-EnvOrDefault {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [string]$Default = ""
    )
    $value = [Environment]::GetEnvironmentVariable($Name)
    if ([string]::IsNullOrWhiteSpace($value)) {
        return $Default
    }
    return $value
}

function Set-EnvIfUnset {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$Value
    )
    $existing = [Environment]::GetEnvironmentVariable($Name)
    if ([string]::IsNullOrWhiteSpace($existing)) {
        Set-Item -Path ("Env:" + $Name) -Value $Value
    }
}

function Load-EnvDefaultsFromFile {
    param(
        [Parameter(Mandatory = $true)][string]$Path
    )
    Get-Content -LiteralPath $Path | ForEach-Object {
        $line = $_.Trim()
        if (-not $line -or $line.StartsWith("#")) {
            return
        }
        $parts = $line.Split("=", 2)
        if ($parts.Count -ne 2) {
            throw "[a64-gate-latency-budget] invalid baseline line (expected KEY=VALUE): $line"
        }
        $key = $parts[0].Trim()
        if ($key -notmatch "^[A-Z0-9_]+$") {
            throw "[a64-gate-latency-budget] invalid baseline key: $key"
        }
        $value = $parts[1].Trim()
        Set-EnvIfUnset -Name $key -Value $value
    }
}

function Parse-PositiveInt {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$Raw
    )
    $parsed = 0
    if (-not [int]::TryParse($Raw, [ref]$parsed) -or $parsed -le 0) {
        throw "[a64-gate-latency-budget] $Name must be a positive integer, got: $Raw"
    }
    return $parsed
}

$defaultBaselineFile = Join-Path $PSScriptRoot "a64-gate-latency-baseline.env"
$baselineFile = (Get-EnvOrDefault -Name "BAYMAX_A64_GATE_LATENCY_BASELINE_FILE" -Default $defaultBaselineFile).Trim()
if ($baselineFile -ne "") {
    if (-not (Test-Path -LiteralPath $baselineFile)) {
        throw "[a64-gate-latency-budget] baseline file not found: $baselineFile"
    }
    Load-EnvDefaultsFromFile -Path $baselineFile
}

$enabled = (Get-EnvOrDefault -Name "BAYMAX_A64_GATE_LATENCY_ENABLED" -Default "true").Trim().ToLowerInvariant()
if ($enabled -ne "true") {
    Write-Host "[a64-gate-latency-budget] skipped by BAYMAX_A64_GATE_LATENCY_ENABLED=$enabled"
    exit 0
}

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}

$maxStepSeconds = Parse-PositiveInt -Name "BAYMAX_A64_GATE_LATENCY_MAX_STEP_SECONDS" -Raw (Get-EnvOrDefault -Name "BAYMAX_A64_GATE_LATENCY_MAX_STEP_SECONDS" -Default "600")
$maxTotalSeconds = Parse-PositiveInt -Name "BAYMAX_A64_GATE_LATENCY_MAX_TOTAL_SECONDS" -Raw (Get-EnvOrDefault -Name "BAYMAX_A64_GATE_LATENCY_MAX_TOTAL_SECONDS" -Default "1200")

$steps = @(
    @{
        Name   = "a64 impacted gate selection"
        Script = "scripts/check-a64-impacted-gate-selection.ps1"
    },
    @{
        Name   = "a64 semantic stability gate"
        Script = "scripts/check-a64-semantic-stability-contract.ps1"
    },
    @{
        Name   = "a64 performance regression gate"
        Script = "scripts/check-a64-performance-regression.ps1"
    }
)

$records = New-Object 'System.Collections.Generic.List[object]'
$totalStopwatch = [System.Diagnostics.Stopwatch]::StartNew()
foreach ($step in $steps) {
    $name = [string]$step.Name
    $scriptPath = [string]$step.Script
    if ([string]::IsNullOrWhiteSpace($name) -or [string]::IsNullOrWhiteSpace($scriptPath)) {
        throw "[a64-gate-latency-budget] invalid step definition"
    }
    if (-not (Test-Path -LiteralPath $scriptPath)) {
        throw "[a64-gate-latency-budget] required script missing: $scriptPath"
    }

    Write-Host "[a64-gate-latency-budget] step start: $name"
    $stepStopwatch = [System.Diagnostics.Stopwatch]::StartNew()
    Invoke-NativeStrict -Label ("pwsh -File " + $scriptPath) -Command {
        pwsh -File $scriptPath
    }
    $stepStopwatch.Stop()
    $stepSeconds = [Math]::Round($stepStopwatch.Elapsed.TotalSeconds, 3)
    $records.Add([ordered]@{
            step     = $name
            script   = $scriptPath
            seconds  = $stepSeconds
            max_step = $maxStepSeconds
            within   = ($stepStopwatch.Elapsed.TotalSeconds -le $maxStepSeconds)
        }) | Out-Null
    Write-Host "[a64-gate-latency-budget] step done: $name seconds=$stepSeconds"
    if ($stepStopwatch.Elapsed.TotalSeconds -gt $maxStepSeconds) {
        throw "[a64-gate-latency-budget] step budget exceeded: $name elapsed=${stepSeconds}s max=${maxStepSeconds}s"
    }
}
$totalStopwatch.Stop()
$totalSeconds = [Math]::Round($totalStopwatch.Elapsed.TotalSeconds, 3)
if ($totalStopwatch.Elapsed.TotalSeconds -gt $maxTotalSeconds) {
    throw "[a64-gate-latency-budget] total budget exceeded: elapsed=${totalSeconds}s max=${maxTotalSeconds}s"
}

$report = [ordered]@{
    generated_at       = (Get-Date).ToString("o")
    max_total_seconds  = $maxTotalSeconds
    max_step_seconds   = $maxStepSeconds
    total_seconds      = $totalSeconds
    total_within       = ($totalStopwatch.Elapsed.TotalSeconds -le $maxTotalSeconds)
    steps              = $records
}
$json = $report | ConvertTo-Json -Depth 6
Write-Host "[a64-gate-latency-budget] report:"
Write-Host $json

$outputPath = (Get-EnvOrDefault -Name "BAYMAX_A64_GATE_LATENCY_REPORT_PATH" -Default "").Trim()
if ($outputPath -ne "") {
    $parent = Split-Path -Parent $outputPath
    if ($parent -and -not (Test-Path -LiteralPath $parent)) {
        New-Item -ItemType Directory -Path $parent -Force | Out-Null
    }
    Set-Content -LiteralPath $outputPath -Value $json -NoNewline
    Write-Host "[a64-gate-latency-budget] report written to $outputPath"
}

Write-Host "[a64-gate-latency-budget] passed"
