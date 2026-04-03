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
        throw "[state-snapshot-contract-gate][$Assertion] missing file: $FilePath"
    }
    $content = Get-Content -LiteralPath $fullPath -Raw
    if (-not $content.Contains($Literal)) {
        throw "[state-snapshot-contract-gate][$Assertion] missing marker '$Literal' in $FilePath"
    }
}

function Assert-PatternAbsentAcrossRepo {
    param(
        [Parameter(Mandatory = $true)][string]$Assertion,
        [Parameter(Mandatory = $true)][string]$Pattern
    )

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
        throw "[state-snapshot-contract-gate][$Assertion] unexpected matches found for /$Pattern/:`n$preview"
    }
}

function Assert-NoParallelA66SnapshotChanges {
    param(
        [Parameter(Mandatory = $true)][string]$Assertion
    )

    $changeRoot = Join-Path $repoRoot "openspec/changes"
    $canonical = "introduce-unified-state-and-session-snapshot-contract-a66"
    $violations = @()
    $dirs = Get-ChildItem -Path $changeRoot -Directory | Where-Object { $_.Name -ne "archive" }
    foreach ($dir in $dirs) {
        $lower = $dir.Name.ToLowerInvariant()
        if ($dir.Name -ne $canonical -and $lower.Contains("snapshot") -and ($lower.Contains("state") -or $lower.Contains("session"))) {
            $violations += $dir.Name
        }
    }
    if ($violations.Count -gt 0) {
        throw "[state-snapshot-contract-gate][$Assertion] parallel state/session snapshot proposal detected: $($violations -join ', ')"
    }
}

function Resolve-A66ChangeDir {
    $active = "openspec/changes/introduce-unified-state-and-session-snapshot-contract-a66"
    $activeFull = Join-Path $repoRoot $active
    if (Test-Path -LiteralPath $activeFull -PathType Container) {
        return $active
    }

    $archiveRoot = Join-Path $repoRoot "openspec/changes/archive"
    if (Test-Path -LiteralPath $archiveRoot -PathType Container) {
        $candidate = Get-ChildItem -Path $archiveRoot -Directory |
            Where-Object { $_.Name -like "*introduce-unified-state-and-session-snapshot-contract-a66" } |
            Select-Object -First 1
        if ($candidate) {
            return "openspec/changes/archive/$($candidate.Name)"
        }
    }

    throw "[state-snapshot-contract-gate] unable to locate A66 change directory in active or archive paths"
}

function Invoke-StateSnapshotStep {
    param(
        [Parameter(Mandatory = $true)][string]$Label,
        [Parameter(Mandatory = $true)][scriptblock]$Command
    )
    Write-Host "[state-snapshot-contract-gate] $Label"
    & $Command
}

$a66ChangeDir = Resolve-A66ChangeDir

Invoke-StateSnapshotStep -Label "assertion state_control_plane_absent: design marker" -Command {
    Assert-ContainsLiteral -Assertion "state_control_plane_absent" -FilePath "$a66ChangeDir/design.md" -Literal "不引入托管状态控制面、远程恢复调度服务或平台化迁移中心。"
}

Invoke-StateSnapshotStep -Label "assertion state_control_plane_absent: gate spec marker" -Command {
    Assert-ContainsLiteral -Assertion "state_control_plane_absent" -FilePath "$a66ChangeDir/specs/go-quality-gate/spec.md" -Literal "check-state-snapshot-contract.sh/.ps1"
}

Invoke-StateSnapshotStep -Label "assertion state_control_plane_absent: active change set closure" -Command {
    Assert-NoParallelA66SnapshotChanges -Assertion "state_control_plane_absent"
}

Invoke-StateSnapshotStep -Label "assertion state_control_plane_absent: reject hosted control-plane config drift" -Command {
    Assert-PatternAbsentAcrossRepo -Assertion "state_control_plane_absent" -Pattern "runtime\.(state\.snapshot|session\.state)\.[a-zA-Z0-9_.-]*(control_plane|controlplane|state_service|orchestrator|controller|managed_state|hosted_state|remote_state|migration_center)"
}

