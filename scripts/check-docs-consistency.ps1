Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$readme = Get-Content -Raw README.md
$docMatches = [regex]::Matches($readme, "docs/[A-Za-z0-9\-]+\.md")

$missing = @()
foreach ($m in $docMatches) {
    $path = $m.Value
    if (-not (Test-Path $path)) {
        $missing += $path
    }
}

if ($missing.Count -gt 0) {
    Write-Error ("Missing docs references in README: " + ($missing -join ", "))
}

$cfgDoc = Get-Content -Raw docs/runtime-config-diagnostics.md
if ($cfgDoc -notmatch "迁移映射") {
    Write-Error "docs/runtime-config-diagnostics.md must include migration mapping section."
}

$boundaryDoc = Get-Content -Raw docs/runtime-module-boundaries.md
if ($boundaryDoc -notmatch "依赖方向") {
    Write-Error "docs/runtime-module-boundaries.md must include dependency direction section."
}

$pre1Issues = @()
$roadmapDoc = Get-Content -Raw docs/development-roadmap.md
$versioningDoc = Get-Content -Raw docs/versioning-and-compatibility.md

foreach ($marker in @(
    '版本阶段口径（延续 0.x）',
    '不做 `1.0.0` / prod-ready 承诺',
    '允许新增能力型提案',
    '新增提案准入规则（0.x 阶段）'
)) {
    if ($roadmapDoc -notmatch [regex]::Escape($marker)) {
        $pre1Issues += "docs/development-roadmap.md missing marker: $marker"
    }
}

foreach ($marker in @(
    '`Why now`',
    '风险',
    '回滚',
    '文档影响',
    '验证命令'
)) {
    if ($roadmapDoc -notmatch [regex]::Escape($marker)) {
        $pre1Issues += "docs/development-roadmap.md missing proposal admission field marker: $marker"
    }
}

foreach ($marker in @(
    '契约一致性',
    '可靠性与安全',
    '质量门禁回归治理',
    '外部接入 DX'
)) {
    if ($roadmapDoc -notmatch [regex]::Escape($marker)) {
        $pre1Issues += "docs/development-roadmap.md missing bounded objective category: $marker"
    }
}

foreach ($marker in @(
    '长期方向（不进入近期主线）',
    '平台化控制面',
    '跨租户全局调度与控制平面',
    '市场化/托管化 adapter registry 能力'
)) {
    if ($roadmapDoc -notmatch [regex]::Escape($marker)) {
        $pre1Issues += "docs/development-roadmap.md missing long-term deferral marker: $marker"
    }
}

foreach ($marker in @(
    'pre-`1.0.0`',
    'does **not** imply `1.0.0/prod-ready` commitments',
    'Pre-1 Proposal Admission Baseline',
    'Capability additions are allowed in `0.x`'
)) {
    if ($versioningDoc -notmatch [regex]::Escape($marker)) {
        $pre1Issues += "docs/versioning-and-compatibility.md missing marker: $marker"
    }
}

foreach ($marker in @(
    '版本阶段快照',
    '`0.x` pre-1 阶段',
    '不做 `1.0.0/prod-ready` 承诺',
    '`0.x` 阶段允许新增能力型提案'
)) {
    if ($readme -notmatch [regex]::Escape($marker)) {
        $pre1Issues += "README.md missing pre-1 release snapshot marker: $marker"
    }
}

if ($pre1Issues.Count -gt 0) {
    Write-Error ("[pre1-governance] missing or stale pre-1 governance entries: " + ($pre1Issues -join "; "))
}

$adapterIssues = @()
$adapterDocs = @(
    "docs/external-adapter-template-index.md",
    "docs/adapter-migration-mapping.md"
)
foreach ($path in $adapterDocs) {
    if (-not (Test-Path $path)) {
        $adapterIssues += "missing required adapter doc file: $path"
    }
    elseif ($readme -notmatch [regex]::Escape($path)) {
        $adapterIssues += "README missing adapter doc link: $path"
    }
}

$apiRef = Get-Content -Raw docs/api-reference-d1.md
foreach ($path in $adapterDocs) {
    if ($apiRef -notmatch [regex]::Escape($path)) {
        $adapterIssues += "docs/api-reference-d1.md missing adapter doc link: $path"
    }
}

if ($apiRef -notmatch "MCP adapter template" -or $apiRef -notmatch "Model provider adapter template" -or $apiRef -notmatch "Tool adapter template") {
    $adapterIssues += "docs/api-reference-d1.md missing adapter onboarding category navigation."
}

if (Test-Path "docs/external-adapter-template-index.md") {
    $templateIndex = Get-Content -Raw docs/external-adapter-template-index.md
    foreach ($marker in @(
        "MCP adapter template",
        "Model provider adapter template",
        "Tool adapter template",
        "onboarding skeleton",
        "check-adapter-conformance.ps1",
        "check-adapter-conformance.sh"
    )) {
        if ($templateIndex -notmatch [regex]::Escape($marker)) {
            $adapterIssues += "docs/external-adapter-template-index.md missing marker: $marker"
        }
    }
}

if (Test-Path "docs/adapter-migration-mapping.md") {
    $mappingDoc = Get-Content -Raw docs/adapter-migration-mapping.md
    foreach ($marker in @(
        "capability-domain",
        "code-snippet",
        "previous pattern",
        "recommended pattern",
        "compatibility notes",
        "additive + nullable + default + fail-fast",
        "check-adapter-conformance.ps1",
        "check-adapter-conformance.sh"
    )) {
        if ($mappingDoc -notmatch [regex]::Escape($marker)) {
            $adapterIssues += "docs/adapter-migration-mapping.md missing marker: $marker"
        }
    }
}

foreach ($path in @(
    "scripts/check-adapter-conformance.sh",
    "scripts/check-adapter-conformance.ps1"
)) {
    if (-not (Test-Path $path)) {
        $adapterIssues += "missing adapter conformance script: $path"
    }
}

if ($adapterIssues.Count -gt 0) {
    Write-Error ("[adapter-docs] missing or stale adapter template/mapping entries: " + ($adapterIssues -join "; "))
}

Invoke-NativeStrict -Label "go test ./tool/contributioncheck (docs consistency suite)" -Command {
    go test ./tool/contributioncheck -run '^(TestMainlineContractIndexReferencesExistingTests|TestAdapterOnboardingDocsConsistency|TestPre1GovernanceDocsConsistency|TestValidatePre1GovernanceDocsDetectsStageConflict|TestReleaseStatusParityDocsConsistency|TestValidateStatusParityDetectsConflict|TestCoreModuleReadmeRichnessBaseline|TestValidateCoreModuleReadmeRichnessDetectsMissingSection)$' -count=1
}

Write-Host "Docs consistency check passed."
