Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
# Contract suite covers fixture privacy/bounds, pressure/quality, strategy drift,
# corpus advisory isolation, ReplayJSON idempotency, and Run/Stream parity.
$cache = if ($env:GOCACHE) { $env:GOCACHE } else { Join-Path $repoRoot ".gocache-tool-schema-audit-gate" }
New-Item -ItemType Directory -Force -Path $cache | Out-Null
$env:GOCACHE = $cache
go test ./tool/schemaaudit ./tool/diagnosticsreplay ./tool/contributioncheck -count=1
$fixtureDir = Join-Path $repoRoot "tool/schemaaudit/testdata"
foreach ($fixture in @("synthetic_within_budget.json", "synthetic_pressure_only.json", "synthetic_quality_only.json", "synthetic_pressure_quality.json", "synthetic_gold_conflict.json", "synthetic_overflow.json", "synthetic_strategy_drift.json", "corpus_advisory_optional.json")) {
    if (-not (Test-Path -LiteralPath (Join-Path $fixtureDir $fixture))) { throw "[schema-audit-gate] missing fixture $fixture" }
}
$auditFiles = @(Get-ChildItem -LiteralPath (Join-Path $repoRoot "tool/schemaaudit") -Filter "*.go" | Select-Object -ExpandProperty FullName)
$auditFiles += Join-Path $repoRoot "tool/diagnosticsreplay/schema_audit.go"
$matches = rg -n 'github\.com/(openai|anthropics)|google\.golang\.org/genai|tokenizer|ModelRequest|registry\.Register|net/http|os\.Open|os\.WriteFile|globalSelector|globalRouter|dynamicDownload|marketplaceResolve|persistRawSchema|persistPrompt|persistModelOutput|persistToolResult' @auditFiles 2>$null
if ($LASTEXITCODE -eq 0 -and $matches) { throw "[schema-audit-gate] forbidden runtime/provider/persistence reference detected" }
Write-Host "[schema-audit-gate] offline schema pressure/selection audit contract passed"
