Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[agent-mode-anti-template-contract] validating anti-template constraints for agent-mode examples"

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

$templateSkeletonDetected = New-Object 'System.Collections.Generic.List[string]'
$semanticOwnershipMissing = New-Object 'System.Collections.Generic.List[string]'
$variantBehaviorNotDiverged = New-Object 'System.Collections.Generic.List[string]'
$structuralHomogeneityDetected = New-Object 'System.Collections.Generic.List[string]'
$missingSemanticFiles = New-Object 'System.Collections.Generic.List[string]'
$wrapperOnlyEntrypoints = New-Object 'System.Collections.Generic.List[string]'

$hashCounts = @{}
$hashPatterns = @{}

foreach ($pattern in $requiredPatterns) {
    $semanticFile = Join-Path $repoRoot ("examples/agent-modes/{0}/semantic_example.go" -f $pattern)
    if (-not (Test-Path -LiteralPath $semanticFile -PathType Leaf)) {
        $missingSemanticFiles.Add("$semanticFile:missing") | Out-Null
        continue
    }

    $semanticRaw = Get-Content -Path $semanticFile -Raw

    if ($semanticRaw.Contains("type modeSemanticModel struct") -and
        $semanticRaw.Contains("type modeSemanticStepTool struct{}") -and
        $semanticRaw.Contains("func runVariant(variant string)")) {
        $templateSkeletonDetected.Add("${pattern}:modeSemanticModel/modeSemanticStepTool skeleton detected") | Out-Null
    }

    if ($semanticRaw.Contains("steps := expectedSemanticSteps(m.variant)") -and
        $semanticRaw.Contains("for idx, step := range steps") -and
        $semanticRaw.Contains("semanticToolName = ""mode_")) {
        $semanticOwnershipMissing.Add("${pattern}:generic expectedSemanticSteps pipeline detected") | Out-Null
    }

    if ($semanticRaw.Contains('governance := strings.HasPrefix(marker, "governance_") || variant == modecommon.VariantProduction')) {
        $variantBehaviorNotDiverged.Add("${pattern}:governance branch inferred from marker naming") | Out-Null
    }

    $normalized = [regex]::Replace($semanticRaw, '"([^"\\]|\\.)*"', '"<s>"')
    $normalized = [regex]::Replace($normalized, '\s+', '')
    $hashBytes = [System.Security.Cryptography.SHA256]::HashData([System.Text.Encoding]::UTF8.GetBytes($normalized))
    $fingerprint = [BitConverter]::ToString($hashBytes).Replace("-", "").ToLowerInvariant()

    if (-not $hashCounts.ContainsKey($fingerprint)) {
        $hashCounts[$fingerprint] = 0
        $hashPatterns[$fingerprint] = New-Object 'System.Collections.Generic.List[string]'
    }
    $hashCounts[$fingerprint] = [int]$hashCounts[$fingerprint] + 1
    $hashPatterns[$fingerprint].Add($pattern) | Out-Null
}

$homogeneityThresholdRaw = if ($env:BAYMAX_AGENT_MODE_TEMPLATE_HOMOGENEITY_THRESHOLD) { $env:BAYMAX_AGENT_MODE_TEMPLATE_HOMOGENEITY_THRESHOLD.Trim() } else { "3" }
$homogeneityThreshold = 3
if (-not [int]::TryParse($homogeneityThresholdRaw, [ref]$homogeneityThreshold) -or $homogeneityThreshold -lt 2) {
    throw "[agent-mode-anti-template-contract] BAYMAX_AGENT_MODE_TEMPLATE_HOMOGENEITY_THRESHOLD must be integer >= 2"
}

foreach ($fingerprint in $hashCounts.Keys) {
    $count = [int]$hashCounts[$fingerprint]
    if ($count -ge $homogeneityThreshold) {
        $patterns = ($hashPatterns[$fingerprint] -join ",")
        $structuralHomogeneityDetected.Add("hash=$fingerprint count=$count patterns=$patterns") | Out-Null
    }
}

$mainFiles = Get-ChildItem -Path "examples/agent-modes" -Recurse -Filter main.go -File |
Where-Object { $_.FullName -notmatch "\\internal\\" } |
Sort-Object -Property FullName
foreach ($mainFile in $mainFiles) {
    $raw = Get-Content -Path $mainFile.FullName -Raw
    if ($raw.Contains("runtimeexample.MustRun(")) {
        $wrapperOnlyEntrypoints.Add("$($mainFile.FullName):runtimeexample.MustRun wrapper detected") | Out-Null
    }
}

if ($missingSemanticFiles.Count -gt 0) {
    Write-Host "[agent-mode-anti-template-contract][agent-mode-template-skeleton-detected] missing semantic files:"
    foreach ($item in $missingSemanticFiles) {
        Write-Host "  - $item"
    }
}

if ($templateSkeletonDetected.Count -gt 0 -or $wrapperOnlyEntrypoints.Count -gt 0 -or $structuralHomogeneityDetected.Count -gt 0) {
    Write-Host "[agent-mode-anti-template-contract][agent-mode-template-skeleton-detected] template skeleton regressions detected:"
    foreach ($item in $templateSkeletonDetected) {
        Write-Host "  - $item"
    }
    foreach ($item in $wrapperOnlyEntrypoints) {
        Write-Host "  - $item"
    }
    foreach ($item in $structuralHomogeneityDetected) {
        Write-Host "  - $item"
    }
}

if ($semanticOwnershipMissing.Count -gt 0) {
    Write-Host "[agent-mode-anti-template-contract][agent-mode-semantic-ownership-missing] mode-owned semantic execution missing:"
    foreach ($item in $semanticOwnershipMissing) {
        Write-Host "  - $item"
    }
}

if ($variantBehaviorNotDiverged.Count -gt 0) {
    Write-Host "[agent-mode-anti-template-contract][agent-mode-variant-behavior-not-diverged] variant behavior appears marker-only:"
    foreach ($item in $variantBehaviorNotDiverged) {
        Write-Host "  - $item"
    }
}

if ($missingSemanticFiles.Count -gt 0 -or
    $templateSkeletonDetected.Count -gt 0 -or
    $wrapperOnlyEntrypoints.Count -gt 0 -or
    $structuralHomogeneityDetected.Count -gt 0 -or
    $semanticOwnershipMissing.Count -gt 0 -or
    $variantBehaviorNotDiverged.Count -gt 0) {
    exit 1
}

Write-Host "[agent-mode-anti-template-contract] passed"
