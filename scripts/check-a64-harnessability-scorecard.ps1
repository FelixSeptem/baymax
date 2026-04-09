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
            throw "[a64-harnessability-scorecard] invalid baseline line (expected KEY=VALUE): $line"
        }
        $key = $parts[0].Trim()
        if ($key -notmatch "^[A-Z0-9_]+$") {
            throw "[a64-harnessability-scorecard] invalid baseline key: $key"
        }
        $value = $parts[1].Trim()
        Set-EnvIfUnset -Name $key -Value $value
    }
}

function Parse-PositiveDouble {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$Raw
    )
    $parsed = 0.0
    $ok = [double]::TryParse($Raw, [System.Globalization.NumberStyles]::Float, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$parsed)
    if (-not $ok -or $parsed -le 0) {
        throw "[a64-harnessability-scorecard] $Name must be a positive number, got: $Raw"
    }
    return $parsed
}

function Parse-NonNegativeInt {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$Raw
    )
    $parsed = 0
    if (-not [int]::TryParse($Raw, [ref]$parsed) -or $parsed -lt 0) {
        throw "[a64-harnessability-scorecard] $Name must be a non-negative integer, got: $Raw"
    }
    return $parsed
}

function Parse-Percent {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$Raw
    )
    $value = 0.0
    $ok = [double]::TryParse($Raw, [System.Globalization.NumberStyles]::Float, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$value)
    if (-not $ok -or $value -lt 0) {
        throw "[a64-harnessability-scorecard] $Name must be within [0,100], got: $Raw"
    }
    if ($value -gt 100) {
        throw "[a64-harnessability-scorecard] $Name must be <= 100, got: $Raw"
    }
    return $value
}

function Parse-SignedPercent {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$Raw
    )
    $parsed = 0.0
    $ok = [double]::TryParse($Raw, [System.Globalization.NumberStyles]::Float, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$parsed)
    if (-not $ok) {
        throw "[a64-harnessability-scorecard] $Name must be numeric, got: $Raw"
    }
    if ($parsed -lt -100 -or $parsed -gt 100) {
        throw "[a64-harnessability-scorecard] $Name must be within [-100,100], got: $Raw"
    }
    return $parsed
}

function Parse-Bool {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$Raw
    )
    $value = $Raw.Trim().ToLowerInvariant()
    if ($value -eq "true") {
        return $true
    }
    if ($value -eq "false") {
        return $false
    }
    throw "[a64-harnessability-scorecard] $Name must be true|false, got: $Raw"
}

function Parse-Tier {
    param([Parameter(Mandatory = $true)][string]$Raw)
    $tier = $Raw.Trim().ToLowerInvariant()
    if ($tier -in @("lightweight", "standard", "enhanced")) {
        return $tier
    }
    throw "[a64-harnessability-scorecard] BAYMAX_A64_HARNESS_COMPLEXITY_TIER must be lightweight|standard|enhanced, got: $Raw"
}

function Round2 {
    param([Parameter(Mandatory = $true)][double]$Value)
    return [Math]::Round($Value, 2)
}

function Get-OverheadPct {
    param(
        [Parameter(Mandatory = $true)][double]$Measured,
        [Parameter(Mandatory = $true)][double]$Baseline
    )
    return (($Measured - $Baseline) / $Baseline) * 100
}

function Test-ContainsToken {
    param(
        [Parameter(Mandatory = $true)][string]$Text,
        [Parameter(Mandatory = $true)][string]$Token
    )
    return $Text.Contains($Token)
}

$defaultBaselineFile = Join-Path $PSScriptRoot "a64-harnessability-scorecard-baseline.env"
$baselineFile = (Get-EnvOrDefault -Name "BAYMAX_A64_HARNESS_SCORECARD_BASELINE_FILE" -Default $defaultBaselineFile).Trim()
if ($baselineFile -ne "") {
    if (-not (Test-Path -LiteralPath $baselineFile)) {
        throw "[a64-harnessability-scorecard] baseline file not found: $baselineFile"
    }
    Load-EnvDefaultsFromFile -Path $baselineFile
}

$enabled = (Get-EnvOrDefault -Name "BAYMAX_A64_HARNESS_SCORECARD_ENABLED" -Default "true").Trim().ToLowerInvariant()
if ($enabled -ne "true") {
    Write-Host "[a64-harnessability-scorecard] skipped by BAYMAX_A64_HARNESS_SCORECARD_ENABLED=$enabled"
    exit 0
}

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}

