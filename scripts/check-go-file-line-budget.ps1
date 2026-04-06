Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$policyPath = Join-Path $repoRoot "openspec/governance/go-file-line-budget-policy.env"
$exceptionsPath = Join-Path $repoRoot "openspec/governance/go-file-line-budget-exceptions.csv"

if (-not (Test-Path -LiteralPath $policyPath)) {
    throw "[go-file-line-budget] missing policy file: $policyPath"
}
if (-not (Test-Path -LiteralPath $exceptionsPath)) {
    throw "[go-file-line-budget] missing exception file: $exceptionsPath"
}

$policy = @{}
foreach ($line in Get-Content -LiteralPath $policyPath) {
    $trimmed = $line.Trim()
    if ([string]::IsNullOrWhiteSpace($trimmed)) {
        continue
    }
    if ($trimmed.StartsWith("#")) {
        continue
    }
    $pair = $trimmed.Split("=", 2)
    if ($pair.Count -ne 2) {
        throw "[go-file-line-budget] invalid policy line: $line"
    }
    $policy[$pair[0].Trim()] = $pair[1].Trim()
}

$warnRaw = if ($policy.ContainsKey("BAYMAX_GO_LINE_BUDGET_WARN")) { $policy["BAYMAX_GO_LINE_BUDGET_WARN"] } else { "800" }
$hardRaw = if ($policy.ContainsKey("BAYMAX_GO_LINE_BUDGET_HARD")) { $policy["BAYMAX_GO_LINE_BUDGET_HARD"] } else { "1200" }
$excludedPrefix = if ($policy.ContainsKey("BAYMAX_GO_LINE_BUDGET_EXCLUDED_PREFIX")) { $policy["BAYMAX_GO_LINE_BUDGET_EXCLUDED_PREFIX"] } else { "openspec/" }

$warnThreshold = 0
$hardThreshold = 0
if (-not [int]::TryParse($warnRaw, [ref]$warnThreshold) -or $warnThreshold -le 0) {
    throw "[go-file-line-budget] warn threshold must be positive integer, got: $warnRaw"
}
if (-not [int]::TryParse($hardRaw, [ref]$hardThreshold) -or $hardThreshold -le 0) {
    throw "[go-file-line-budget] hard threshold must be positive integer, got: $hardRaw"
}
if ($warnThreshold -ge $hardThreshold) {
    throw "[go-file-line-budget] warn threshold must be < hard threshold: warn=$warnThreshold hard=$hardThreshold"
}

$exceptionRows = Import-Csv -LiteralPath $exceptionsPath
$exceptions = @{}
foreach ($row in $exceptionRows) {
    if ([string]::IsNullOrWhiteSpace($row.path)) {
        continue
    }
    $baseline = 0
    if (-not [int]::TryParse($row.baseline_lines, [ref]$baseline)) {
        throw "[go-file-line-budget] invalid baseline_lines in exception row: $($row.path),$($row.baseline_lines)"
    }
    $normalized = [PSCustomObject]@{
        path         = $row.path
        owner        = $row.owner
        reason       = $row.reason
        expiry       = $row.expiry
        baseline     = $baseline
        allow_growth = if ($null -eq $row.allow_growth) { "false" } else { $row.allow_growth.ToLowerInvariant() }
    }
    $exceptions[$row.path] = $normalized
}

$files = @(Invoke-NativeCaptureStrict -Label "git ls-files '*.go'" -Command {
        git ls-files '*.go'
    })

$today = (Get-Date).ToString("yyyy-MM-dd")
$checked = 0
$warnHits = 0
$hardHits = 0
$violations = New-Object 'System.Collections.Generic.List[string]'

foreach ($file in $files) {
    if ([string]::IsNullOrWhiteSpace($file)) {
        continue
    }
    if ($file.StartsWith($excludedPrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
        continue
    }
    if ($file.EndsWith("_test.go", [System.StringComparison]::OrdinalIgnoreCase)) {
        continue
    }
    if (-not (Test-Path -LiteralPath $file)) {
        continue
    }

    $lines = (Get-Content -LiteralPath $file | Measure-Object -Line).Lines
    $checked += 1

    if ($lines -gt $warnThreshold) {
        $warnHits += 1
    }
    if ($lines -le $hardThreshold) {
        continue
    }

    $hardHits += 1
    if ($exceptions.ContainsKey($file)) {
        $entry = $exceptions[$file]
        if (-not [string]::IsNullOrWhiteSpace($entry.expiry) -and $entry.expiry -lt $today) {
            $violations.Add("[go-file-line-budget][violation] expired exception: $file expiry=$($entry.expiry) today=$today") | Out-Null
            continue
        }
        if ($entry.allow_growth -ne "true" -and $lines -gt $entry.baseline) {
            $violations.Add("[go-file-line-budget][violation] oversized debt expanded: $file lines=$lines baseline=$($entry.baseline)") | Out-Null
            continue
        }
        Write-Host "[go-file-line-budget][debt] $file lines=$lines baseline=$($entry.baseline) owner=$($entry.owner) expiry=$($entry.expiry)"
    }
    else {
        $violations.Add("[go-file-line-budget][violation] oversized file without exception: $file lines=$lines hard=$hardThreshold") | Out-Null
    }
}

foreach ($key in $exceptions.Keys) {
    if (-not (Test-Path -LiteralPath $key)) {
        $violations.Add("[go-file-line-budget][violation] stale exception path missing: $key") | Out-Null
        continue
    }
    $lines = (Get-Content -LiteralPath $key | Measure-Object -Line).Lines
    if ($lines -le $hardThreshold) {
        $violations.Add("[go-file-line-budget][violation] stale exception no longer needed: $key lines=$lines hard=$hardThreshold") | Out-Null
    }
}

Write-Host "[go-file-line-budget] checked=$checked warn_threshold=$warnThreshold hard_threshold=$hardThreshold warn_hits=$warnHits hard_hits=$hardHits"
if ($violations.Count -gt 0) {
    foreach ($msg in $violations) {
        Write-Host $msg
    }
    throw "[go-file-line-budget] failed: violations=$($violations.Count)"
}
Write-Host "[go-file-line-budget] passed"
