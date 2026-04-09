Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[agent-mode-real-runtime-semantic-contract] validating agent mode semantic runtime contract"

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

$runtimeFile = Join-Path $repoRoot "examples/agent-modes/internal/runtimeexample/runtime.go"
$specFile = Join-Path $repoRoot "examples/agent-modes/internal/runtimeexample/specs.go"
$matrixFile = Join-Path $repoRoot "examples/agent-modes/MATRIX.md"

$sharedSemanticEngine = New-Object 'System.Collections.Generic.List[string]'
$semanticOwnershipMissing = New-Object 'System.Collections.Generic.List[string]'
$runtimePathMissing = New-Object 'System.Collections.Generic.List[string]'

if (-not (Test-Path -LiteralPath $specFile -PathType Leaf)) {
    $semanticOwnershipMissing.Add("$specFile:missing") | Out-Null
}
if (-not (Test-Path -LiteralPath $matrixFile -PathType Leaf)) {
    $runtimePathMissing.Add("$matrixFile:missing") | Out-Null
}

if (Test-Path -LiteralPath $runtimeFile -PathType Leaf) {
    $runtimeRaw = Get-Content -Path $runtimeFile -Raw
    if ($runtimeRaw -match 'func\s+MustRun\(|type\s+semanticModel|semanticStepTool') {
        $sharedSemanticEngine.Add("$runtimeFile:legacy-shared-semantic-engine") | Out-Null
    }
}

$mainFiles = Get-ChildItem -Path "examples/agent-modes" -Recurse -Filter "main.go" -File |
Where-Object { $_.FullName -notmatch "\\internal\\" } |
Sort-Object -Property FullName

if ($mainFiles.Count -eq 0) {
    $semanticOwnershipMissing.Add("examples/agent-modes/*/*/main.go:missing") | Out-Null
}

foreach ($file in $mainFiles) {
    $variant = Split-Path -Path $file.DirectoryName -Leaf
    $pattern = Split-Path -Path (Split-Path -Path $file.DirectoryName -Parent) -Leaf
    $raw = Get-Content -Path $file.FullName -Raw

    if ($raw.Contains('runtimeexample.MustRun(')) {
        $sharedSemanticEngine.Add("$($file.FullName):shared-wrapper-detected") | Out-Null
    }

    $expectedImport = ('modeimpl "github.com/FelixSeptem/baymax/examples/agent-modes/{0}"' -f $pattern)
    if (-not $raw.Contains($expectedImport)) {
        $semanticOwnershipMissing.Add("$($file.FullName):missing-mode-owned-import:$expectedImport") | Out-Null
    }

    $expectedCall = if ($variant -eq "minimal") { "modeimpl.RunMinimal()" } else { "modeimpl.RunProduction()" }
    if (-not $raw.Contains($expectedCall)) {
        $semanticOwnershipMissing.Add("$($file.FullName):missing-mode-owned-entry:$expectedCall") | Out-Null
    }
}

$requiredRuntimeTokens = @(
    "verification.mainline_runtime_path=",
    "verification.semantic.anchor=",
    "verification.semantic.classification=",
    "verification.semantic.runtime_path=",
    "verification.semantic.expected_markers=",
    "verification.semantic.governance=",
    "verification.semantic.marker_count=",
    "verification.semantic.marker."
)

foreach ($pattern in $requiredPatterns) {
    $semanticFile = Join-Path $repoRoot ("examples/agent-modes/{0}/semantic_example.go" -f $pattern)
    if (-not (Test-Path -LiteralPath $semanticFile -PathType Leaf)) {
        $semanticOwnershipMissing.Add("$semanticFile:missing") | Out-Null
        continue
    }

    $semanticRaw = Get-Content -Path $semanticFile -Raw
    if (-not $semanticRaw.Contains(('patternName      = "{0}"' -f $pattern))) {
        $semanticOwnershipMissing.Add("$semanticFile:missing-pattern-constant") | Out-Null
    }

    $requiredSemanticTokens = @(
        "func RunMinimal()",
        "func RunProduction()",
        "var minimalSemanticSteps",
        "var productionGovernanceSteps",
        'semanticToolName = "mode_'
    )
    foreach ($token in $requiredSemanticTokens) {
        if (-not $semanticRaw.Contains($token)) {
            $semanticOwnershipMissing.Add("$semanticFile:missing-token:$token") | Out-Null
        }
    }

    $markerCount = [regex]::Matches($semanticRaw, 'Marker:').Count
    if ($markerCount -lt 5) {
        $semanticOwnershipMissing.Add("$semanticFile:insufficient-semantic-steps") | Out-Null
    }

    foreach ($token in $requiredRuntimeTokens) {
        if (-not $semanticRaw.Contains($token)) {
            $runtimePathMissing.Add("$semanticFile:missing-token:$token") | Out-Null
        }
    }
}

