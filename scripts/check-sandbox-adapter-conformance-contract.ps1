Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}

Write-Host "[sandbox-adapter-gate] sandbox manifest profile-pack contracts"
Invoke-NativeStrict -Label "go test ./adapter/manifest -run 'Test(ParseSandboxManifest|ActivateSandboxManifest|SandboxProfilePack)' -count=1" -Command {
    go test ./adapter/manifest -run 'Test(ParseSandboxManifest|ActivateSandboxManifest|SandboxProfilePack)' -count=1
}

Write-Host "[sandbox-adapter-gate] external adapter conformance backend/session/capability matrix"
Invoke-NativeStrict -Label "go test ./integration/adapterconformance -run 'TestSandboxAdapterConformance' -count=1" -Command {
    go test ./integration/adapterconformance -run 'TestSandboxAdapterConformance' -count=1
}

Write-Host "[sandbox-adapter-gate] runtime readiness sandbox adapter findings"
Invoke-NativeStrict -Label "go test ./runtime/config -run 'TestManagerReadinessPreflightSandboxAdapter' -count=1" -Command {
    go test ./runtime/config -run 'TestManagerReadinessPreflightSandboxAdapter' -count=1
}

Write-Host "[sandbox-adapter-gate] adapter contract replay sandbox.v1 + mixed tracks"
Invoke-NativeStrict -Label "go test ./integration/adaptercontractreplay -run 'TestReplayContract(SandboxProfilePackTrack|MixedTracksBackwardCompatible|ProfileVersionValidation)' -count=1" -Command {
    go test ./integration/adaptercontractreplay -run 'TestReplayContract(SandboxProfilePackTrack|MixedTracksBackwardCompatible|ProfileVersionValidation)' -count=1
}

