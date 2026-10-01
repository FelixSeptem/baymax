Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) { $env:GOCACHE = Join-Path $repoRoot ".gocache" }
$env:GOTELEMETRY = "off"

Write-Host "[sandbox-lifecycle-success-cost-evidence] running offline contract checks"
Invoke-NativeStrict -Label "sandbox lifecycle success cost evidence tests" -Command {
    go test ./tool/diagnosticsreplay ./tool/contributioncheck -run "SandboxLifecycleSuccessCostEvidence|SandboxLifecycleSuccessCost|QualityGateIncludesSandboxLifecycle" -count=1
}

$fixture = "tool/diagnosticsreplay/testdata/sandbox_lifecycle_success_cost_evidence.v1.json"
if (-not (Test-Path $fixture)) { throw "[sandbox-lifecycle-success-cost-evidence] missing fixture: $fixture" }
$raw = Get-Content -Raw $fixture
if ($raw.Length -gt 1MB) { throw "[sandbox-lifecycle-success-cost-evidence] fixture exceeds 1 MiB" }
foreach ($needle in @("endpoint", "sk-", "token", "raw_response", "raw_payload", "password", "secret", "command")) {
    if ($raw.ToLowerInvariant().Contains($needle)) { throw "[sandbox-lifecycle-success-cost-evidence] fixture contains forbidden material: $needle" }
}
$source = Get-Content -Raw "tool/diagnosticsreplay/sandbox_lifecycle_success_cost_evidence.go"
foreach ($needle in @("sandbox-exec", "bubblewrap", "net/http", "os/exec", "security.sandbox")) {
    if ($source.ToLowerInvariant().Contains($needle)) { throw "[sandbox-lifecycle-success-cost-evidence] forbidden execution coupling: $needle" }
}
Write-Host "[sandbox-lifecycle-success-cost-evidence] passed"
