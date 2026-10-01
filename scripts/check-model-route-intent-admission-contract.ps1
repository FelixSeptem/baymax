Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) { $env:GOCACHE = Join-Path $repoRoot ".gocache" }
$env:GOTELEMETRY = "off"

Write-Host "[model-route-intent-admission] running contract checks"
Invoke-NativeStrict -Label "route intent tests" -Command {
    go test ./model/catalog ./tool/diagnosticsreplay ./tool/contributioncheck -run 'ModelRouteIntentAdmission|RouteIntentAdmission' -count=1
}

$fixture = "tool/diagnosticsreplay/testdata/model_route_intent_admission.v1.json"
if (-not (Test-Path $fixture)) { throw "[model-route-intent-admission] missing fixture: $fixture" }
$raw = Get-Content -Raw $fixture
if ($raw.Length -gt 2MB) { throw "[model-route-intent-admission] fixture exceeds 2 MiB" }
foreach ($needle in @("endpoint", "sk-", "token", "raw_response", "raw_payload", "password", "secret")) {
    if ($raw.ToLowerInvariant().Contains($needle)) { throw "[model-route-intent-admission] fixture contains forbidden material: $needle" }
}
foreach ($needle in @("github.com/openai/openai-go", "github.com/anthropics/anthropic-sdk-go", "google.golang.org/genai", "net/http")) {
    $matches = @(rg -n --glob '*.go' $needle model/catalog tool/diagnosticsreplay 2>$null)
    if ($matches.Count -gt 0) { throw "[model-route-intent-admission] forbidden coupling: $needle`n$($matches -join "`n")" }
}
Write-Host "[model-route-intent-admission] passed"
