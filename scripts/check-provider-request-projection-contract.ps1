Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) { $env:GOCACHE = Join-Path $repoRoot ".gocache" }

$fixture = "tool/diagnosticsreplay/testdata/model_request_projection.v1.json"
$fixtureVersion = "provider_request_projection.v1"
$contractFile = "model/conformance/request_projection.go"
$taxonomyDoc = "model/README.md"

Write-Host "[provider-request-projection] fixture presence, version, and bounds"
if (-not (Test-Path -LiteralPath $fixture)) { throw "provider_request_schema_drift: missing fixture $fixture" }
if ((Get-Item -LiteralPath $fixture).Length -gt 2097152) { throw "provider_request_overflow_drift: fixture exceeds 2 MiB" }
$fixtureText = Get-Content -LiteralPath $fixture -Raw
if ($fixtureText -notmatch [regex]::Escape($fixtureVersion)) { throw "provider_request_schema_drift: fixture does not declare $fixtureVersion" }
if ($fixtureText -match 'tool_result_envelope') { throw "provider_request_tool_result_native_drift: fixture still declares text-envelope projection" }
foreach ($resolvedGap in @("provider_request_role_projection_drift", "provider_request_tool_result_native_drift", "provider_request_part_ordering_drift", "provider_request_run_stream_parity_drift")) {
    if ($fixtureText -match [regex]::Escape($resolvedGap)) {
        throw "provider_request_contract_drift: resolved gap remains declared in fixture: $resolvedGap"
    }
}

Write-Host "[provider-request-projection] stable classification taxonomy"
$requestProjectionCodes = @(
    "provider_request_schema_drift",
    "provider_request_role_projection_drift",
    "provider_request_tool_result_native_drift",
    "provider_request_part_ordering_drift",
    "provider_request_stable_prefix_drift",
    "provider_request_tool_order_drift",
    "provider_request_capability_projection_drift",
    "provider_request_run_stream_parity_drift",
    "provider_cache_usage_projection_drift",
    "provider_request_overflow_drift",
    "provider_request_contract_drift"
)
$contractText = Get-Content -LiteralPath $contractFile -Raw
$taxonomyText = Get-Content -LiteralPath $taxonomyDoc -Raw
foreach ($code in $requestProjectionCodes) {
    if ($contractText -notmatch [regex]::Escape('"' + $code + '"')) {
        throw "provider_request_contract_drift: $code missing from $contractFile"
    }
    if ($taxonomyText -notmatch [regex]::Escape($code)) {
        throw "provider_request_contract_drift: $code missing from $taxonomyDoc"
    }
}

Write-Host "[provider-request-projection] adapter ownership, provider neutrality, no raw payload"
Invoke-NativeStrict -Label "request projection contract boundary" -Command { go test ./tool/contributioncheck -run 'TestProviderRequestProjectionContractBoundary' -count=1 }

Write-Host "[provider-request-projection] adapter SDK request projection shape"
Invoke-NativeStrict -Label "request projection adapter shape" -Command { go test ./model/conformance ./model/openai ./model/anthropic ./model/gemini -run 'RequestProjection|CacheUsage|ProjectionAudit|NativeMessageParams|NativeGenerateRequest|CountTokens' -count=1 }

Write-Host "[provider-request-projection] offline replay idempotency"
Invoke-NativeStrict -Label "request projection replay idempotency" -Command { go test ./tool/diagnosticsreplay -run 'ProviderRequestProjection' -count=2 }

Write-Host "[provider-request-projection-contract] passed"
