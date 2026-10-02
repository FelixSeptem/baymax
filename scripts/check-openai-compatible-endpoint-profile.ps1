Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) { $env:GOCACHE = Join-Path $repoRoot ".gocache-openai-compatible-profile" }
$fixture = "tool/diagnosticsreplay/testdata/openai_endpoint_profile_conformance.v1.json"
# ReplayOpenAIEndpointProfileFixtureJSON is exercised by the package tests below.
if (-not (Test-Path -LiteralPath $fixture)) { throw "openai_endpoint_profile_schema_drift: missing fixture $fixture" }
if ((Get-Item -LiteralPath $fixture).Length -gt 1048576) { throw "openai_endpoint_profile_overflow_drift: fixture exceeds 1 MiB" }
$fixtureText = Get-Content -LiteralPath $fixture -Raw
if ($fixtureText -notmatch [regex]::Escape('"openai-compatible-endpoint-profile-conformance.v1"')) { throw "openai_endpoint_profile_schema_drift: unsupported fixture version" }
if ($fixtureText -match '(?i)raw_prompt|raw_reasoning|raw_payload|credential|password|bearer token') { throw "openai_endpoint_profile_privacy_drift: sensitive fixture marker" }
Invoke-NativeStrict -Label "openai compatible endpoint profile offline gate" -Command { go test ./model/conformance ./model/openai ./tool/diagnosticsreplay ./tool/contributioncheck -run 'OpenAIEndpointProfile|EndpointProfile|Profile' -count=1 }
Write-Host "[openai-compatible-endpoint-profile] offline fixture, capability, parity, privacy, and review-only gate passed (no network)"
