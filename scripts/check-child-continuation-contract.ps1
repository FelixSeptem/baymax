Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[child-continuation-contract] source-owned child restore and read-only replay"
Invoke-NativeStrict -Label "child continuation contract suites" -Command {
    go test ./orchestration/composer ./orchestration/planbridge ./tool/diagnosticsreplay -count=1
}
$file = "tool/diagnosticsreplay/child_continuation.go"
$pattern = '"(net/|database/|os"|github.com/FelixSeptem/baymax/(orchestration/|core/runner|runtime/|observability/))|^[\t ]*go[\t ]+|\.(Run|Stream|Resume|SpawnChild|Claim|Save|Load|Commit)\('
$matches = @(rg -n -- $pattern $file)
if ($LASTEXITCODE -eq 0) { throw "[child-continuation-contract][ownership] $($matches -join '; ')" }
if ($LASTEXITCODE -ne 1) { throw "[child-continuation-contract] ownership scan failed" }
Write-Host "[child-continuation-contract] passed"