$failures = New-Object 'System.Collections.Generic.List[string]'

# Contract coverage: reuse impacted-gate selection in full mode.
$impactedReportPath = (Get-EnvOrDefault -Name "BAYMAX_A64_SCORECARD_IMPACTED_REPORT_PATH" -Default (Join-Path $repoRoot ".artifacts/a64/impacted-full-report.json")).Trim()
$impactedReportParent = Split-Path -Parent $impactedReportPath
if ($impactedReportParent -and -not (Test-Path -LiteralPath $impactedReportParent)) {
    New-Item -ItemType Directory -Path $impactedReportParent -Force | Out-Null
}
$prevSelectionMode = [Environment]::GetEnvironmentVariable("BAYMAX_A64_GATE_SELECTION_MODE")
$prevImpactedReport = [Environment]::GetEnvironmentVariable("BAYMAX_A64_IMPACTED_REPORT_PATH")
try {
    Set-Item -Path Env:BAYMAX_A64_GATE_SELECTION_MODE -Value "full"
    Set-Item -Path Env:BAYMAX_A64_IMPACTED_REPORT_PATH -Value $impactedReportPath
    Invoke-NativeStrict -Label "pwsh -File scripts/check-a64-impacted-gate-selection.ps1" -Command {
        pwsh -File scripts/check-a64-impacted-gate-selection.ps1
    }
}
finally {
    if ($null -eq $prevSelectionMode) {
        Remove-Item -Path Env:BAYMAX_A64_GATE_SELECTION_MODE -ErrorAction SilentlyContinue
    }
    else {
        Set-Item -Path Env:BAYMAX_A64_GATE_SELECTION_MODE -Value $prevSelectionMode
    }
    if ($null -eq $prevImpactedReport) {
        Remove-Item -Path Env:BAYMAX_A64_IMPACTED_REPORT_PATH -ErrorAction SilentlyContinue
    }
    else {
        Set-Item -Path Env:BAYMAX_A64_IMPACTED_REPORT_PATH -Value $prevImpactedReport
    }
}

if (-not (Test-Path -LiteralPath $impactedReportPath)) {
    throw "[a64-harnessability-scorecard] impacted report missing: $impactedReportPath"
}
$impactedReport = Get-Content -Raw -LiteralPath $impactedReportPath | ConvertFrom-Json
$impactedItems = @()
if ($null -ne $impactedReport.impacted_s_items) {
    $impactedItems = @($impactedReport.impacted_s_items)
}
$contractCoveragePct = Round2 -Value (($impactedItems.Count / 10.0) * 100.0)
$minContractCoveragePct = Parse-Percent -Name "BAYMAX_A64_HARNESS_SCORECARD_MIN_CONTRACT_COVERAGE_PCT" -Raw (Get-EnvOrDefault -Name "BAYMAX_A64_HARNESS_SCORECARD_MIN_CONTRACT_COVERAGE_PCT" -Default "100")
$contractCoverageWithin = ($contractCoveragePct -ge $minContractCoveragePct)
if (-not $contractCoverageWithin) {
    $failures.Add(("contract_coverage_pct={0} < min={1}" -f $contractCoveragePct, $minContractCoveragePct)) | Out-Null
}

# Drift health: fixture count + unclassified drift count.
$driftFixtureCount = @(Get-ChildItem -Path "tool/diagnosticsreplay/testdata" -File -ErrorAction SilentlyContinue | Where-Object {
            $_.Name -match "(?i)(inferential|drift)"
        }).Count
$minDriftFixtureCount = Parse-NonNegativeInt -Name "BAYMAX_A64_HARNESS_SCORECARD_MIN_DRIFT_FIXTURE_COUNT" -Raw (Get-EnvOrDefault -Name "BAYMAX_A64_HARNESS_SCORECARD_MIN_DRIFT_FIXTURE_COUNT" -Default "2")
$unclassifiedDriftCount = Parse-NonNegativeInt -Name "BAYMAX_A64_HARNESS_SCORECARD_UNCLASSIFIED_DRIFT_COUNT" -Raw (Get-EnvOrDefault -Name "BAYMAX_A64_HARNESS_SCORECARD_UNCLASSIFIED_DRIFT_COUNT" -Default "0")
$maxUnclassifiedDriftCount = Parse-NonNegativeInt -Name "BAYMAX_A64_HARNESS_SCORECARD_MAX_UNCLASSIFIED_DRIFT_COUNT" -Raw (Get-EnvOrDefault -Name "BAYMAX_A64_HARNESS_SCORECARD_MAX_UNCLASSIFIED_DRIFT_COUNT" -Default "0")
$driftWithin = ($driftFixtureCount -ge $minDriftFixtureCount -and $unclassifiedDriftCount -le $maxUnclassifiedDriftCount)
if (-not $driftWithin) {
    $failures.Add(("drift_stats fixture_count={0} (min={1}) unclassified={2} (max={3})" -f $driftFixtureCount, $minDriftFixtureCount, $unclassifiedDriftCount, $maxUnclassifiedDriftCount)) | Out-Null
}

