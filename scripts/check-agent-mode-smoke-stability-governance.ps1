Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$defaultGoCache = Join-Path $repoRoot ".tmp/go-cache-agent-mode-smoke-stability"
if ([string]::IsNullOrWhiteSpace($env:GOCACHE)) {
    $env:GOCACHE = $defaultGoCache
}
if (-not (Test-Path -LiteralPath $env:GOCACHE -PathType Container)) {
    New-Item -ItemType Directory -Path $env:GOCACHE -Force | Out-Null
}

$baselinePath = if (-not [string]::IsNullOrWhiteSpace($env:BAYMAX_AGENT_MODE_STABILITY_BASELINE_PATH)) {
    $env:BAYMAX_AGENT_MODE_STABILITY_BASELINE_PATH
}
else {
    "examples/agent-modes/STABILITY_BASELINE.json"
}
$reportPath = if (-not [string]::IsNullOrWhiteSpace($env:BAYMAX_AGENT_MODE_STABILITY_REPORT_PATH)) {
    $env:BAYMAX_AGENT_MODE_STABILITY_REPORT_PATH
}
else {
    ".tmp/agent-mode-smoke-stability-last-run.json"
}
$timeoutSec = if (-not [string]::IsNullOrWhiteSpace($env:BAYMAX_AGENT_MODE_STABILITY_TIMEOUT_SEC)) {
    [int]$env:BAYMAX_AGENT_MODE_STABILITY_TIMEOUT_SEC
}
else {
    120
}
$retryMax = if (-not [string]::IsNullOrWhiteSpace($env:BAYMAX_AGENT_MODE_STABILITY_RETRY_MAX)) {
    [int]$env:BAYMAX_AGENT_MODE_STABILITY_RETRY_MAX
}
else {
    1
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

if (-not (Test-Path -LiteralPath $baselinePath -PathType Leaf)) {
    throw "[agent-mode-smoke-stability-governance][missing-checklist] missing baseline: $baselinePath"
}
$baseline = Get-Content -Path $baselinePath -Raw | ConvertFrom-Json
if ($null -eq $baseline.thresholds -or
    $null -eq $baseline.thresholds.max_p95_ms -or
    $null -eq $baseline.thresholds.max_flaky_rate -or
    $null -eq $baseline.thresholds.max_retry_rate) {
    throw "[agent-mode-smoke-stability-governance][missing-checklist] baseline missing thresholds: max_p95_ms/max_flaky_rate/max_retry_rate"
}

$maxP95Ms = [double]$baseline.thresholds.max_p95_ms
$maxFlakyRate = [double]$baseline.thresholds.max_flaky_rate
$maxRetryRate = [double]$baseline.thresholds.max_retry_rate

$selectedPatterns = New-Object 'System.Collections.Generic.List[string]'
if (-not [string]::IsNullOrWhiteSpace($env:BAYMAX_AGENT_MODE_STABILITY_PATTERNS)) {
    $requested = $env:BAYMAX_AGENT_MODE_STABILITY_PATTERNS.Split(",")
    foreach ($raw in $requested) {
        $pattern = $raw.Trim()
        if ([string]::IsNullOrWhiteSpace($pattern)) {
            continue
        }
        if ($requiredPatterns -notcontains $pattern) {
            throw "[agent-mode-smoke-stability-governance] unsupported pattern in BAYMAX_AGENT_MODE_STABILITY_PATTERNS: $pattern"
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
if (-not [string]::IsNullOrWhiteSpace($env:BAYMAX_AGENT_MODE_STABILITY_VARIANTS)) {
    $requested = $env:BAYMAX_AGENT_MODE_STABILITY_VARIANTS.Split(",")
    foreach ($raw in $requested) {
        $variant = $raw.Trim()
        if ([string]::IsNullOrWhiteSpace($variant)) {
            continue
        }
        if ($variant -ne "minimal" -and $variant -ne "production-ish") {
            throw "[agent-mode-smoke-stability-governance] unsupported variant: $variant"
        }
        $selectedVariants.Add($variant) | Out-Null
    }
}
else {
    $selectedVariants.Add("minimal") | Out-Null
}

if ($selectedPatterns.Count -eq 0) {
    throw "[agent-mode-smoke-stability-governance] no patterns selected"
}
if ($selectedVariants.Count -eq 0) {
    throw "[agent-mode-smoke-stability-governance] no variants selected"
}

function Invoke-GoRunWithTimeout {
    param(
        [Parameter(Mandatory = $true)][string]$EntryRelative,
        [Parameter(Mandatory = $true)][int]$TimeoutSeconds
    )

    $stdoutPath = [System.IO.Path]::GetTempFileName()
    $stderrPath = [System.IO.Path]::GetTempFileName()
    try {
        $proc = Start-Process -FilePath "go" -ArgumentList @("run", $EntryRelative) -NoNewWindow -PassThru -RedirectStandardOutput $stdoutPath -RedirectStandardError $stderrPath
        $finished = $proc.WaitForExit($TimeoutSeconds * 1000)
        if (-not $finished) {
            try {
                $proc.Kill($true)
            }
            catch {
            }
            return @{
                ExitCode = 124
                TimedOut = $true
                Stdout = Get-Content -Path $stdoutPath -Raw -ErrorAction SilentlyContinue
                Stderr = Get-Content -Path $stderrPath -Raw -ErrorAction SilentlyContinue
            }
        }
        return @{
            ExitCode = $proc.ExitCode
            TimedOut = $false
            Stdout = Get-Content -Path $stdoutPath -Raw -ErrorAction SilentlyContinue
            Stderr = Get-Content -Path $stderrPath -Raw -ErrorAction SilentlyContinue
        }
    }
    finally {
        Remove-Item -Path $stdoutPath -Force -ErrorAction SilentlyContinue
        Remove-Item -Path $stderrPath -Force -ErrorAction SilentlyContinue
    }
}

function Get-PercentileValue {
    param(
        [Parameter(Mandatory = $true)][double[]]$Values,
        [Parameter(Mandatory = $true)][double]$Percent
    )
    if ($Values.Count -eq 0) {
        return 0
    }
    $sorted = $Values | Sort-Object
    $index = [Math]::Floor(($sorted.Count - 1) * $Percent)
    return [int]$sorted[$index]
}

Write-Host "[agent-mode-smoke-stability-governance] running stability checks for $($selectedPatterns.Count) patterns and $($selectedVariants.Count) variants"

$durations = New-Object 'System.Collections.Generic.List[double]'
$totalCases = 0
$failedCases = 0
$retryTotal = 0
$flakyCases = 0
$runStart = Get-Date

foreach ($pattern in $selectedPatterns) {
    foreach ($variant in $selectedVariants) {
        $entryRelative = "./examples/agent-modes/$pattern/$variant"
        $entryFull = Join-Path $repoRoot "examples/agent-modes/$pattern/$variant"
        if (-not (Test-Path -LiteralPath $entryFull -PathType Container)) {
            throw "[agent-mode-smoke-stability-governance][missing-checklist] missing example directory: $entryRelative"
        }

        $totalCases += 1
        $attempt = 0
        $success = $false

        while ($attempt -le $retryMax) {
            $attempt += 1
            $caseStart = Get-Date
            $result = Invoke-GoRunWithTimeout -EntryRelative $entryRelative -TimeoutSeconds $timeoutSec
            $caseEnd = Get-Date
            $durationMs = [int][Math]::Round(($caseEnd - $caseStart).TotalMilliseconds)

            if (-not [string]::IsNullOrWhiteSpace($result.Stdout)) {
                Write-Host $result.Stdout.TrimEnd()
            }
            if ($result.ExitCode -eq 0) {
                $success = $true
                $durations.Add($durationMs) | Out-Null
                if ($attempt -gt 1) {
                    $flakyCases += 1
                    $retryTotal += ($attempt - 1)
                }
                Write-Host "[agent-mode-smoke-stability-governance][metric] pattern=$pattern variant=$variant duration_ms=$durationMs attempts=$attempt"
                break
            }

            if (-not [string]::IsNullOrWhiteSpace($result.Stderr)) {
                Write-Host $result.Stderr.TrimEnd()
            }

            if ($attempt -le $retryMax) {
                $retryTotal += 1
                Write-Host "[agent-mode-smoke-stability-governance][retry] pattern=$pattern variant=$variant attempt=$attempt exit=$($result.ExitCode)"
                continue
            }

            $failedCases += 1
            if ($result.TimedOut) {
                Write-Host "[agent-mode-smoke-stability-governance][timeout] pattern=$pattern variant=$variant timeout_sec=$timeoutSec"
            }
            Write-Host "[agent-mode-smoke-stability-governance][failure] pattern=$pattern variant=$variant exit=$($result.ExitCode)"
        }
    }
}

if ($totalCases -eq 0) {
    throw "[agent-mode-smoke-stability-governance][missing-checklist] no cases executed"
}

$p50Ms = Get-PercentileValue -Values $durations.ToArray() -Percent 0.50
$p95Ms = Get-PercentileValue -Values $durations.ToArray() -Percent 0.95
$elapsedMs = [int][Math]::Round(((Get-Date) - $runStart).TotalMilliseconds)
$failureRate = [math]::Round($failedCases / [double]$totalCases, 6)
$retryRate = [math]::Round($retryTotal / [double]$totalCases, 6)
$flakyRate = [math]::Round($flakyCases / [double]$totalCases, 6)

$reportDir = Split-Path -Path $reportPath -Parent
if (-not [string]::IsNullOrWhiteSpace($reportDir) -and -not (Test-Path -LiteralPath $reportDir -PathType Container)) {
    New-Item -ItemType Directory -Path $reportDir -Force | Out-Null
}
$report = [ordered]@{
    timestamp_utc = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
    total_cases = $totalCases
    failed_cases = $failedCases
    retry_total = $retryTotal
    flaky_cases = $flakyCases
    p50_ms = $p50Ms
    p95_ms = $p95Ms
    elapsed_ms = $elapsedMs
    failure_rate = $failureRate
    retry_rate = $retryRate
    flaky_rate = $flakyRate
}
$report | ConvertTo-Json -Depth 6 | Set-Content -Path $reportPath -Encoding utf8

Write-Host "[agent-mode-smoke-stability-governance] summary total=$totalCases failed=$failedCases retries=$retryTotal flaky=$flakyCases p50_ms=$p50Ms p95_ms=$p95Ms elapsed_ms=$elapsedMs"
Write-Host "[agent-mode-smoke-stability-governance] report=$reportPath"

$breach = $false
if ($p95Ms -gt $maxP95Ms) {
    Write-Host "[agent-mode-smoke-stability-governance][example-smoke-latency-regression] current_p95_ms=$p95Ms threshold_p95_ms=$maxP95Ms"
    $breach = $true
}
if ($flakyRate -gt $maxFlakyRate) {
    Write-Host "[agent-mode-smoke-stability-governance][example-smoke-flaky-regression] current_flaky_rate=$flakyRate threshold_flaky_rate=$maxFlakyRate"
    $breach = $true
}
if ($retryRate -gt $maxRetryRate) {
    Write-Host "[agent-mode-smoke-stability-governance][example-smoke-flaky-regression] current_retry_rate=$retryRate threshold_retry_rate=$maxRetryRate"
    $breach = $true
}
if ($failedCases -gt 0) {
    Write-Host "[agent-mode-smoke-stability-governance][example-smoke-flaky-regression] failed_cases=$failedCases"
    $breach = $true
}

if ($breach) {
    exit 1
}

Write-Host "[agent-mode-smoke-stability-governance] stability is within baseline thresholds"
