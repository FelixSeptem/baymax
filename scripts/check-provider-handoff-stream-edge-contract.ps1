Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) { $env:GOCACHE = Join-Path $repoRoot ".gocache" }
$fixture = "tool/diagnosticsreplay/testdata/provider_handoff_stream_edge.v1.json"
if (-not (Test-Path -LiteralPath $fixture)) { throw "provider_handoff_schema_drift: missing fixture" }
if ((Get-Item -LiteralPath $fixture).Length -gt 2097152) { throw "provider_overflow_drift: fixture exceeds 2 MiB" }
Invoke-NativeStrict -Label "provider handoff stream edge replay" -Command { go test ./tool/diagnosticsreplay -run 'ProviderHandoffStreamEdge' -count=2 }
Write-Host "[provider-handoff-stream-edge-contract] passed"