# Gate coverage: ensure required A64 gates are wired in both shell + PowerShell quality gates.
$requiredGatePairs = @(
    @{ Name = "a64 impacted gate selection"; Shell = "check-a64-impacted-gate-selection.sh"; PowerShell = "check-a64-impacted-gate-selection.ps1" },
    @{ Name = "a64 semantic stability gate"; Shell = "check-a64-semantic-stability-contract.sh"; PowerShell = "check-a64-semantic-stability-contract.ps1" },
    @{ Name = "a64 performance regression gate"; Shell = "check-a64-performance-regression.sh"; PowerShell = "check-a64-performance-regression.ps1" },
    @{ Name = "a64 harnessability scorecard"; Shell = "check-a64-harnessability-scorecard.sh"; PowerShell = "check-a64-harnessability-scorecard.ps1" }
)
$qualityGateShell = Get-Content -Raw "scripts/check-quality-gate.sh"
$qualityGatePS = Get-Content -Raw "scripts/check-quality-gate.ps1"
$coveredGateCount = 0
$gateCoverageDetails = New-Object 'System.Collections.Generic.List[object]'
foreach ($pair in $requiredGatePairs) {
    $shellFound = Test-ContainsToken -Text $qualityGateShell -Token $pair.Shell
    $psFound = Test-ContainsToken -Text $qualityGatePS -Token $pair.PowerShell
    if ($shellFound -and $psFound) {
        $coveredGateCount++
    }
    $gateCoverageDetails.Add([ordered]@{
            gate            = $pair.Name
            shell_token     = $pair.Shell
            powershell_token = $pair.PowerShell
            shell_found     = $shellFound
            powershell_found = $psFound
            covered         = ($shellFound -and $psFound)
        }) | Out-Null
}
$gateCoveragePct = Round2 -Value (($coveredGateCount / [double]$requiredGatePairs.Count) * 100.0)
$minGateCoveragePct = Parse-Percent -Name "BAYMAX_A64_HARNESS_SCORECARD_MIN_GATE_COVERAGE_PCT" -Raw (Get-EnvOrDefault -Name "BAYMAX_A64_HARNESS_SCORECARD_MIN_GATE_COVERAGE_PCT" -Default "100")
$gateCoverageWithin = ($gateCoveragePct -ge $minGateCoveragePct)
if (-not $gateCoverageWithin) {
    $failures.Add(("gate_coverage_pct={0} < min={1}" -f $gateCoveragePct, $minGateCoveragePct)) | Out-Null
}

