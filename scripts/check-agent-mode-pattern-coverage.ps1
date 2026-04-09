Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$matrixPath = Join-Path $repoRoot "examples/agent-modes/MATRIX.md"
$rootPath = Join-Path $repoRoot "examples/agent-modes"

$requiredPatterns = @(
    "rag-hybrid-retrieval",
    "structured-output-schema-contract",
    "skill-driven-discovery-hybrid",
    "mcp-governed-stdio-http",
    "hitl-governed-checkpoint",
    "context-governed-reference-first",
    "sandbox-governed-toolchain",
    "realtime-interrupt-resume",
    "multi-agents-collab-recovery",
    "workflow-branch-retry-failfast",
    "mapreduce-large-batch",
    "state-session-snapshot-recovery",
    "policy-budget-admission",
    "tracing-eval-smoke",
    "react-plan-notebook-loop",
    "hooks-middleware-extension-pipeline",
    "observability-export-bundle",
    "adapter-onboarding-manifest-capability",
    "security-policy-event-delivery",
    "config-hot-reload-rollback",
    "workflow-routing-strategy-switch",
    "multi-agents-hierarchical-planner-validator",
    "mainline-mailbox-async-delayed-reconcile",
    "mainline-task-board-query-control",
    "mainline-scheduler-qos-backoff-dlq",
    "mainline-readiness-admission-degradation",
    "custom-adapter-mcp-model-tool-memory-pack",
    "custom-adapter-health-readiness-circuit"
)

$requiredFamilies = @(
    "agent",
    "workflow",
    "rag",
    "mapreduce",
    "structured-output",
    "multi-agents",
    "skill",
    "mcp",
    "react",
    "hitl",
    "context",
    "sandbox",
    "realtime"
)

Write-Host "[agent-mode-pattern-coverage] validating matrix and skeleton coverage"

if (-not (Test-Path -LiteralPath $matrixPath -PathType Leaf)) {
    throw "[agent-mode-pattern-coverage] missing matrix: $matrixPath"
}

$matrixRaw = Get-Content -Path $matrixPath -Raw
$matrixLines = Get-Content -Path $matrixPath
if (-not $matrixRaw.Contains("pattern -> minimal -> production-ish -> contracts -> gates -> replay")) {
    throw "[agent-mode-pattern-coverage] matrix missing canonical column declaration"
}

$familyHits = @{}
foreach ($family in $requiredFamilies) {
    $familyHits[$family] = $false
}

$missingMatrixRows = New-Object 'System.Collections.Generic.List[string]'
$missingFiles = New-Object 'System.Collections.Generic.List[string]'

foreach ($pattern in $requiredPatterns) {
    $rowToken = "| ``$pattern`` |"
    if (-not ($matrixLines | Where-Object { $_.Contains($rowToken) })) {
        $missingMatrixRows.Add($pattern) | Out-Null
    }
    foreach ($variant in @("minimal", "production-ish")) {
        $base = Join-Path $rootPath "$pattern/$variant"
        if (-not (Test-Path -LiteralPath $base -PathType Container)) {
            $missingFiles.Add("$base/") | Out-Null
        }
        if (-not (Test-Path -LiteralPath (Join-Path $base "main.go") -PathType Leaf)) {
            $missingFiles.Add((Join-Path $base "main.go")) | Out-Null
        }
        if (-not (Test-Path -LiteralPath (Join-Path $base "README.md") -PathType Leaf)) {
            $missingFiles.Add((Join-Path $base "README.md")) | Out-Null
        }
    }

    if ($pattern -like "*agent*") {
        $familyHits["agent"] = $true
    }
    if ($pattern -like "workflow-*") {
        $familyHits["workflow"] = $true
    }
    if ($pattern -like "rag-*") {
        $familyHits["rag"] = $true
    }
    if ($pattern -like "mapreduce-*") {
        $familyHits["mapreduce"] = $true
    }
    if ($pattern -like "structured-output-*") {
        $familyHits["structured-output"] = $true
    }
    if ($pattern -like "multi-agents-*") {
        $familyHits["multi-agents"] = $true
    }
    if ($pattern -like "skill-*") {
        $familyHits["skill"] = $true
    }
    if ($pattern -like "mcp-*" -or $pattern -like "custom-adapter-mcp-*") {
        $familyHits["mcp"] = $true
    }
    if ($pattern -like "react-*") {
        $familyHits["react"] = $true
    }
    if ($pattern -like "hitl-*") {
        $familyHits["hitl"] = $true
    }
    if ($pattern -like "context-*") {
        $familyHits["context"] = $true
    }
    if ($pattern -like "sandbox-*") {
        $familyHits["sandbox"] = $true
    }
    if ($pattern -like "realtime-*") {
        $familyHits["realtime"] = $true
    }
}

$missingFamilies = New-Object 'System.Collections.Generic.List[string]'
foreach ($family in $requiredFamilies) {
    if (-not $familyHits[$family]) {
        $missingFamilies.Add($family) | Out-Null
    }
}

if ($missingMatrixRows.Count -gt 0) {
    Write-Host "[agent-mode-pattern-coverage] missing matrix rows:"
    foreach ($item in $missingMatrixRows) {
        Write-Host "  - $item"
    }
}
if ($missingFiles.Count -gt 0) {
    Write-Host "[agent-mode-pattern-coverage] missing pattern skeleton files:"
    foreach ($item in $missingFiles) {
        Write-Host "  - $item"
    }
}
if ($missingFamilies.Count -gt 0) {
    Write-Host "[agent-mode-pattern-coverage] missing required mode families:"
    foreach ($item in $missingFamilies) {
        Write-Host "  - $item"
    }
}

if ($missingMatrixRows.Count -gt 0 -or $missingFiles.Count -gt 0 -or $missingFamilies.Count -gt 0) {
    exit 1
}

Write-Host "[agent-mode-pattern-coverage] coverage is complete"
