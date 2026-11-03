Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
Set-Location (Resolve-Path (Join-Path $PSScriptRoot ".."))
Write-Host "[attention-diagnostics-contract] bounded pure snapshot and replay"
Invoke-NativeStrict -Label "attention suites" -Command {
    go test ./runtime/diagnostics ./integration ./tool/diagnosticsreplay ./tool/contributioncheck ./examples/attention-diagnostics -run TestAttention -count=1 -timeout 2m
}
$priorFixture = [Environment]::GetEnvironmentVariable("BAYMAX_ATTENTION_FIXTURE")
try {
    $env:BAYMAX_ATTENTION_FIXTURE = "testdata/attention.invalid.json"
    $invalidOutput = @(go test ./tool/diagnosticsreplay -run TestAttentionReplayGoldenAndDrift -count=1 2>&1)
    $invalidExit = $LASTEXITCODE
    if ($invalidExit -eq 0 -or ($invalidOutput -join "`n") -notmatch 'attention.replay_drift') {
        throw "[attention-diagnostics-contract] invalid fixture lacks attention.replay_drift failure"
    }
    Write-Host "[attention-diagnostics-contract] expected drift exit=$invalidExit classification=attention.replay_drift"
} finally {
    [Environment]::SetEnvironmentVariable("BAYMAX_ATTENTION_FIXTURE", $priorFixture)
}
Invoke-NativeStrict -Label "attention example" -Command { $script:smoke = @(go run ./examples/attention-diagnostics) }
foreach ($marker in @("healthy_retrying_ignored", "effect_pending_explicit", "truncation_visible")) {
    if ($smoke -notcontains $marker) { throw "[attention-diagnostics-contract] missing marker $marker" }
}
Write-Host "[attention-diagnostics-contract] passed"
