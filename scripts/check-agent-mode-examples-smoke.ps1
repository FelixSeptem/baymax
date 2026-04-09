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

function Assert-Contains {
    param(
        [Parameter(Mandatory = $true)][string]$Output,
        [Parameter(Mandatory = $true)][string]$Token,
        [Parameter(Mandatory = $true)][string]$Entry
    )
    if (-not $Output.Contains($Token)) {
        throw "[agent-mode-examples-smoke][agent-mode-smoke-semantic-evidence-missing] missing token '$Token': $Entry"
    }
}

function Get-Value {
    param(
        [Parameter(Mandatory = $true)][string]$Output,
        [Parameter(Mandatory = $true)][string]$Key
    )
    $line = ($Output -split "`r?`n" | Where-Object { $_.StartsWith($Key + "=") } | Select-Object -Last 1)
    if ([string]::IsNullOrWhiteSpace($line)) {
        return ""
    }
    return $line.Substring($Key.Length + 1)
}

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
    $variantOutput = @{}

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

        Assert-Contains -Output $joined -Token "verification.mainline_runtime_path=ok" -Entry $entryRelative
        Assert-Contains -Output $joined -Token "verification.semantic.anchor=" -Entry $entryRelative
        Assert-Contains -Output $joined -Token "verification.semantic.classification=" -Entry $entryRelative
        Assert-Contains -Output $joined -Token "verification.semantic.runtime_path=" -Entry $entryRelative
        Assert-Contains -Output $joined -Token "verification.semantic.expected_markers=" -Entry $entryRelative
        Assert-Contains -Output $joined -Token "verification.semantic.governance=" -Entry $entryRelative
        Assert-Contains -Output $joined -Token "verification.semantic.marker_count=" -Entry $entryRelative
        Assert-Contains -Output $joined -Token "verification.semantic.marker." -Entry $entryRelative
        Assert-Contains -Output $joined -Token "result.final_answer=" -Entry $entryRelative
        Assert-Contains -Output $joined -Token "result.signature=" -Entry $entryRelative

        $variantOutput[$variant] = $joined
    }

    $hasMinimal = $selectedVariants -contains "minimal"
    $hasProduction = $selectedVariants -contains "production-ish"
    if ($hasMinimal -and $hasProduction) {
        $minimalOutput = if ($variantOutput.ContainsKey("minimal")) { [string]$variantOutput["minimal"] } else { "" }
        $productionOutput = if ($variantOutput.ContainsKey("production-ish")) { [string]$variantOutput["production-ish"] } else { "" }

        if ([string]::IsNullOrWhiteSpace($minimalOutput) -or [string]::IsNullOrWhiteSpace($productionOutput)) {
            throw "[agent-mode-examples-smoke][agent-mode-smoke-semantic-evidence-missing] missing dual-variant output for pattern=$pattern"
        }

        $minimalExpected = Get-Value -Output $minimalOutput -Key "verification.semantic.expected_markers"
        $productionExpected = Get-Value -Output $productionOutput -Key "verification.semantic.expected_markers"
        if ([string]::IsNullOrWhiteSpace($minimalExpected) -or [string]::IsNullOrWhiteSpace($productionExpected) -or $minimalExpected -eq $productionExpected) {
            throw "[agent-mode-examples-smoke][agent-mode-smoke-semantic-evidence-missing] expected marker set did not diverge for pattern=$pattern"
        }

        $minimalGovernance = Get-Value -Output $minimalOutput -Key "verification.semantic.governance"
        $productionGovernance = Get-Value -Output $productionOutput -Key "verification.semantic.governance"
        if ($minimalGovernance -ne "baseline" -or $productionGovernance -ne "enforced") {
            throw "[agent-mode-examples-smoke][agent-mode-smoke-semantic-evidence-missing] governance marker mismatch for pattern=$pattern minimal=$minimalGovernance production-ish=$productionGovernance"
        }

        $minimalSignature = Get-Value -Output $minimalOutput -Key "result.signature"
        $productionSignature = Get-Value -Output $productionOutput -Key "result.signature"
        if ([string]::IsNullOrWhiteSpace($minimalSignature) -or [string]::IsNullOrWhiteSpace($productionSignature) -or $minimalSignature -eq $productionSignature) {
            throw "[agent-mode-examples-smoke][agent-mode-smoke-semantic-evidence-missing] result.signature must differ between variants for pattern=$pattern"
        }
    }
}

Write-Host "[agent-mode-examples-smoke] done"