Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[native-continuation-contract] source evidence, unsupported boundaries and read-only crash-window replay"
Invoke-NativeStrict -Label "native continuation contract suites" -Command {
    go test ./core/types ./core/checkpoint ./core/runner ./orchestration/planbridge ./tool/diagnosticsreplay -count=1
}

$file = "tool/diagnosticsreplay/native_continuation.go"
$pattern = '"(net/|database/|os"|github.com/FelixSeptem/baymax/(core/runner|core/checkpoint|runtime/|observability/|orchestration/|model/|mcp/))|^[\t ]*go[\t ]+|\.(Run|Stream|Resume|ResumeFromCheckpoint|Cancel|Retry|Record|AppendEvent|Save|Load|ClaimResume|CommitResumeResult)\('
$matches = @(rg -n -- $pattern $file)
if ($LASTEXITCODE -eq 0) { throw "[native-continuation-contract][ownership] $($matches -join '; ')" }
if ($LASTEXITCODE -ne 1) { throw "[native-continuation-contract] ownership scan failed" }

$priorFixture = $env:BAYMAX_NATIVE_CONTINUATION_FIXTURE
try {
    $env:BAYMAX_NATIVE_CONTINUATION_FIXTURE = "testdata/native_continuation.invalid.json"
    $invalidExit = Invoke-NativeStrict -Label "invalid continuation replay fixture" -AllowFailure -Command {
        go test ./tool/diagnosticsreplay -run '^TestNativeContinuationReplayGoldenAndDrift$' -count=1
    }
    if ($invalidExit -eq 0) { throw "[native-continuation-contract] invented terminal fixture unexpectedly passed" }
} finally {
    if ($null -eq $priorFixture) { Remove-Item Env:BAYMAX_NATIVE_CONTINUATION_FIXTURE -ErrorAction SilentlyContinue }
    else { $env:BAYMAX_NATIVE_CONTINUATION_FIXTURE = $priorFixture }
}
Write-Host "[native-continuation-contract] passed"
