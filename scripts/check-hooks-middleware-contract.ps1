Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}
if ($env:GODEBUG) {
    if ($env:GODEBUG -notmatch "(^|,)goindex=") {
        $env:GODEBUG = "$($env:GODEBUG),goindex=0"
    }
}
else {
    $env:GODEBUG = "goindex=0"
}

function Assert-ContainsLiteral {
    param(
        [Parameter(Mandatory = $true)][string]$Assertion,
        [Parameter(Mandatory = $true)][string]$FilePath,
        [Parameter(Mandatory = $true)][string]$Literal
    )
    $fullPath = Join-Path $repoRoot $FilePath
    if (-not (Test-Path -LiteralPath $fullPath)) {
        throw "[hooks-middleware-contract-gate][$Assertion] missing file: $FilePath"
    }
    $content = Get-Content -LiteralPath $fullPath -Raw
    if (-not $content.Contains($Literal)) {
        throw "[hooks-middleware-contract-gate][$Assertion] missing marker '$Literal' in $FilePath"
    }
}

function Assert-PatternAbsentAcrossRepo {
    param(
        [Parameter(Mandatory = $true)][string]$Assertion,
        [Parameter(Mandatory = $true)][string]$Pattern
    )

    $rgPath = Get-Command rg -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty Source
    if (-not [string]::IsNullOrWhiteSpace($rgPath)) {
        $scanOutput = @(& $rgPath -n --glob '!openspec/changes/archive/**' -- $Pattern $repoRoot.Path 2>&1)
        $scanExit = $LASTEXITCODE
        if ($scanExit -eq 0) {
            $preview = ($scanOutput | Where-Object { $null -ne $_ } | Select-Object -First 10 | ForEach-Object {
                    $_.ToString().Trim()
                }) -join "`n"
            throw "[hooks-middleware-contract-gate][$Assertion] unexpected matches found for /$Pattern/:`n$preview"
        }
        if ($scanExit -eq 1) {
            return
        }
        $details = ($scanOutput | Where-Object { $null -ne $_ } | Select-Object -First 10 | ForEach-Object {
                $_.ToString().Trim()
            }) -join "`n"
        throw "[hooks-middleware-contract-gate][$Assertion] rg scan failed for /$Pattern/ (exit=$scanExit):`n$details"
    }

    $archiveRoot = [Regex]::Escape((Join-Path $repoRoot "openspec\changes\archive"))
    $files = Get-ChildItem -Path $repoRoot -Recurse -File | Where-Object {
        $_.FullName -notmatch $archiveRoot
    }

    $matches = @()
    foreach ($file in $files) {
        $hit = Select-String -Path $file.FullName -Pattern $Pattern -ErrorAction SilentlyContinue
        if ($hit) {
            $matches += $hit
            if ($matches.Count -ge 10) {
                break
            }
        }
    }

    if ($matches.Count -gt 0) {
        $preview = ($matches | Select-Object -First 10 | ForEach-Object {
                "$($_.Path):$($_.LineNumber): $($_.Line.Trim())"
            }) -join "`n"
        throw "[hooks-middleware-contract-gate][$Assertion] unexpected matches found for /$Pattern/:`n$preview"
    }
}

function Assert-NoParallelHooksMiddlewareChanges {
    param(
        [Parameter(Mandatory = $true)][string]$Assertion
    )

    $changeRoot = Join-Path $repoRoot "openspec/changes"
    $canonicalHint = "introduce-agent-lifecycle-hooks-and-tool-middleware-contract"
    $violations = @()
    $dirs = Get-ChildItem -Path $changeRoot -Directory | Where-Object { $_.Name -ne "archive" }
    foreach ($dir in $dirs) {
        $lower = $dir.Name.ToLowerInvariant()
        if (-not $lower.Contains($canonicalHint) -and $lower.Contains("hook") -and $lower.Contains("middleware")) {
            $violations += $dir.Name
        }
    }
    if ($violations.Count -gt 0) {
        throw "[hooks-middleware-contract-gate][$Assertion] parallel hooks/middleware proposal detected: $($violations -join ', ')"
    }
}

