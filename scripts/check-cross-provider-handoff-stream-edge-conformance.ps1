Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
& (Join-Path $PSScriptRoot "check-provider-handoff-stream-edge-contract.ps1")
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
