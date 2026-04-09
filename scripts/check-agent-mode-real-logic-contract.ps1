Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

pwsh -File (Join-Path $PSScriptRoot "check-agent-mode-real-runtime-semantic-contract.ps1")