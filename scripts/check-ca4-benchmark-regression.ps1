Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Get-Median {
    param([Parameter(Mandatory = $true)][double[]]$Values)
    if (-not $Values -or $Values.Count -eq 0) {
        throw "[ca4-bench] no benchmark samples provided for median"
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

$enabled = if ($env:BAYMAX_CA4_BENCH_ENABLED) { $env:BAYMAX_CA4_BENCH_ENABLED.Trim().ToLowerInvariant() } else { "true" }
if ($enabled -ne "true") {
    Write-Host "[ca4-bench] skipped by BAYMAX_CA4_BENCH_ENABLED=$enabled"
    exit 0
}

if ([string]::IsNullOrWhiteSpace($env:GOCACHE)) {
    $env:GOCACHE = (Join-Path (Resolve-Path (Join-Path $PSScriptRoot "..")) ".gocache")
}

$baselineFile = Join-Path $PSScriptRoot "ca4-benchmark-baseline.env"
if (Test-Path $baselineFile) {
    Get-Content $baselineFile | ForEach-Object {
        $line = $_.Trim()
        if (-not $line -or $line.StartsWith("#")) { return }
        $parts = $line.Split("=", 2)
        if ($parts.Count -eq 2 -and -not (Get-Item -Path "Env:$($parts[0])" -ErrorAction SilentlyContinue)) {
            Set-Item -Path "Env:$($parts[0])" -Value $parts[1]
        }
    }
}

$benchtime = if ($env:BAYMAX_CA4_BENCH_BENCHTIME) { $env:BAYMAX_CA4_BENCH_BENCHTIME } else { "150ms" }
$count = if ($env:BAYMAX_CA4_BENCH_COUNT) { $env:BAYMAX_CA4_BENCH_COUNT } else { "3" }
$maxDegPctRaw = if ($env:BAYMAX_CA4_BENCH_MAX_DEGRADATION_PCT) { $env:BAYMAX_CA4_BENCH_MAX_DEGRADATION_PCT } else { "5" }
$maxP95DegPctRaw = if ($env:BAYMAX_CA4_BENCH_MAX_P95_DEGRADATION_PCT) { $env:BAYMAX_CA4_BENCH_MAX_P95_DEGRADATION_PCT } else { "8" }
$maxDegPct = [double]$maxDegPctRaw
$maxP95DegPct = [double]$maxP95DegPctRaw
$baselineNsRaw = $env:BAYMAX_CA4_BENCH_BASELINE_NS_OP
$baselineP95Raw = $env:BAYMAX_CA4_BENCH_BASELINE_P95_NS_OP
if ([string]::IsNullOrWhiteSpace($baselineNsRaw) -or [string]::IsNullOrWhiteSpace($baselineP95Raw)) {
    throw "[ca4-bench] missing baseline values; set BAYMAX_CA4_BENCH_BASELINE_NS_OP and BAYMAX_CA4_BENCH_BASELINE_P95_NS_OP"
}
$baselineNs = [double]$baselineNsRaw
$baselineP95 = [double]$baselineP95Raw

Write-Host "[ca4-bench] running benchmark (benchtime=$benchtime, count=$count)"
$output = & go test ./integration -run '^$' -bench '^BenchmarkCA4PressureEvaluation$' -benchmem "-benchtime=$benchtime" "-count=$count" 2>&1
$output | ForEach-Object { Write-Host $_ }
$outputText = ($output | Out-String)
$nsMatches = [regex]::Matches($outputText, "BenchmarkCA4PressureEvaluation[^\r\n]*?([0-9]+(?:\.[0-9]+)?)\s+ns/op")
$p95Matches = [regex]::Matches($outputText, "BenchmarkCA4PressureEvaluation[^\r\n]*?([0-9]+(?:\.[0-9]+)?)\s+p95-ns/op")
if ($nsMatches.Count -eq 0 -or $p95Matches.Count -eq 0) {
    throw "[ca4-bench] benchmark output not found"
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
    throw "[ca4-bench] failed to parse aligned ns/op and p95-ns/op samples"
}

$candidateNs = Get-Median -Values $nsSamples.ToArray()
$candidateP95 = Get-Median -Values $p95Samples.ToArray()

$degPct = (($candidateNs - $baselineNs) / $baselineNs) * 100
$p95DegPct = (($candidateP95 - $baselineP95) / $baselineP95) * 100

Write-Host ("[ca4-bench] baseline ns/op={0}, candidate ns/op={1}, degradation={2:N4}%" -f [int64]$baselineNs, [int64]$candidateNs, $degPct)
Write-Host ("[ca4-bench] baseline p95-ns/op={0}, candidate p95-ns/op={1}, degradation={2:N4}%" -f [int64]$baselineP95, [int64]$candidateP95, $p95DegPct)

if ($degPct -gt $maxDegPct -or $p95DegPct -gt $maxP95DegPct) {
    throw "[ca4-bench] regression threshold exceeded (ns>$maxDegPct% or p95>$maxP95DegPct%)"
}

Write-Host "[ca4-bench] passed"
