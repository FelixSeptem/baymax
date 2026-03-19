Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}

Write-Host "[example-smoke] running full-chain example"
$output = & go run ./examples/09-multi-agent-full-chain-reference 2>&1
if ($LASTEXITCODE -ne 0) {
    $output | ForEach-Object { Write-Host $_ }
    throw "[example-smoke] full-chain example execution failed"
}
$output | ForEach-Object { Write-Host $_ }

$requiredMarkers = @(
    "CHECKPOINT async_report_succeeded=true",
    "CHECKPOINT delayed_dispatch_claimed=true",
    "CHECKPOINT recovery_replayed=true",
    "CHECKPOINT correlation ",
    "CHECKPOINT run_stream_aligned=true",
    "A20_RUN_TERMINAL",
    "A20_STREAM_TERMINAL",
    "A20_TERMINAL_SUMMARY=",
    "A20_SUCCESS"
)

foreach ($marker in $requiredMarkers) {
    if (-not ($output | Where-Object { $_ -like "*$marker*" } | Select-Object -First 1)) {
        throw "[example-smoke] missing required marker: $marker"
    }
}

Write-Host "[example-smoke] passed"
