Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) { $env:GOCACHE = Join-Path $repoRoot ".gocache" }
$env:GOSUMDB = "sum.golang.org"
$env:GOPROXY = "https://proxy.golang.org,direct"
Write-Host "[extension-lifecycle-replay] running deterministic extension lifecycle checks"
Invoke-NativeStrict -Label "go test ./integration/extensionlifecycle -run '^TestReplayContract' -count=1" -Command {
    go test ./integration/extensionlifecycle -run '^TestReplayContract' -count=1
}
Write-Host "[extension-lifecycle-replay] passed"