function Resolve-HooksMiddlewareChangeDir {
    $activeRoot = Join-Path $repoRoot "openspec/changes"
    if (Test-Path -LiteralPath $activeRoot -PathType Container) {
        $candidate = Get-ChildItem -Path $activeRoot -Directory |
            Where-Object { $_.Name -ne "archive" -and $_.Name -like "*introduce-agent-lifecycle-hooks-and-tool-middleware-contract*" } |
            Select-Object -First 1
        if ($candidate) {
            return "openspec/changes/$($candidate.Name)"
        }
    }

    $archiveRoot = Join-Path $repoRoot "openspec/changes/archive"
    if (Test-Path -LiteralPath $archiveRoot -PathType Container) {
        $candidate = Get-ChildItem -Path $archiveRoot -Directory |
            Where-Object { $_.Name -like "*introduce-agent-lifecycle-hooks-and-tool-middleware-contract*" } |
            Select-Object -First 1
        if ($candidate) {
            return "openspec/changes/archive/$($candidate.Name)"
        }
    }

    throw "[hooks-middleware-contract-gate] unable to locate hooks/middleware change directory in active or archive paths"
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

function Invoke-HooksMiddlewareStep {
    param(
        [Parameter(Mandatory = $true)][string]$Label,
        [Parameter(Mandatory = $true)][scriptblock]$Command
    )
    Write-Host "[hooks-middleware-contract-gate] $Label"
    [void](Invoke-NativeStrict -Label $Label -Command $Command)
}

$hooksMiddlewareChangeDir = Resolve-HooksMiddlewareChangeDir

Write-Host "[hooks-middleware-contract-gate] assertion control_plane_absent: contract marker"
Assert-ContainsLiteral -Assertion "control_plane_absent" -FilePath "$hooksMiddlewareChangeDir/specs/agent-lifecycle-hooks-and-tool-middleware-contract/spec.md" -Literal "MUST NOT require hosted control-plane services"

Write-Host "[hooks-middleware-contract-gate] assertion control_plane_absent: gate spec marker"
Assert-ContainsLiteral -Assertion "control_plane_absent" -FilePath "$hooksMiddlewareChangeDir/specs/go-quality-gate/spec.md" -Literal "control_plane_absent"

Write-Host "[hooks-middleware-contract-gate] assertion control_plane_absent: active change set closure"
Assert-NoParallelHooksMiddlewareChanges -Assertion "control_plane_absent"

Write-Host "[hooks-middleware-contract-gate] assertion control_plane_absent: reject hooks/middleware control-plane key drift"
Assert-PatternAbsentAcrossRepo -Assertion "control_plane_absent" -Pattern "runtime\.(hooks|tool_middleware)\.[a-zA-Z0-9_.-]*(control_plane|controlplane|orchestrator|controller|service_endpoint|remote_hook|hosted_hook|managed_middleware)"

Write-Host "[hooks-middleware-contract-gate] assertion hooks_middleware_same_domain_closure: roadmap marker"
Assert-ContainsLiteral -Assertion "hooks_middleware_same_domain_closure" -FilePath "docs/development-roadmap.md" -Literal "Hooks/middleware 同域增量需求（lifecycle、middleware、discovery、preprocess、mapping、回放、门禁）仅允许在本提案内以增量任务吸收，不再新开平行提案。"

Invoke-HooksMiddlewareStep -Label "hooks/middleware run-stream parity suites" -Command {
    go test ./core/runner -run 'Test(LifecycleHooksRunAndStreamPhaseOrderParity|LifecycleHooksFailFastStopsRunAndStream|LifecycleHooksDegradeContinuesRunAndStream|ToolMiddlewareTimeoutClassifiedAsPolicyTimeoutInRunAndStream|SkillPreprocessRunsBeforeRunAndStreamModelLoop|SkillPreprocessFailFastAbortsRunAndStream|SkillPreprocessDegradeContinuesRunAndStream|SkillBundlePromptMappingAppendDeterministicForRunAndStream|SkillBundlePromptMappingConflictFailFastForRunAndStream|SkillBundleWhitelistFailFastRejectsBlockedToolForRunAndStream|SkillBundleWhitelistUpperBoundSandboxRejectsDuringPreprocess|SkillBundleWhitelistFirstWinFiltersBlockedToolForRunAndStream)' -count=1
}

Invoke-HooksMiddlewareStep -Label "hooks/middleware diagnostics additive compatibility suites" -Command {
    go test ./runtime/diagnostics ./observability/event -run 'Test(StoreRunHooksMiddlewareAdditiveFieldsPersistAndReplayIdempotent|RuntimeRecorderParsesHooksMiddlewareSkillAdditiveFields|RuntimeRecorderHooksMiddlewareParserCompatibilityAdditiveNullableDefault|RuntimeRecorderHooksMiddlewareReasonTaxonomyDriftGuardCanonicalFallback)' -count=1
}

Invoke-HooksMiddlewareStep -Label "hooks/middleware replay fixture + drift suites" -Command {
    go test ./tool/diagnosticsreplay ./integration -run 'TestReplayContractHooksMiddleware(FixtureSuite|DriftClassification|DriftGuardFailFast)' -count=1
}

$changedFiles = @(Get-ChangedFiles)
$runnerImpacted = $false
$skillImpacted = $false
$observabilityImpacted = $false
if ($changedFiles.Count -eq 0) {
    $runnerImpacted = $true
    $skillImpacted = $true
    $observabilityImpacted = $true
}
else {
    if ((Test-ChangedPrefix -Prefix "core/runner/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "tool/local/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "core/types/" -Files $changedFiles)) {
        $runnerImpacted = $true
    }
    if ((Test-ChangedPrefix -Prefix "skill/loader/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "core/runner/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "runtime/config/runtime_hooks_middleware" -Files $changedFiles)) {
        $skillImpacted = $true
    }
    if ((Test-ChangedPrefix -Prefix "runtime/diagnostics/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "observability/event/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "tool/diagnosticsreplay/" -Files $changedFiles)) {
        $observabilityImpacted = $true
    }
}

if ($runnerImpacted) {
    Invoke-HooksMiddlewareStep -Label "impacted-contract suites (runner scope): security policy gate" -Command {
        pwsh -File scripts/check-security-policy-contract.ps1
    }
    Invoke-HooksMiddlewareStep -Label "impacted-contract suites (runner scope): security event gate" -Command {
        pwsh -File scripts/check-security-event-contract.ps1
    }
}

if ($skillImpacted) {
    Invoke-HooksMiddlewareStep -Label "impacted-contract suites (skill scope): skill loader + runtime skill config suites" -Command {
        go test ./skill/loader ./runtime/config -run 'Test(Compile|RuntimeHooksToolMiddlewareSkillConfig|ManagerRuntimeHooksAndSkillInvalidReloadRollsBack)' -count=1
    }
    Invoke-HooksMiddlewareStep -Label "impacted-contract suites (skill scope): replay/contract compatibility suites" -Command {
        go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractHooksMiddleware|ReplayContractPrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|PrimaryReasonArbitrationReplayContractFixtureSuite)' -count=1
    }
}

if ($observabilityImpacted) {
    Invoke-HooksMiddlewareStep -Label "impacted-contract suites (observability scope): observability export+bundle gate" -Command {
        pwsh -File scripts/check-observability-export-and-bundle-contract.ps1
    }
    Invoke-HooksMiddlewareStep -Label "impacted-contract suites (observability scope): diagnostics replay compatibility suites" -Command {
        go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractHooksMiddleware|ReplayContractTracingEval|ReplayContractPrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput)' -count=1
    }
}

Invoke-HooksMiddlewareStep -Label "contributioncheck parity suites for hooks/middleware gate" -Command {
    go test ./tool/contributioncheck -run 'Test(HooksMiddlewareGateScriptParity|QualityGateIncludesHooksMiddlewareGate|HooksMiddlewareRoadmapAndContractIndexClosureMarkers)' -count=1
}

Write-Host "[hooks-middleware-contract-gate] done"
