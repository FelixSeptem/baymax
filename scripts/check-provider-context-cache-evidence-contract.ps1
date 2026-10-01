Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
$cache = if ($env:GOCACHE) { $env:GOCACHE } else { Join-Path $repoRoot ".gocache-provider-context-cache-evidence-gate" }
New-Item -ItemType Directory -Force -Path $cache | Out-Null
$env:GOCACHE = $cache

go test ./model/conformance ./tool/diagnosticsreplay ./tool/contributioncheck -run 'ProviderContextCache|AdmitProviderContextCache' -count=1

$fixture = Join-Path $repoRoot "tool/diagnosticsreplay/testdata/provider_context_cache_evidence.v1.json"
if (-not (Test-Path -LiteralPath $fixture)) { throw "[provider-context-cache-evidence-gate] missing fixture $fixture" }

$forbidden = rg -n 'github\.com/(openai|anthropics)|google\.golang\.org/genai|net/http|RuntimeRecorder|ModelRequest|raw_prompt|raw_reasoning' (Join-Path $repoRoot "model/conformance/provider_context_cache_evidence.go") (Join-Path $repoRoot "tool/diagnosticsreplay/provider_context_cache_evidence.go") 2>$null
if ($LASTEXITCODE -eq 0 -and $forbidden) { throw "[provider-context-cache-evidence-gate] forbidden provider/runtime/raw payload reference detected`n$forbidden" }

Write-Host "[provider-context-cache-evidence-gate] offline evidence, privacy, replay, and review-only route passed"
