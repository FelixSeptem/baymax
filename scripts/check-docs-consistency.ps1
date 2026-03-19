Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

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
        "onboarding skeleton"
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
        "additive + nullable + default + fail-fast"
    )) {
        if ($mappingDoc -notmatch [regex]::Escape($marker)) {
            $adapterIssues += "docs/adapter-migration-mapping.md missing marker: $marker"
        }
    }
}

if ($adapterIssues.Count -gt 0) {
    Write-Error ("[adapter-docs] missing or stale adapter template/mapping entries: " + ($adapterIssues -join "; "))
}

go test ./tool/contributioncheck -run '^(TestMainlineContractIndexReferencesExistingTests|TestAdapterOnboardingDocsConsistency)$' -count=1

Write-Host "Docs consistency check passed."
