Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}
if ($env:GODEBUG) {
    if ($env:GODEBUG -notmatch "(^|,)goindex=") {
        $env:GODEBUG = "$($env:GODEBUG),goindex=0"
    }
}
else {
    $env:GODEBUG = "goindex=0"
}

$mode = if ($env:BAYMAX_MEMORY_CONTRACT_MODE) { $env:BAYMAX_MEMORY_CONTRACT_MODE.Trim().ToLowerInvariant() } else { "smoke" }
if ($mode -ne "smoke" -and $mode -ne "full") {
    throw "[memory-contract] unsupported BAYMAX_MEMORY_CONTRACT_MODE=$mode; expected smoke|full"
}

function Invoke-MemoryStep {
    param(
        [Parameter(Mandatory = $true)][string]$Label,
        [Parameter(Mandatory = $true)][scriptblock]$Command
    )
    Write-Host "[memory-contract] $Label"
    [void](Invoke-NativeStrict -Label $Label -Command $Command)
}

Invoke-MemoryStep -Label "adapter manifest memory contract suites" -Command {
    go test ./adapter/manifest -run 'Test(ParseMemoryManifest|ActivateMemoryManifest)' -count=1
}

Invoke-MemoryStep -Label "memory conformance matrix suites" -Command {
    go test ./integration/adapterconformance -run '^TestMemoryAdapterConformance' -count=1
}

Invoke-MemoryStep -Label "runtime memory config/readiness suites" -Command {
    go test ./runtime/config -run 'Test(RuntimeMemoryConfig|ManagerRuntimeMemoryInvalidReloadRollsBack|ManagerReadinessPreflightMemory)' -count=1
}

Invoke-MemoryStep -Label "memory replay fixture suites" -Command {
    go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractPrimaryReasonArbitrationFixture|PrimaryReasonArbitrationReplayContract)' -count=1
}

if ($mode -eq "full") {
    Invoke-MemoryStep -Label "full memory adapter conformance package" -Command {
        go test ./integration/adapterconformance -count=1
    }
    Invoke-MemoryStep -Label "full diagnostics replay package" -Command {
        go test ./tool/diagnosticsreplay -count=1
    }
}

Write-Host "[memory-contract] done (mode=$mode)"