# Docs consistency: marker-driven, machine-auditable checks.
$docsIssueCount = 0
$governanceIndexPath = "openspec/changes/introduce-engineering-and-performance-optimization-contract-a64/a64-governance-index.md"
if (-not (Test-Path -LiteralPath $governanceIndexPath)) {
    $archived = Get-ChildItem -Path "openspec/changes/archive" -Directory -Filter "*introduce-engineering-and-performance-optimization-contract-a64" -ErrorAction SilentlyContinue |
        Sort-Object Name -Descending |
        Select-Object -First 1
    if ($archived) {
        $governanceIndexPath = Join-Path "openspec/changes/archive" "$($archived.Name)/a64-governance-index.md"
    }
}
$docChecks = @(
    @{
        Path    = "docs/development-roadmap.md"
        Markers = @("harnessability scorecard", "harness ROI/depth", "computational-first, inferential-second", "门禁耗时预算治理")
    },
    @{
        Path    = "docs/mainline-contract-test-index.md"
        Markers = @("check-a64-harnessability-scorecard.sh", "check-a64-harnessability-scorecard.ps1", "a64-harnessability-scorecard-baseline.env", "a64-gate-latency-baseline.env")
    },
    @{
        Path    = $governanceIndexPath
        Markers = @("Harnessability Scorecard", "门禁耗时基线")
    }
)
$docsCheckDetails = New-Object 'System.Collections.Generic.List[object]'
foreach ($entry in $docChecks) {
    $path = [string]$entry.Path
    if (-not (Test-Path -LiteralPath $path)) {
        $docsIssueCount++
        $docsCheckDetails.Add([ordered]@{
                path   = $path
                marker = "<file>"
                found  = $false
            }) | Out-Null
        continue
    }
    $raw = Get-Content -Raw -LiteralPath $path
    foreach ($marker in @($entry.Markers)) {
        $found = $raw.Contains([string]$marker)
        if (-not $found) {
            $docsIssueCount++
        }
        $docsCheckDetails.Add([ordered]@{
                path   = $path
                marker = [string]$marker
                found  = $found
            }) | Out-Null
    }
}
$maxDocsIssueCount = Parse-NonNegativeInt -Name "BAYMAX_A64_HARNESS_SCORECARD_MAX_DOCS_ISSUE_COUNT" -Raw (Get-EnvOrDefault -Name "BAYMAX_A64_HARNESS_SCORECARD_MAX_DOCS_ISSUE_COUNT" -Default "0")
$docsWithin = ($docsIssueCount -le $maxDocsIssueCount)
if (-not $docsWithin) {
    $failures.Add(("docs_consistency_issue_count={0} > max={1}" -f $docsIssueCount, $maxDocsIssueCount)) | Out-Null
}

# ROI + adaptive depth governance.
$tier = Parse-Tier -Raw (Get-EnvOrDefault -Name "BAYMAX_A64_HARNESS_COMPLEXITY_TIER" -Default "standard")
$tierUpper = $tier.ToUpperInvariant()
$baselineToken = Parse-PositiveDouble -Name ("BAYMAX_A64_HARNESS_BASELINE_TOKEN_" + $tierUpper) -Raw (Get-EnvOrDefault -Name ("BAYMAX_A64_HARNESS_BASELINE_TOKEN_" + $tierUpper) -Default "")
$baselineLatencyMs = Parse-PositiveDouble -Name ("BAYMAX_A64_HARNESS_BASELINE_LATENCY_MS_" + $tierUpper) -Raw (Get-EnvOrDefault -Name ("BAYMAX_A64_HARNESS_BASELINE_LATENCY_MS_" + $tierUpper) -Default "")
$baselineQuality = Parse-PositiveDouble -Name ("BAYMAX_A64_HARNESS_BASELINE_QUALITY_" + $tierUpper) -Raw (Get-EnvOrDefault -Name ("BAYMAX_A64_HARNESS_BASELINE_QUALITY_" + $tierUpper) -Default "")

$measuredToken = Parse-PositiveDouble -Name "BAYMAX_A64_HARNESS_MEASURED_TOKEN" -Raw (Get-EnvOrDefault -Name "BAYMAX_A64_HARNESS_MEASURED_TOKEN" -Default "$baselineToken")
$measuredLatencyMs = Parse-PositiveDouble -Name "BAYMAX_A64_HARNESS_MEASURED_LATENCY_MS" -Raw (Get-EnvOrDefault -Name "BAYMAX_A64_HARNESS_MEASURED_LATENCY_MS" -Default "$baselineLatencyMs")
$measuredQuality = Parse-PositiveDouble -Name "BAYMAX_A64_HARNESS_MEASURED_QUALITY_SCORE" -Raw (Get-EnvOrDefault -Name "BAYMAX_A64_HARNESS_MEASURED_QUALITY_SCORE" -Default "$baselineQuality")

$maxTokenOverheadPct = Parse-Percent -Name ("BAYMAX_A64_HARNESS_MAX_TOKEN_OVERHEAD_PCT_" + $tierUpper) -Raw (Get-EnvOrDefault -Name ("BAYMAX_A64_HARNESS_MAX_TOKEN_OVERHEAD_PCT_" + $tierUpper) -Default "")
$maxLatencyOverheadPct = Parse-Percent -Name ("BAYMAX_A64_HARNESS_MAX_LATENCY_OVERHEAD_PCT_" + $tierUpper) -Raw (Get-EnvOrDefault -Name ("BAYMAX_A64_HARNESS_MAX_LATENCY_OVERHEAD_PCT_" + $tierUpper) -Default "")
$minQualityDeltaPct = Parse-SignedPercent -Name ("BAYMAX_A64_HARNESS_MIN_QUALITY_DELTA_PCT_" + $tierUpper) -Raw (Get-EnvOrDefault -Name ("BAYMAX_A64_HARNESS_MIN_QUALITY_DELTA_PCT_" + $tierUpper) -Default "")

