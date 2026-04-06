Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$canonicalPrefix = "BAYMAX_CONTEXT_PRODUCTION_HARDENING_BENCH"
$benchmarkRegex = "^(BenchmarkContextProductionHardeningPressureEvaluation)$"

function Write-BenchLog {
    param([Parameter(Mandatory = $true)][string]$Message)
    Write-Host "[context-production-hardening-bench] $Message"
}

function Get-BenchSetting {
    param(
        [Parameter(Mandatory = $true)][string]$CanonicalName,
        [Parameter(Mandatory = $false)][string]$DefaultValue = ""
    )
    $canonical = [Environment]::GetEnvironmentVariable($CanonicalName)
    if (-not [string]::IsNullOrWhiteSpace($canonical)) {
        return $canonical
    }
    return $DefaultValue
}

function Get-Median {
    param([Parameter(Mandatory = $true)][double[]]$Values)
    if (-not $Values -or $Values.Count -eq 0) {
        throw "[context-production-hardening-bench] no benchmark samples provided for median"
    }
    $sorted = $Values | Sort-Object
    $count = $sorted.Count
    if ($count % 2 -eq 1) {
        $mid = [int][math]::Floor($count / 2)
        return [double]$sorted[$mid]
    }
    $left = [double]$sorted[($count / 2) - 1]
    $right = [double]$sorted[$count / 2]
    return ($left + $right) / 2
}

$enabled = (Get-BenchSetting -CanonicalName "${canonicalPrefix}_ENABLED" -DefaultValue "true").Trim().ToLowerInvariant()
if ($enabled -ne "true") {
    Write-BenchLog "skipped by ${canonicalPrefix}_ENABLED=$enabled"
    exit 0
}

if ([string]::IsNullOrWhiteSpace($env:GOCACHE)) {
    $env:GOCACHE = (Join-Path (Resolve-Path (Join-Path $PSScriptRoot "..")) ".gocache")
}

$canonicalBaselineFile = Join-Path $PSScriptRoot "context-production-hardening-benchmark-baseline.env"
if (Test-Path $canonicalBaselineFile) {
    Get-Content $canonicalBaselineFile | ForEach-Object {
        $line = $_.Trim()
        if (-not $line -or $line.StartsWith("#")) { return }
        $parts = $line.Split("=", 2)
        if ($parts.Count -eq 2 -and -not (Get-Item -Path "Env:$($parts[0])" -ErrorAction SilentlyContinue)) {
            Set-Item -Path "Env:$($parts[0])" -Value $parts[1]
        }
    }
}

$benchtime = Get-BenchSetting -CanonicalName "${canonicalPrefix}_BENCHTIME" -DefaultValue "150ms"
$count = Get-BenchSetting -CanonicalName "${canonicalPrefix}_COUNT" -DefaultValue "3"
$maxDegPctRaw = Get-BenchSetting -CanonicalName "${canonicalPrefix}_MAX_DEGRADATION_PCT" -DefaultValue "5"
$maxP95DegPctRaw = Get-BenchSetting -CanonicalName "${canonicalPrefix}_MAX_P95_DEGRADATION_PCT" -DefaultValue "8"
$maxDegPct = [double]$maxDegPctRaw
$maxP95DegPct = [double]$maxP95DegPctRaw
$baselineNsRaw = Get-BenchSetting -CanonicalName "${canonicalPrefix}_BASELINE_NS_OP"
$baselineP95Raw = Get-BenchSetting -CanonicalName "${canonicalPrefix}_BASELINE_P95_NS_OP"
if ([string]::IsNullOrWhiteSpace($baselineNsRaw) -or [string]::IsNullOrWhiteSpace($baselineP95Raw)) {
    throw "[context-production-hardening-bench] missing baseline values; set ${canonicalPrefix}_BASELINE_NS_OP and ${canonicalPrefix}_BASELINE_P95_NS_OP"
}
$baselineNs = [double]$baselineNsRaw
$baselineP95 = [double]$baselineP95Raw

Write-BenchLog "running benchmark (benchtime=$benchtime, count=$count)"
$output = & go test ./integration -run '^$' -bench $benchmarkRegex -benchmem "-benchtime=$benchtime" "-count=$count" 2>&1
$output | ForEach-Object { Write-Host $_ }
$outputText = ($output | Out-String)
$nsMatches = [regex]::Matches($outputText, "BenchmarkContextProductionHardeningPressureEvaluation[^\r\n]*?([0-9]+(?:\.[0-9]+)?)\s+ns/op")
$p95Matches = [regex]::Matches($outputText, "BenchmarkContextProductionHardeningPressureEvaluation[^\r\n]*?([0-9]+(?:\.[0-9]+)?)\s+p95-ns/op")
if ($nsMatches.Count -eq 0 -or $p95Matches.Count -eq 0) {
    throw "[context-production-hardening-bench] benchmark output not found"
}

$nsSamples = New-Object 'System.Collections.Generic.List[double]'
$p95Samples = New-Object 'System.Collections.Generic.List[double]'
foreach ($m in $nsMatches) {
    $nsSamples.Add([double]$m.Groups[1].Value)
}
foreach ($m in $p95Matches) {
    $p95Samples.Add([double]$m.Groups[1].Value)
}
if ($nsSamples.Count -eq 0 -or $p95Samples.Count -eq 0 -or $nsSamples.Count -ne $p95Samples.Count) {
    throw "[context-production-hardening-bench] failed to parse aligned ns/op and p95-ns/op samples"
}

$candidateNs = Get-Median -Values $nsSamples.ToArray()
$candidateP95 = Get-Median -Values $p95Samples.ToArray()

$degPct = (($candidateNs - $baselineNs) / $baselineNs) * 100
$p95DegPct = (($candidateP95 - $baselineP95) / $baselineP95) * 100

Write-BenchLog ("baseline ns/op={0}, candidate ns/op={1}, degradation={2:N4}%" -f [int64]$baselineNs, [int64]$candidateNs, $degPct)
Write-BenchLog ("baseline p95-ns/op={0}, candidate p95-ns/op={1}, degradation={2:N4}%" -f [int64]$baselineP95, [int64]$candidateP95, $p95DegPct)

if ($degPct -gt $maxDegPct -or $p95DegPct -gt $maxP95DegPct) {
    throw "[context-production-hardening-bench] regression threshold exceeded (ns>$maxDegPct% or p95>$maxP95DegPct%)"
}

Write-BenchLog "passed"
