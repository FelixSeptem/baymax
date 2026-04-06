Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}

Write-Host "[example-smoke] running full-chain example"
$output = Invoke-NativeCaptureStrict -Label "go run ./examples/09-multi-agent-full-chain-reference" -Command {
    go run ./examples/09-multi-agent-full-chain-reference 2>&1
}
$output | ForEach-Object { Write-Host $_ }

$requiredMarkers = @(
    "CHECKPOINT async_report_succeeded=true",
    "CHECKPOINT delayed_dispatch_claimed=true",
    "CHECKPOINT recovery_replayed=true",
    "CHECKPOINT correlation ",
    "CHECKPOINT run_stream_aligned=true"
)

foreach ($marker in $requiredMarkers) {
    if (-not ($output | Where-Object { $_ -like "*$marker*" } | Select-Object -First 1)) {
        throw "[example-smoke] missing required marker: $marker"
    }
}

function Assert-Marker {
    param(
        [Parameter(Mandatory = $true)][string]$Marker
    )
    if ($output | Where-Object { $_ -like "*$Marker*" } | Select-Object -First 1) {
        return
    }
    throw "[example-smoke] missing required marker: $Marker"
}

Assert-Marker -Marker "FULL_CHAIN_RUN_TERMINAL"
Assert-Marker -Marker "FULL_CHAIN_STREAM_TERMINAL"
Assert-Marker -Marker "FULL_CHAIN_TERMINAL_SUMMARY="
Assert-Marker -Marker "FULL_CHAIN_SUCCESS"

Write-Host "[example-smoke] passed"
