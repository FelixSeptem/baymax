Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

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

$line = $output | Where-Object { $_ -match "BenchmarkCA4PressureEvaluation" } | Select-Object -Last 1
if (-not $line) {
    throw "[ca4-bench] benchmark output not found"
}

$tokens = $line -split "\s+"
$candidateNs = $null
$candidateP95 = $null
for ($i = 0; $i -lt $tokens.Length; $i++) {
    if ($tokens[$i] -eq "ns/op" -and $i -gt 0) {
        $candidateNs = [double]$tokens[$i - 1]
    }
    if ($tokens[$i] -eq "p95-ns/op" -and $i -gt 0) {
        $candidateP95 = [double]$tokens[$i - 1]
    }
}
if ($null -eq $candidateNs -or $null -eq $candidateP95) {
    throw "[ca4-bench] failed to parse ns/op or p95-ns/op from benchmark line: $line"
}

$degPct = (($candidateNs - $baselineNs) / $baselineNs) * 100
$p95DegPct = (($candidateP95 - $baselineP95) / $baselineP95) * 100

Write-Host ("[ca4-bench] baseline ns/op={0}, candidate ns/op={1}, degradation={2:N4}%" -f [int64]$baselineNs, [int64]$candidateNs, $degPct)
Write-Host ("[ca4-bench] baseline p95-ns/op={0}, candidate p95-ns/op={1}, degradation={2:N4}%" -f [int64]$baselineP95, [int64]$candidateP95, $p95DegPct)

if ($degPct -gt $maxDegPct -or $p95DegPct -gt $maxP95DegPct) {
    throw "[ca4-bench] regression threshold exceeded (ns>$maxDegPct% or p95>$maxP95DegPct%)"
}

Write-Host "[ca4-bench] passed"