$tokenOverheadPct = Round2 -Value (Get-OverheadPct -Measured $measuredToken -Baseline $baselineToken)
$latencyOverheadPct = Round2 -Value (Get-OverheadPct -Measured $measuredLatencyMs -Baseline $baselineLatencyMs)
$qualityDeltaPct = Round2 -Value (Get-OverheadPct -Measured $measuredQuality -Baseline $baselineQuality)

$roiWithin = ($tokenOverheadPct -le $maxTokenOverheadPct -and $latencyOverheadPct -le $maxLatencyOverheadPct -and $qualityDeltaPct -ge $minQualityDeltaPct)
$recommendedTier = $tier
if (-not $roiWithin) {
    switch ($tier) {
        "enhanced" { $recommendedTier = "standard" }
        "standard" { $recommendedTier = "lightweight" }
        default { $recommendedTier = "lightweight" }
    }
    $failures.Add(("roi thresholds breached tier={0} token_overhead={1}% (max={2}%) latency_overhead={3}% (max={4}%) quality_delta={5}% (min={6}%)" -f $tier, $tokenOverheadPct, $maxTokenOverheadPct, $latencyOverheadPct, $maxLatencyOverheadPct, $qualityDeltaPct, $minQualityDeltaPct)) | Out-Null
}

# Computational-first, inferential-second hierarchy + structured evidence.
$computationalSuites = @(
    "scripts/check-a64-impacted-gate-selection.ps1",
    "scripts/check-a64-semantic-stability-contract.ps1",
    "scripts/check-a64-performance-regression.ps1"
)
$computationalPresent = @($computationalSuites | Where-Object { Test-Path -LiteralPath $_ }).Count
$computationalCoveragePct = Round2 -Value (($computationalPresent / [double]$computationalSuites.Count) * 100.0)
$inferentialBlockingRequested = Parse-Bool -Name "BAYMAX_A64_INFERENTIAL_BLOCKING_REQUESTED" -Raw (Get-EnvOrDefault -Name "BAYMAX_A64_INFERENTIAL_BLOCKING_REQUESTED" -Default "false")
$computationalFirstCompliant = ($computationalCoveragePct -ge 100.0 -and -not $inferentialBlockingRequested)
if (-not $computationalFirstCompliant) {
    $failures.Add(("computational-first hierarchy violated: computational_coverage_pct={0}, inferential_blocking_requested={1}" -f $computationalCoveragePct, $inferentialBlockingRequested.ToString().ToLowerInvariant())) | Out-Null
}

$inputSnapshotPath = (Get-EnvOrDefault -Name "BAYMAX_A64_INFERENTIAL_INPUT_SNAPSHOT" -Default "tool/diagnosticsreplay/testdata/a61_inferential_advisory_distributed_success_input.json").Trim()
$promptVersion = (Get-EnvOrDefault -Name "BAYMAX_A64_INFERENTIAL_PROMPT_VERSION" -Default "a64-harnessability-v1").Trim()
$scoringSummary = (Get-EnvOrDefault -Name "BAYMAX_A64_INFERENTIAL_SCORING_SUMMARY" -Default ("tier=" + $tier + "; quality_delta_pct=" + $qualityDeltaPct)).Trim()
$uncertaintyPctRaw = (Get-EnvOrDefault -Name "BAYMAX_A64_INFERENTIAL_UNCERTAINTY_PCT" -Default "15")
$uncertaintyPct = Parse-Percent -Name "BAYMAX_A64_INFERENTIAL_UNCERTAINTY_PCT" -Raw $uncertaintyPctRaw
$maxUncertaintyPct = Parse-Percent -Name "BAYMAX_A64_HARNESS_SCORECARD_MAX_INFERENTIAL_UNCERTAINTY_PCT" -Raw (Get-EnvOrDefault -Name "BAYMAX_A64_HARNESS_SCORECARD_MAX_INFERENTIAL_UNCERTAINTY_PCT" -Default "35")
$uncertaintyWithin = ($uncertaintyPct -le $maxUncertaintyPct)
$evidenceComplete = (-not [string]::IsNullOrWhiteSpace($inputSnapshotPath) -and (Test-Path -LiteralPath $inputSnapshotPath) -and -not [string]::IsNullOrWhiteSpace($promptVersion) -and -not [string]::IsNullOrWhiteSpace($scoringSummary))
if (-not $evidenceComplete) {
    $failures.Add("inferential evidence incomplete: input_snapshot/prompt_version/scoring_summary must be provided and snapshot file must exist") | Out-Null
}
if ($inferentialBlockingRequested -and -not $uncertaintyWithin) {
    $failures.Add(("inferential uncertainty {0}% exceeds max {1}% and cannot be used as blocking signal" -f $uncertaintyPct, $maxUncertaintyPct)) | Out-Null
}