if (Test-Path -LiteralPath $specFile -PathType Leaf) {
    $specRaw = Get-Content -Path $specFile -Raw
    $anchorMatches = [regex]::Matches($specRaw, 'SemanticAnchor:\s+"([^"]+)"')
    $anchorCount = @{}
    foreach ($match in $anchorMatches) {
        $anchor = $match.Groups[1].Value
        if ([string]::IsNullOrWhiteSpace($anchor)) {
            continue
        }
        if (-not $anchorCount.ContainsKey($anchor)) {
            $anchorCount[$anchor] = 0
        }
        $anchorCount[$anchor]++
    }
    foreach ($pair in $anchorCount.GetEnumerator()) {
        if ($pair.Value -gt 1) {
            $semanticOwnershipMissing.Add("$specFile:duplicate-semantic-anchor:$($pair.Key)") | Out-Null
        }
    }

    foreach ($pattern in $requiredPatterns) {
        if (-not $specRaw.Contains(("`"{0}`":" -f $pattern))) {
            $semanticOwnershipMissing.Add("$specFile:missing-pattern-spec:$pattern") | Out-Null
        }
    }
}

if (Test-Path -LiteralPath $matrixFile -PathType Leaf) {
    $matrixRaw = Get-Content -Path $matrixFile -Raw
    $matrixLines = Get-Content -Path $matrixFile
    if (-not $matrixRaw.Contains("semantic_anchor -> runtime_path_evidence -> expected_verification_markers") -and
        -not $matrixRaw.Contains("semantic_anchor -> runtime_path_evidence")) {
        $runtimePathMissing.Add("$matrixFile:missing-semantic-runtime-columns") | Out-Null
    }

    foreach ($pattern in $requiredPatterns) {
        $token = "| ``$pattern`` |"
        $row = ($matrixLines | Where-Object { $_.Contains($token) } | Select-Object -First 1)
        if ([string]::IsNullOrWhiteSpace($row)) {
            $runtimePathMissing.Add("$matrixFile:missing-row:$pattern") | Out-Null
            continue
        }
        if (-not $row.Contains("runtime/config")) {
            $runtimePathMissing.Add("$matrixFile:missing-runtime-path-evidence:$pattern") | Out-Null
        }
        if (-not $row.Contains("minimal:") -or -not $row.Contains("production-ish:")) {
            $runtimePathMissing.Add("$matrixFile:missing-expected-marker-evidence:$pattern") | Out-Null
        }
    }
}

if ($sharedSemanticEngine.Count -gt 0) {
    Write-Host "[agent-mode-real-runtime-semantic-contract][agent-mode-shared-semantic-engine-detected] shared semantic engine regressions detected:"
    foreach ($item in $sharedSemanticEngine) {
        Write-Host "  - $item"
    }
}

if ($semanticOwnershipMissing.Count -gt 0) {
    Write-Host "[agent-mode-real-runtime-semantic-contract][agent-mode-semantic-ownership-missing] per-mode semantic ownership is incomplete:"
    foreach ($item in $semanticOwnershipMissing) {
        Write-Host "  - $item"
    }
}

if ($runtimePathMissing.Count -gt 0) {
    Write-Host "[agent-mode-real-runtime-semantic-contract][agent-mode-missing-runtime-path-evidence] runtime path evidence is incomplete:"
    foreach ($item in $runtimePathMissing) {
        Write-Host "  - $item"
    }
}

if ($sharedSemanticEngine.Count -gt 0 -or $semanticOwnershipMissing.Count -gt 0 -or $runtimePathMissing.Count -gt 0) {
    exit 1
}

Write-Host "[agent-mode-real-runtime-semantic-contract] passed"
