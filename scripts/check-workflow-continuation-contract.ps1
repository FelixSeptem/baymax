Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[workflow-continuation-contract] source restore, unsupported forms, and read-only replay"
Invoke-NativeStrict -Label "workflow continuation contract suites" -Command {
    go test ./orchestration/workflow ./orchestration/planbridge ./tool/diagnosticsreplay -count=1
}

$file = "tool/diagnosticsreplay/workflow_continuation.go"
$pattern = '"(net/|database/|os"|github.com/FelixSeptem/baymax/(orchestration/workflow|core/checkpoint|core/runner|runtime/|observability/|model/|mcp/))|^[\t ]*go[\t ]+|\.(Run|Stream|Resume|ClaimResume|MarkResumeRunning|CommitResumeResult|Save|Load)\('
$matches = @(rg -n -- $pattern $file)
if ($LASTEXITCODE -eq 0) { throw "[workflow-continuation-contract][ownership] $($matches -join '; ')" }
if ($LASTEXITCODE -ne 1) { throw "[workflow-continuation-contract] ownership scan failed" }

$priorFixture = $env:BAYMAX_WORKFLOW_CONTINUATION_FIXTURE
try {
    $env:BAYMAX_WORKFLOW_CONTINUATION_FIXTURE = "testdata/workflow_continuation.invalid.json"
    $invalidExit = Invoke-NativeStrict -Label "invalid workflow continuation replay fixture" -AllowFailure -Command {
        go test ./tool/diagnosticsreplay -run '^TestWorkflowContinuationReplayGoldenAndDrift$' -count=1
    }
    if ($invalidExit -eq 0) { throw "[workflow-continuation-contract] invalid fixture unexpectedly passed" }
} finally {
    if ($null -eq $priorFixture) { Remove-Item Env:BAYMAX_WORKFLOW_CONTINUATION_FIXTURE -ErrorAction SilentlyContinue }
    else { $env:BAYMAX_WORKFLOW_CONTINUATION_FIXTURE = $priorFixture }
}
Write-Host "[workflow-continuation-contract] passed"