Invoke-StateSnapshotStep -Label "assertion state_source_of_truth_reuse_required: canonical source-of-truth marker" -Command {
    Assert-ContainsLiteral -Assertion "state_source_of_truth_reuse_required" -FilePath "$a66ChangeDir/specs/memory-scope-and-builtin-filesystem-v2-governance-contract/spec.md" -Literal "MUST NOT redefine memory source-of-truth behavior."
}

Invoke-StateSnapshotStep -Label "assertion state_source_of_truth_reuse_required: roadmap closure marker" -Command {
    Assert-ContainsLiteral -Assertion "state_source_of_truth_reuse_required" -FilePath "docs/development-roadmap.md" -Literal "A66 必须复用现有 checkpoint/snapshot 语义与 A59 memory lifecycle，不得重写存储层事实源。"
}

Invoke-StateSnapshotStep -Label "assertion state_source_of_truth_reuse_required: reject duplicated memory source aliases in snapshot config" -Command {
    Assert-PatternAbsentAcrossRepo -Assertion "state_source_of_truth_reuse_required" -Pattern "runtime\.state\.snapshot\.[a-zA-Z0-9_.-]*(memory_mode|memory_provider|memory_profile|memory_contract_version|memory_scope_selected|memory_budget_used|memory_hits|memory_rerank_stats|memory_lifecycle_action)"
}

Write-Host "[state-snapshot-contract-gate] snapshot config governance suites"
Invoke-NativeStrict -Label "go test ./runtime/config -run 'Test(RuntimeStateSnapshotSessionConfig|ManagerRuntimeStateSnapshotInvalidReloadRollsBack)' -count=1" -Command {
    go test ./runtime/config -run 'Test(RuntimeStateSnapshotSessionConfig|ManagerRuntimeStateSnapshotInvalidReloadRollsBack)' -count=1
}

Write-Host "[state-snapshot-contract-gate] unified snapshot manifest contract suites"
Invoke-NativeStrict -Label "go test ./orchestration/snapshot -run '^Test(ExportImportRoundTripStable|ImportIdempotencyNoInflation|ImportStrictRejectsIncompatibleVersion|ImportCompatibleWithinWindow|ImportSameOperationDifferentDigestConflict)$' -count=1" -Command {
    go test ./orchestration/snapshot -run '^Test(ExportImportRoundTripStable|ImportIdempotencyNoInflation|ImportStrictRejectsIncompatibleVersion|ImportCompatibleWithinWindow|ImportSameOperationDifferentDigestConflict)$' -count=1
}

Write-Host "[state-snapshot-contract-gate] composer unified snapshot runtime suites"
Invoke-NativeStrict -Label "go test ./orchestration/composer -run '^TestComposerUnifiedSnapshot' -count=1" -Command {
    go test ./orchestration/composer -run '^TestComposerUnifiedSnapshot' -count=1
}

Write-Host "[state-snapshot-contract-gate] a66 restore integration suites"
Invoke-NativeStrict -Label "go test ./integration -run '^TestA66UnifiedSnapshot' -count=1" -Command {
    go test ./integration -run '^TestA66UnifiedSnapshot' -count=1
}

Write-Host "[state-snapshot-contract-gate] shared recovery suites for impacted scope"
Invoke-NativeStrict -Label "go test ./integration -run '^Test(SchedulerRecovery|ComposerRecovery)' -count=1" -Command {
    go test ./integration -run '^Test(SchedulerRecovery|ComposerRecovery)' -count=1
}

Write-Host "[state-snapshot-contract-gate] diagnostics replay suites for impacted scope"
Invoke-NativeStrict -Label "go test ./tool/diagnosticsreplay -run '^TestReplayContractPrimaryReasonArbitrationFixture(SuccessAndDeterministicOutput|DriftClassification)$' -count=1" -Command {
    go test ./tool/diagnosticsreplay -run '^TestReplayContractPrimaryReasonArbitrationFixture(SuccessAndDeterministicOutput|DriftClassification)$' -count=1
}

Write-Host "[state-snapshot-contract-gate] done"
