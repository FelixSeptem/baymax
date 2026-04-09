Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$defaultGoCache = Join-Path $repoRoot ".tmp/go-cache-agent-mode-smoke"
if ([string]::IsNullOrWhiteSpace($env:GOCACHE)) {
    $env:GOCACHE = $defaultGoCache
}
if (-not (Test-Path -LiteralPath $env:GOCACHE -PathType Container)) {
    New-Item -ItemType Directory -Path $env:GOCACHE -Force | Out-Null
}

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

$selectedPatterns = New-Object 'System.Collections.Generic.List[string]'
if (-not [string]::IsNullOrWhiteSpace($env:BAYMAX_AGENT_MODE_SMOKE_PATTERNS)) {
    $requested = $env:BAYMAX_AGENT_MODE_SMOKE_PATTERNS.Split(",")
    foreach ($raw in $requested) {
        $pattern = $raw.Trim()
        if ([string]::IsNullOrWhiteSpace($pattern)) {
            continue
        }
        if ($requiredPatterns -notcontains $pattern) {
            throw "[agent-mode-examples-smoke] unsupported pattern in BAYMAX_AGENT_MODE_SMOKE_PATTERNS: $pattern"
        }
        $selectedPatterns.Add($pattern) | Out-Null
    }
}
else {
    foreach ($pattern in $requiredPatterns) {
        $selectedPatterns.Add($pattern) | Out-Null
    }
}

$selectedVariants = New-Object 'System.Collections.Generic.List[string]'
if (-not [string]::IsNullOrWhiteSpace($env:BAYMAX_AGENT_MODE_SMOKE_VARIANTS)) {
    $requested = $env:BAYMAX_AGENT_MODE_SMOKE_VARIANTS.Split(",")
    foreach ($raw in $requested) {
        $variant = $raw.Trim()
        if ([string]::IsNullOrWhiteSpace($variant)) {
            continue
        }
        if ($variant -ne "minimal" -and $variant -ne "production-ish") {
            throw "[agent-mode-examples-smoke] unsupported variant: $variant"
        }
        $selectedVariants.Add($variant) | Out-Null
    }
}
else {
    $selectedVariants.Add("minimal") | Out-Null
    $selectedVariants.Add("production-ish") | Out-Null
}

if ($selectedPatterns.Count -eq 0) {
    throw "[agent-mode-examples-smoke] no patterns selected"
}
if ($selectedVariants.Count -eq 0) {
    throw "[agent-mode-examples-smoke] no variants selected"
}

Write-Host "[agent-mode-examples-smoke] running smoke checks for $($selectedPatterns.Count) patterns and $($selectedVariants.Count) variants"

foreach ($pattern in $selectedPatterns) {
    foreach ($variant in $selectedVariants) {
        $entryRelative = "./examples/agent-modes/$pattern/$variant"
        $entryFull = Join-Path $repoRoot "examples/agent-modes/$pattern/$variant"
        if (-not (Test-Path -LiteralPath $entryFull -PathType Container)) {
            throw "[agent-mode-examples-smoke] missing example directory: $entryRelative"
        }
        $output = Invoke-NativeCaptureStrict -Label ("go run " + $entryRelative) -Command {
            go run $entryRelative
        }
        foreach ($line in $output) {
            if ($null -eq $line) {
                continue
            }
            if ($line -is [System.Management.Automation.ErrorRecord]) {
                Write-Host ($line.ToString())
                continue
            }
            Write-Host ([string]$line)
        }

        $joined = ($output | ForEach-Object {
                if ($null -eq $_) {
                    return ""
                }
                if ($_ -is [System.Management.Automation.ErrorRecord]) {
                    return $_.ToString()
                }
                return [string]$_
            }) -join "`n"
        if (-not $joined.Contains("verification.mainline_runtime_path=ok")) {
            throw "[agent-mode-examples-smoke] missing runtime path verification marker: $entryRelative"
        }
        if (-not $joined.Contains("result.final_answer=")) {
            throw "[agent-mode-examples-smoke] missing final answer marker: $entryRelative"
        }
        if (-not $joined.Contains("result.signature=")) {
            throw "[agent-mode-examples-smoke] missing signature marker: $entryRelative"
        }
    }
}

Write-Host "[agent-mode-examples-smoke] done"
