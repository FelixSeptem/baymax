Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
& (Join-Path $PSScriptRoot "check-runtime-input-contract.ps1")
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
