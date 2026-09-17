Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) { $env:GOCACHE = Join-Path $repoRoot ".gocache" }
Write-Host "[external-extension-authoring] running external extension authoring conformance profile external_extension_authoring.v1"
Invoke-NativeStrict -Label "go test external extension authoring conformance" -Command {
    go test ./extension ./integration/extensionauthoring ./tool/diagnosticsreplay -run 'Test(DecodeAuthoring|Authoring|EvaluateAuthoring|ReplayContractExternalExtensionAuthoring)' -count=1
}
Write-Host "[external-extension-authoring] passed profile=external_extension_authoring.v1"
