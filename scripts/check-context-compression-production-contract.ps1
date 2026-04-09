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
        throw "[context-compression-production-contract-gate] unable to prepare writable cache directory for $EnvName at $FallbackPath"
    }
    Set-Item -Path ("Env:" + $EnvName) -Value $FallbackPath
}

Ensure-WritableCacheEnv -EnvName "GOCACHE" -FallbackPath (Join-Path $repoRoot ".gocache")

if ($env:GODEBUG) {
    if ($env:GODEBUG -notmatch "(^|,)goindex=") {
        $env:GODEBUG = "$($env:GODEBUG),goindex=0"
    }
}
else {
    $env:GODEBUG = "goindex=0"
}

function Invoke-ContextCompressionStep {
    param(
        [Parameter(Mandatory = $true)][string]$Label,
        [Parameter(Mandatory = $true)][scriptblock]$Command
    )
    Write-Host "[context-compression-production-contract-gate] $Label"
    [void](Invoke-NativeStrict -Label $Label -Command $Command)
}

function Get-ChangedFiles {
    git rev-parse --verify origin/main *> $null
    if ($LASTEXITCODE -eq 0) {
        $mergeBase = (git merge-base HEAD origin/main 2>$null | Select-Object -First 1).Trim()
        if (-not [string]::IsNullOrWhiteSpace($mergeBase)) {
            return @(git diff --name-only --diff-filter=ACMRTUXB "$mergeBase..HEAD" | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
        }
    }
    git rev-parse --verify HEAD~1 *> $null
    if ($LASTEXITCODE -eq 0) {
        return @(git diff --name-only --diff-filter=ACMRTUXB HEAD~1..HEAD | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    }
    return @()
}

function Test-ChangedPrefix {
    param(
        [Parameter(Mandatory = $true)][string]$Prefix,
        [Parameter(Mandatory = $true)][string[]]$Files
    )
    foreach ($item in $Files) {
        if ($item.StartsWith($Prefix, [System.StringComparison]::OrdinalIgnoreCase)) {
            return $true
        }
    }
    return $false
}

function Test-TruthyEnv {
    param(
        [Parameter(Mandatory = $false)][string]$Value
    )
    if ([string]::IsNullOrWhiteSpace($Value)) {
        return $false
    }
    switch ($Value.Trim().ToLowerInvariant()) {
        "1" { return $true }
        "true" { return $true }
        "yes" { return $true }
        "on" { return $true }
        default { return $false }
    }
}

Invoke-ContextCompressionStep -Label "context compression runtime config governance suites" -Command {
    go test ./runtime/config -run 'Test(ContextAssemblerContextPressure|RuntimeContextJITConfig|ManagerRuntimeContextJITInvalidReloadRollsBack)' -count=1
}

Invoke-ContextCompressionStep -Label "context compression context assembler suites" -Command {
    go test ./context/assembler -run 'Test(AssemblerContextPressure(SemanticCompactionUsesModelClient|SemanticCompactionBestEffortFallback|SemanticCompactionFailFast|SemanticCompactionQualityGateBestEffortFallback|PruneRetainsEvidenceAndReportsCount|SpillIdempotentAcrossRetry|SwapBackAndTieringCombination)|SwapBackIfNeededUsesRelevanceThreshold|ApplyLifecycleTieringTransitionsAndPrune)' -count=1
}

Invoke-ContextCompressionStep -Label "context compression run/stream parity suites" -Command {
    go test ./core/runner -run 'Test(RunAndStreamContextPressure(SemanticsEquivalent|GovernanceSemanticsEquivalent)|ContextJITRunAndStreamSemanticEquivalent)' -count=1
}

Invoke-ContextCompressionStep -Label "context compression diagnostics + recorder additive suites" -Command {
    go test ./runtime/diagnostics ./observability/event -run 'Test(StoreRunContextJIT(AdditiveFieldsPersistAndReplayIdempotent|QueryRunsParserCompatibilityAdditiveNullableDefault)|RuntimeRecorder(AcceptsSemanticContextPressurePayload|ParsesContextJITOrganizationAdditiveFields|ContextJITParserCompatibilityAdditiveNullableDefault|RecoveryParserCompatibilityAdditiveNullableDefault))' -count=1
}

Invoke-ContextCompressionStep -Label "context compression replay fixture + drift taxonomy suites" -Command {
    go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractContextCompressionProductionFixtureSuite|ReplayContractContextCompressionProductionDriftClassification|ReplayContractContextCompressionProductionMixedFixtureBackwardCompatibility|ReplayContractPrimaryReasonArbitrationA69ContextCompressionFixtureSuite|PrimaryReasonArbitrationReplayContractA69ContextCompressionDriftGuard|ReplayContractA69ContextCompressionMixedFixtureBackwardCompatibility)' -count=1
}

$changedFiles = @(Get-ChangedFiles)
$contextImpacted = $false
$replayImpacted = $false
$benchmarkImpacted = $false
if ($changedFiles.Count -eq 0) {
    $contextImpacted = $true
    $replayImpacted = $true
    $benchmarkImpacted = $true
}
else {
    if ((Test-ChangedPrefix -Prefix "context/assembler/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "core/runner/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "runtime/config/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "runtime/diagnostics/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "observability/event/" -Files $changedFiles)) {
        $contextImpacted = $true
    }
    if ((Test-ChangedPrefix -Prefix "tool/diagnosticsreplay/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "integration/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "runtime/diagnostics/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "observability/event/" -Files $changedFiles)) {
        $replayImpacted = $true
    }
    if ((Test-ChangedPrefix -Prefix "context/assembler/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "core/runner/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "integration/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "scripts/check-context-production-hardening-benchmark-regression" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "scripts/context-production-hardening-benchmark-baseline.env" -Files $changedFiles)) {
        $benchmarkImpacted = $true
    }
}

$skipImpactedSuites = Test-TruthyEnv -Value $env:BAYMAX_CONTEXT_COMPRESSION_SKIP_IMPACTED_CONTRACT_SUITES
Write-Host "[context-compression-production-contract-gate] impacted-evaluation context=$contextImpacted replay=$replayImpacted benchmark=$benchmarkImpacted skip_impacted=$skipImpactedSuites"

if (-not $skipImpactedSuites) {
    if ($contextImpacted) {
        Invoke-ContextCompressionStep -Label "impacted-contract suites (context scope): context jit organization gate" -Command {
            $env:BAYMAX_CONTEXT_JIT_SKIP_IMPACTED_CONTRACT_SUITES = "1"
            pwsh -File scripts/check-context-jit-organization-contract.ps1
        }
    }
    if ($replayImpacted) {
        Invoke-ContextCompressionStep -Label "impacted-contract suites (replay scope): diagnostics replay contract gate" -Command {
            pwsh -File scripts/check-diagnostics-replay-contract.ps1
        }
    }
    if ($benchmarkImpacted) {
        Invoke-ContextCompressionStep -Label "impacted-contract suites (benchmark scope): context production hardening benchmark regression gate" -Command {
            pwsh -File scripts/check-context-production-hardening-benchmark-regression.ps1
        }
    }
}
else {
    Write-Host "[context-compression-production-contract-gate] skip impacted-contract suites (BAYMAX_CONTEXT_COMPRESSION_SKIP_IMPACTED_CONTRACT_SUITES=$($env:BAYMAX_CONTEXT_COMPRESSION_SKIP_IMPACTED_CONTRACT_SUITES))"
}

Invoke-ContextCompressionStep -Label "contributioncheck parity suites for context-compression-production gate" -Command {
    go test ./tool/contributioncheck -run 'Test(ContextCompressionProductionGateScriptParity|QualityGateIncludesContextCompressionProductionGate|CIIncludesContextCompressionProductionRequiredCheckCandidate|ContextCompressionProductionRoadmapAndContractIndexClosureMarkers)' -count=1
}

Write-Host "[context-compression-production-contract-gate] done"
