Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Get-EnvOrDefault {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [string]$Default = ""
    )
    if (Test-Path "Env:$Name") {
        $value = (Get-Item "Env:$Name").Value
        if (-not [string]::IsNullOrWhiteSpace($value)) {
            return $value
        }
    }
    return $Default
}

function Parse-PositiveDouble {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$Raw
    )
    if ([string]::IsNullOrWhiteSpace($Raw)) {
        throw "[diagnostics-query-bench] invalid numeric value for ${Name}: <empty>"
    }
    $parsed = 0.0
    $ok = [double]::TryParse($Raw, [System.Globalization.NumberStyles]::Float, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$parsed)
    if (-not $ok) {
        throw "[diagnostics-query-bench] invalid numeric value for ${Name}: $Raw"
    }
    if ($parsed -le 0) {
        throw "[diagnostics-query-bench] $Name must be > 0, got $Raw"
    }
    return $parsed
}

function Get-MetricFromLine {
    param(
        [Parameter(Mandatory = $true)][string]$Line,
        [Parameter(Mandatory = $true)][string]$Metric
    )
    $tokens = $Line -split "\s+"
    for ($i = 0; $i -lt $tokens.Length; $i++) {
        if ($tokens[$i] -eq $Metric -and $i -gt 0) {
            return $tokens[$i - 1]
        }
    }
    return $null
}