$report = [ordered]@{
    generated_at     = (Get-Date).ToString("o")
    complexity_tier  = $tier
    metrics          = [ordered]@{
        contract_coverage_pct = $contractCoveragePct
        drift = [ordered]@{
            fixture_count                 = $driftFixtureCount
            min_fixture_count             = $minDriftFixtureCount
            unclassified_count            = $unclassifiedDriftCount
            max_unclassified_count        = $maxUnclassifiedDriftCount
            within_threshold              = $driftWithin
        }
        gate_coverage_pct = $gateCoveragePct
        docs_consistency = [ordered]@{
            issue_count       = $docsIssueCount
            max_issue_count   = $maxDocsIssueCount
            within_threshold  = $docsWithin
        }
        roi = [ordered]@{
            baseline = [ordered]@{
                token         = $baselineToken
                latency_ms    = $baselineLatencyMs
                quality_score = $baselineQuality
            }
            measured = [ordered]@{
                token         = $measuredToken
                latency_ms    = $measuredLatencyMs
                quality_score = $measuredQuality
            }
            overhead_pct = [ordered]@{
                token   = $tokenOverheadPct
                latency = $latencyOverheadPct
                quality = $qualityDeltaPct
            }
            thresholds = [ordered]@{
                max_token_overhead_pct   = $maxTokenOverheadPct
                max_latency_overhead_pct = $maxLatencyOverheadPct
                min_quality_delta_pct    = $minQualityDeltaPct
            }
            within_threshold        = $roiWithin
            downgrade_recommendation = $recommendedTier
        }
    }
    hierarchy        = [ordered]@{
        objective_domains             = @("contract", "replay", "schema", "taxonomy")
        computational_suites          = $computationalSuites
        computational_coverage_pct    = $computationalCoveragePct
        inferential_blocking_requested = $inferentialBlockingRequested
        computational_first_compliant = $computationalFirstCompliant
    }
    inferential_evidence = [ordered]@{
        input_snapshot              = $inputSnapshotPath
        prompt_version              = $promptVersion
        scoring_summary             = $scoringSummary
        uncertainty_pct             = $uncertaintyPct
        max_uncertainty_pct         = $maxUncertaintyPct
        uncertainty_within_threshold = $uncertaintyWithin
        evidence_complete           = $evidenceComplete
    }
    gate_coverage_details = $gateCoverageDetails
    docs_check_details    = $docsCheckDetails
    score = [ordered]@{
        pass          = ($failures.Count -eq 0)
        failed_checks = $failures
    }
}

$json = $report | ConvertTo-Json -Depth 8
Write-Host "[a64-harnessability-scorecard] report:"
Write-Host $json

$outputPath = (Get-EnvOrDefault -Name "BAYMAX_A64_HARNESS_SCORECARD_REPORT_PATH" -Default ".artifacts/a64/harnessability-scorecard.json").Trim()
if ($outputPath -ne "") {
    $parent = Split-Path -Parent $outputPath
    if ($parent -and -not (Test-Path -LiteralPath $parent)) {
        New-Item -ItemType Directory -Path $parent -Force | Out-Null
    }
    Set-Content -LiteralPath $outputPath -Value $json -NoNewline
    Write-Host "[a64-harnessability-scorecard] report written to $outputPath"
}

if ($failures.Count -gt 0) {
    throw ("[a64-harnessability-scorecard] failed: " + ($failures -join " | "))
}

Write-Host "[a64-harnessability-scorecard] passed"
