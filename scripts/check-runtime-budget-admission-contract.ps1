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
        throw "[runtime-budget-admission-gate][$Assertion] missing file: $FilePath"
    }
    $content = Get-Content -LiteralPath $fullPath -Raw
    if (-not $content.Contains($Literal)) {
        throw "[runtime-budget-admission-gate][$Assertion] missing marker '$Literal' in $FilePath"
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
        throw "[runtime-budget-admission-gate][$Assertion] unexpected matches found for /$Pattern/:`n$preview"
    }
}

function Assert-NoParallelBudgetAdmissionChanges {
    param(
        [Parameter(Mandatory = $true)][string]$Assertion
    )

    $changeRoot = Join-Path $repoRoot "openspec/changes"
    $canonical = "introduce-runtime-cost-latency-budget-and-admission-contract-a60"
    $violations = @()
    $dirs = Get-ChildItem -Path $changeRoot -Directory | Where-Object { $_.Name -ne "archive" }
    foreach ($dir in $dirs) {
        $lower = $dir.Name.ToLowerInvariant()
        if ($dir.Name -ne $canonical -and $lower.Contains("budget") -and $lower.Contains("admission")) {
            $violations += $dir.Name
        }
    }
    if ($violations.Count -gt 0) {
        throw "[runtime-budget-admission-gate][$Assertion] parallel budget-admission proposal detected: $($violations -join ', ')"
    }
}

function Invoke-BudgetAdmissionStep {
    param(
        [Parameter(Mandatory = $true)][string]$Label,
        [Parameter(Mandatory = $true)][scriptblock]$Command
    )
    Write-Host "[runtime-budget-admission-gate] $Label"
    & $Command
}

Invoke-BudgetAdmissionStep -Label "assertion budget_control_plane_absent: contract markers + no parallel control-plane config key" -Command {
    Assert-ContainsLiteral -Assertion "budget_control_plane_absent" -FilePath "openspec/changes/introduce-runtime-cost-latency-budget-and-admission-contract-a60/specs/runtime-cost-latency-budget-and-admission-contract/spec.md" -Literal "MUST NOT require hosted control-plane services"
}

Invoke-BudgetAdmissionStep -Label "assertion budget_control_plane_absent: gate spec marker" -Command {
    Assert-ContainsLiteral -Assertion "budget_control_plane_absent" -FilePath "openspec/changes/introduce-runtime-cost-latency-budget-and-admission-contract-a60/specs/go-quality-gate/spec.md" -Literal "budget_control_plane_absent"
}

Invoke-BudgetAdmissionStep -Label "assertion budget_control_plane_absent: active change set closure" -Command {
    Assert-NoParallelBudgetAdmissionChanges -Assertion "budget_control_plane_absent"
}

Invoke-BudgetAdmissionStep -Label "assertion budget_control_plane_absent: reject runtime admission control-plane key drift" -Command {
    Assert-PatternAbsentAcrossRepo -Assertion "budget_control_plane_absent" -Pattern "runtime\.admission\.[a-zA-Z0-9_.-]*(control_plane|controlplane|admission_service|policy_center)"
}

Invoke-BudgetAdmissionStep -Label "assertion budget_field_reuse_required: canonical field reuse marker" -Command {
    Assert-ContainsLiteral -Assertion "budget_field_reuse_required" -FilePath "openspec/changes/introduce-runtime-cost-latency-budget-and-admission-contract-a60/specs/runtime-cost-latency-budget-and-admission-contract/spec.md" -Literal "policy_decision_path"
}

Invoke-BudgetAdmissionStep -Label "assertion budget_field_reuse_required: gate spec marker" -Command {
    Assert-ContainsLiteral -Assertion "budget_field_reuse_required" -FilePath "openspec/changes/introduce-runtime-cost-latency-budget-and-admission-contract-a60/specs/go-quality-gate/spec.md" -Literal "budget_field_reuse_required"
}

Invoke-BudgetAdmissionStep -Label "assertion budget_field_reuse_required: roadmap closure marker" -Command {
    Assert-ContainsLiteral -Assertion "budget_field_reuse_required" -FilePath "docs/development-roadmap.md" -Literal "A60 预算 admission 同域增量需求（阈值、维度、降级动作、回放、门禁）仅允许在 A60 内以增量任务吸收，不再新开平行提案。"
}

Invoke-BudgetAdmissionStep -Label "assertion budget_field_reuse_required: reject duplicated upstream field aliases" -Command {
    Assert-PatternAbsentAcrossRepo -Assertion "budget_field_reuse_required" -Pattern "runtime\.admission\.[a-zA-Z0-9_.-]*(policy_decision_path|deny_source|winner_stage|memory_scope_selected|memory_budget_used|memory_hits|memory_rerank_stats|memory_lifecycle_action)"
}

Write-Host "[runtime-budget-admission-gate] contributioncheck parity suites for runtime budget admission gate"
Invoke-NativeStrict -Label "go test ./tool/contributioncheck -run 'Test(RuntimeBudgetAdmissionGateScriptParity|QualityGateIncludesRuntimeBudgetAdmissionGate|RuntimeBudgetAdmissionRoadmapAndContractIndexClosureMarkers)' -count=1" -Command {
    go test ./tool/contributioncheck -run 'Test(RuntimeBudgetAdmissionGateScriptParity|QualityGateIncludesRuntimeBudgetAdmissionGate|RuntimeBudgetAdmissionRoadmapAndContractIndexClosureMarkers)' -count=1
}

Write-Host "[runtime-budget-admission-gate] done"
