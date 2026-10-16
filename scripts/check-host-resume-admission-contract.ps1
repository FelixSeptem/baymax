Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
Set-Location (Resolve-Path (Join-Path $PSScriptRoot ".."))
Write-Host "[host-resume-admission-contract] native admission, concurrency, Host, replay and example"
Invoke-NativeStrict -Label "host resume suites" -Command {
    go test ./core/types ./core/runner ./host ./integration ./tool/diagnosticsreplay ./tool/contributioncheck ./examples/host-resume-admission -run 'TestHostResume|TestDynamicAction|TestConnectionAdmitsDynamicAction|TestConnectionRejectsDynamicAction' -count=1 -timeout 2m
}
$priorFixture = [Environment]::GetEnvironmentVariable("BAYMAX_HOST_RESUME_ADMISSION_FIXTURE")
try {
    $env:BAYMAX_HOST_RESUME_ADMISSION_FIXTURE = "testdata/host_resume_admission.invalid.json"
    $invalidOutput = @(go test ./tool/diagnosticsreplay -run TestHostResumeReplayGoldenAndDrift -count=1 2>&1)
    $invalidExit = $LASTEXITCODE
    if ($invalidExit -eq 0 -or ($invalidOutput -join "`n") -notmatch 'host_resume.replay_drift') {
        throw "[host-resume-admission-contract] invalid fixture lacks host_resume.replay_drift failure"
    }
    Write-Host "[host-resume-admission-contract] expected drift exit=$invalidExit classification=host_resume.replay_drift"
} finally {
    [Environment]::SetEnvironmentVariable("BAYMAX_HOST_RESUME_ADMISSION_FIXTURE", $priorFixture)
}
$smoke = @()
Invoke-NativeStrict -Label "host resume example" -Command { $script:smoke = @(go run ./examples/host-resume-admission) }
foreach ($marker in @("authority_denied", "resume_accepted", "completed", "revision_drift", "duplicate_no_effect", "resume_in_progress", "effect_cas_rejected", "run_stream_parity")) {
    if ($smoke -notcontains $marker) { throw "[host-resume-admission-contract] missing marker $marker" }
}
Write-Host "[host-resume-admission-contract] passed"
