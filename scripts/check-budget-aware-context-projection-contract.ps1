Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) { $env:GOCACHE = Join-Path $repoRoot ".gocache" }

$contractDir = "context/budgetprojection"
$fixture = "tool/diagnosticsreplay/testdata/budget_projection.v1.json"
$fixtureVersion = "budget_projection.v1"
$projectionFile = Join-Path $contractDir "projection.go"
$taxonomyDoc = "context/README.md"

Write-Host "[budget-aware-context-projection] fixture presence, version, and bounds"
if (-not (Test-Path -LiteralPath $fixture)) { throw "budget_projection_schema_drift: missing fixture $fixture" }
if ((Get-Item -LiteralPath $fixture).Length -gt 2097152) { throw "budget_projection_overflow_drift: fixture exceeds 2 MiB" }
$fixtureText = Get-Content -LiteralPath $fixture -Raw
if ($fixtureText -notmatch [regex]::Escape($fixtureVersion)) { throw "budget_projection_schema_drift: fixture does not declare $fixtureVersion" }

Write-Host "[budget-aware-context-projection] stable classification taxonomy"
$budgetProjectionCodes = @(
    "budget_projection_schema_drift",
    "budget_projection_unknown_version",
    "budget_facts_missing_iteration_limit",
    "budget_facts_missing_tool_call_limit",
    "budget_facts_missing_time_budget",
    "budget_facts_missing_cost_threshold",
    "budget_projection_negative_remaining",
    "budget_projection_ratio_out_of_range",
    "budget_projection_pressure_level_mismatch",
    "budget_projection_note_unbounded",
    "budget_projection_digest_mismatch",
    "budget_projection_overflow_drift",
    "budget_projection_writeback_shape_detected",
    "budget_benchmark_schema_drift",
    "budget_benchmark_metric_mismatch",
    "budget_benchmark_recovery_recompute_drift",
    "budget_benchmark_overflow_drift",
    "budget_projection_replay_not_idempotent"
)
$contractText = Get-Content -LiteralPath $projectionFile -Raw
$taxonomyText = Get-Content -LiteralPath $taxonomyDoc -Raw
foreach ($code in $budgetProjectionCodes) {
    if ($contractText -notmatch [regex]::Escape('"' + $code + '"')) {
        throw "budget_projection_taxonomy_stable: $code missing from $projectionFile"
    }
    if ($taxonomyText -notmatch [regex]::Escape($code)) {
        throw "budget_projection_taxonomy_stable: $code missing from $taxonomyDoc"
    }
}

Write-Host "[budget-aware-context-projection] read-only contract and boundedness markers"
if ($contractText -notmatch [regex]::Escape("MaxProjectionBytes = 4096")) {
    throw "budget_projection_bounded: projection byte bound missing from $projectionFile"
}
if ($contractText -notmatch [regex]::Escape("MaxTraces          = 32")) {
    throw "budget_projection_bounded: benchmark trace bound missing from $projectionFile"
}
if ($contractText -notmatch [regex]::Escape("Available bool")) {
    throw "budget_projection_nullable_default: nullable dimension marker missing from $projectionFile"
}

Write-Host "[budget-aware-context-projection] dependency-free contract boundary"
Invoke-NativeStrict -Label "budget projection contract boundary" -Command { go test ./tool/contributioncheck -run 'TestBudgetProjectionContractBoundary' -count=1 }

Write-Host "[budget-aware-context-projection] derived projection and benchmark suites"
Invoke-NativeStrict -Label "budget projection contract suites" -Command { go test ./context/budgetprojection -count=1 }

Write-Host "[budget-aware-context-projection] offline replay idempotency"
Invoke-NativeStrict -Label "budget projection replay idempotency" -Command { go test ./tool/diagnosticsreplay -run 'BudgetProjection' -count=2 }

Write-Host "[budget-aware-context-projection-gate] passed"
