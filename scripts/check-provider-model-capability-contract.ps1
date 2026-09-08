Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) { $env:GOCACHE = Join-Path $repoRoot ".gocache" }
$env:GOSUMDB = "sum.golang.org"
$env:GOTELEMETRY = "off"

Write-Host "[provider-model] running admission contract checks"
Invoke-NativeStrict -Label "provider catalog/config/diagnostics tests" -Command {
    go test ./model/catalog ./runtime/config ./runtime/diagnostics ./observability/event ./orchestration/composer ./tool/diagnosticsreplay -run 'ProviderCatalog|ProviderModel|ProviderAdmission|RunRecordProviderAdmission|RuntimeRecorderProjectsRedacted' -count=1
}

$forbidden = @('remote discovery', 'credential material', 'background refresh', 'global mutable registry')
foreach ($needle in $forbidden) {
    $matches = @(rg -n -i --glob '*.go' $needle model/catalog runtime/config orchestration/composer 2>$null)
    if ($matches.Count -gt 0) {
        throw "[provider-model] forbidden admission coupling detected: $needle`n$($matches -join "`n")"
    }
}

$contextSDK = @(rg -n 'github.com/.*/(anthropic|openai|gemini|genai)' context 2>$null | Where-Object { $_ -notmatch 'context[\\/]assembler[\\/]embedding_adapter.go' })
if ($contextSDK.Count -gt 0) { throw "[provider-model] Provider SDK import found under context/*`n$($contextSDK -join "`n")" }

Write-Host "[provider-model] passed"
