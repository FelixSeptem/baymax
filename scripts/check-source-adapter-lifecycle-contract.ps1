Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
Set-Location (Resolve-Path (Join-Path $PSScriptRoot ".."))
Write-Host "[source-adapter-lifecycle-contract] bounded delivery, durable state and pure replay"
Invoke-NativeStrict -Label "source adapter suites" -Command {
    go test ./host/sourceadapter ./integration ./tool/diagnosticsreplay ./tool/contributioncheck ./examples/source-adapter-lifecycle -run 'TestSourceAdapter|TestDisposition|TestWholePage|TestPending|TestPoll|TestAbort|TestOperation|TestHostEffect|TestBounds|TestRecent|TestFileStore' -count=1 -timeout 2m
}
$priorFixture = [Environment]::GetEnvironmentVariable("BAYMAX_SOURCE_ADAPTER_FIXTURE")
try {
    $env:BAYMAX_SOURCE_ADAPTER_FIXTURE = "testdata/source_adapter_lifecycle.invalid.json"
    $invalidOutput = @(go test ./tool/diagnosticsreplay -run TestSourceAdapterReplayGoldenAndDrift -count=1 2>&1)
    $invalidExit = $LASTEXITCODE
    if ($invalidExit -eq 0 -or ($invalidOutput -join "`n") -notmatch 'source_adapter.replay_drift') {
        throw "[source-adapter-lifecycle-contract] invalid fixture lacks source_adapter.replay_drift failure"
    }
    Write-Host "[source-adapter-lifecycle-contract] expected drift exit=$invalidExit classification=source_adapter.replay_drift"
} finally {
    [Environment]::SetEnvironmentVariable("BAYMAX_SOURCE_ADAPTER_FIXTURE", $priorFixture)
}
Invoke-NativeStrict -Label "source adapter example" -Command { $script:smoke = @(go run ./examples/source-adapter-lifecycle) }
foreach ($marker in @("sent_confirmed", "gone_confirmed", "retryable_preserved", "duplicate_no_effect", "restart_confirmed_cursor", "abort_preserved")) {
    if ($smoke -notcontains $marker) { throw "[source-adapter-lifecycle-contract] missing marker $marker" }
}
Write-Host "[source-adapter-lifecycle-contract] passed"
