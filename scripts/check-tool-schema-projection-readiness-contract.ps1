Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
# Contract covers stable-window pressure, subset integrity, privacy, overflow,
# replay determinism, and Run/Stream parity.
$cache = if ($env:GOCACHE) { $env:GOCACHE } else { Join-Path $repoRoot ".gocache-tool-schema-projection-readiness-gate" }
New-Item -ItemType Directory -Force -Path $cache | Out-Null
$env:GOCACHE = $cache
go test ./tool/schemaaudit ./tool/diagnosticsreplay ./tool/contributioncheck -run "Readiness|ToolSchemaProjection" -count=1
foreach ($fixture in @("tool_schema_projection_readiness.v1.json", "tool_schema_projection_readiness_overflow.json", "tool_schema_projection_readiness_privacy.json")) {
    if (-not (Test-Path -LiteralPath (Join-Path $repoRoot "tool/diagnosticsreplay/testdata/$fixture"))) { throw "[tool-schema-projection-readiness-gate] missing fixture $fixture" }
}
$files = @(Get-ChildItem -LiteralPath (Join-Path $repoRoot "tool/schemaaudit") -Filter "readiness*.go" | Select-Object -ExpandProperty FullName)
$files += Join-Path $repoRoot "tool/diagnosticsreplay/tool_schema_projection_readiness.go"
$matches = rg -n 'github\.com/(openai|anthropics)|google\.golang\.org/genai|tokenizer|ModelRequest|net/http|os\.Open|os\.WriteFile|globalSelector|globalRouter|dynamicDownload|marketplaceResolve|persistRawSchema|persistPrompt|persistModelOutput|persistToolResult|RuntimeRecorder' @files 2>$null
if ($LASTEXITCODE -eq 0 -and $matches) { throw "[tool-schema-projection-readiness-gate] forbidden runtime/provider/persistence reference detected" }
Write-Host "[tool-schema-projection-readiness-gate] offline readiness contract passed"