function Get-Median {
    param(
        [Parameter(Mandatory = $true)][double[]]$Values
    )
    if (-not $Values -or $Values.Count -eq 0) {
        throw "[diagnostics-query-bench] parse-failure reason=empty_samples"
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

$enabled = (Get-EnvOrDefault -Name "BAYMAX_DIAGNOSTICS_QUERY_BENCH_ENABLED" -Default "true").Trim().ToLowerInvariant()
if ($enabled -ne "true") {
    Write-Host "[diagnostics-query-bench] skipped by BAYMAX_DIAGNOSTICS_QUERY_BENCH_ENABLED=$enabled"
    exit 0
}

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if ([string]::IsNullOrWhiteSpace($env:GOCACHE)) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}

$baselineFile = Join-Path $PSScriptRoot "diagnostics-query-benchmark-baseline.env"
if (Test-Path $baselineFile) {
    Get-Content $baselineFile | ForEach-Object {
        $line = $_.Trim()
        if (-not $line -or $line.StartsWith("#")) { return }
        $parts = $line.Split("=", 2)
        if ($parts.Count -ne 2) {
            throw "[diagnostics-query-bench] invalid baseline line (expected KEY=VALUE): $line"
        }
        $key = $parts[0].Trim()
        if ($key -notmatch "^[A-Z0-9_]+$") {
            throw "[diagnostics-query-bench] invalid baseline key: $key"
        }
        $value = $parts[1].Trim()
        if (-not (Get-Item -Path "Env:$key" -ErrorAction SilentlyContinue)) {
            Set-Item -Path "Env:$key" -Value $value
        }
    }
}

$benchtime = Get-EnvOrDefault -Name "BAYMAX_DIAGNOSTICS_QUERY_BENCH_BENCHTIME" -Default "200ms"
if ([string]::IsNullOrWhiteSpace($benchtime)) {
    throw "[diagnostics-query-bench] invalid BAYMAX_DIAGNOSTICS_QUERY_BENCH_BENCHTIME=<empty>"
}

$countRaw = Get-EnvOrDefault -Name "BAYMAX_DIAGNOSTICS_QUERY_BENCH_COUNT" -Default "5"
$count = 0
if (-not [int]::TryParse($countRaw, [ref]$count) -or $count -le 0) {
    throw "[diagnostics-query-bench] invalid BAYMAX_DIAGNOSTICS_QUERY_BENCH_COUNT=$countRaw; expected positive integer"
}

$maxNsDegPct = Parse-PositiveDouble -Name "BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_NS_DEGRADATION_PCT" -Raw (Get-EnvOrDefault -Name "BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_NS_DEGRADATION_PCT" -Default "12")
$maxP95DegPct = Parse-PositiveDouble -Name "BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_P95_DEGRADATION_PCT" -Raw (Get-EnvOrDefault -Name "BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_P95_DEGRADATION_PCT" -Default "15")
$maxAllocsDegPct = Parse-PositiveDouble -Name "BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_ALLOCS_DEGRADATION_PCT" -Raw (Get-EnvOrDefault -Name "BAYMAX_DIAGNOSTICS_QUERY_BENCH_MAX_ALLOCS_DEGRADATION_PCT" -Default "12")

$benchmarks = @(
    @{ Name = "BenchmarkDiagnosticsQueryRuns"; Key = "QUERY_RUNS" },
    @{ Name = "BenchmarkDiagnosticsQueryMailbox"; Key = "QUERY_MAILBOX" },
    @{ Name = "BenchmarkDiagnosticsMailboxAggregates"; Key = "MAILBOX_AGGREGATES" }
)

foreach ($bench in $benchmarks) {
    foreach ($metric in @("NS_OP", "P95_NS_OP", "ALLOCS_OP")) {
        $baselineKey = "BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_$($bench.Key)_$metric"
        $baselineRaw = Get-EnvOrDefault -Name $baselineKey -Default ""
        [void](Parse-PositiveDouble -Name $baselineKey -Raw $baselineRaw)
    }
}

Write-Host "[diagnostics-query-bench] running benchmarks (benchtime=$benchtime, count=$count)"
$output = & go test ./integration -run '^$' -bench '^BenchmarkDiagnostics(QueryRuns|QueryMailbox|MailboxAggregates)$' -benchmem "-benchtime=$benchtime" "-count=$count" 2>&1
$output | ForEach-Object { Write-Host $_ }

$failed = $false
foreach ($bench in $benchmarks) {
    $lines = $output | Where-Object { $_ -match [regex]::Escape($bench.Name) }
    if (-not $lines -or $lines.Count -eq 0) {
        throw "[diagnostics-query-bench] parse-failure benchmark=$($bench.Name) reason=missing_output_line"
    }

    $nsSamples = New-Object 'System.Collections.Generic.List[double]'
    $p95Samples = New-Object 'System.Collections.Generic.List[double]'
    $allocsSamples = New-Object 'System.Collections.Generic.List[double]'
    foreach ($line in $lines) {
        $sampleNsRaw = Get-MetricFromLine -Line $line -Metric "ns/op"
        $sampleP95Raw = Get-MetricFromLine -Line $line -Metric "p95-ns/op"
        $sampleAllocsRaw = Get-MetricFromLine -Line $line -Metric "allocs/op"
        if ([string]::IsNullOrWhiteSpace($sampleNsRaw) -or [string]::IsNullOrWhiteSpace($sampleP95Raw) -or [string]::IsNullOrWhiteSpace($sampleAllocsRaw)) {
            throw "[diagnostics-query-bench] parse-failure benchmark=$($bench.Name) reason=missing_required_metric line=$line"
        }
        $nsSamples.Add((Parse-PositiveDouble -Name "$($bench.Name).sample.ns/op" -Raw $sampleNsRaw))
        $p95Samples.Add((Parse-PositiveDouble -Name "$($bench.Name).sample.p95-ns/op" -Raw $sampleP95Raw))
        $allocsSamples.Add((Parse-PositiveDouble -Name "$($bench.Name).sample.allocs/op" -Raw $sampleAllocsRaw))
    }

    $candidateNs = Get-Median -Values $nsSamples.ToArray()
    $candidateP95 = Get-Median -Values $p95Samples.ToArray()
    $candidateAllocs = Get-Median -Values $allocsSamples.ToArray()
    $candidateNs = Parse-PositiveDouble -Name "$($bench.Name).candidate.ns/op" -Raw "$candidateNs"
    $candidateP95 = Parse-PositiveDouble -Name "$($bench.Name).candidate.p95-ns/op" -Raw "$candidateP95"
    $candidateAllocs = Parse-PositiveDouble -Name "$($bench.Name).candidate.allocs/op" -Raw "$candidateAllocs"

    $baselineNs = Parse-PositiveDouble -Name "BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_$($bench.Key)_NS_OP" -Raw (Get-EnvOrDefault -Name "BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_$($bench.Key)_NS_OP" -Default "")
    $baselineP95 = Parse-PositiveDouble -Name "BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_$($bench.Key)_P95_NS_OP" -Raw (Get-EnvOrDefault -Name "BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_$($bench.Key)_P95_NS_OP" -Default "")
    $baselineAllocs = Parse-PositiveDouble -Name "BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_$($bench.Key)_ALLOCS_OP" -Raw (Get-EnvOrDefault -Name "BAYMAX_DIAGNOSTICS_QUERY_BENCH_BASELINE_$($bench.Key)_ALLOCS_OP" -Default "")

    $nsDegPct = (($candidateNs - $baselineNs) / $baselineNs) * 100
    $p95DegPct = (($candidateP95 - $baselineP95) / $baselineP95) * 100
    $allocsDegPct = (($candidateAllocs - $baselineAllocs) / $baselineAllocs) * 100

    Write-Host ("[diagnostics-query-bench] {0} ns/op baseline={1:N0} candidate={2:N0} degradation={3:N4}% (max={4:N4}%)" -f $bench.Name, $baselineNs, $candidateNs, $nsDegPct, $maxNsDegPct)
    Write-Host ("[diagnostics-query-bench] {0} p95-ns/op baseline={1:N0} candidate={2:N0} degradation={3:N4}% (max={4:N4}%)" -f $bench.Name, $baselineP95, $candidateP95, $p95DegPct, $maxP95DegPct)
    Write-Host ("[diagnostics-query-bench] {0} allocs/op baseline={1:N0} candidate={2:N0} degradation={3:N4}% (max={4:N4}%)" -f $bench.Name, $baselineAllocs, $candidateAllocs, $allocsDegPct, $maxAllocsDegPct)

    if ($nsDegPct -gt $maxNsDegPct -or $p95DegPct -gt $maxP95DegPct -or $allocsDegPct -gt $maxAllocsDegPct) {
        Write-Host "[diagnostics-query-bench] regression-threshold-exceeded benchmark=$($bench.Name)"
        $failed = $true
    }
}

if ($failed) {
    throw "[diagnostics-query-bench] failed"
}

Write-Host "[diagnostics-query-bench] passed"
