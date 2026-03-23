Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

if ([string]::IsNullOrWhiteSpace($env:GOCACHE)) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}
if (-not (Test-Path $env:GOCACHE)) {
    New-Item -ItemType Directory -Path $env:GOCACHE | Out-Null
}

function Invoke-GoTest {
    param([Parameter(ValueFromRemainingArguments = $true)][string[]]$GoArgs)
    Invoke-NativeStrict -Label ("go test " + ($GoArgs -join " ")) -Command {
        & go test @GoArgs
    }
}

Write-Host "[canonical-mailbox-entrypoints] contributioncheck"
Invoke-GoTest "./tool/contributioncheck" "-run" "^TestCanonicalMailboxInvokeEntrypoints$" "-count=1"

Write-Host "[canonical-mailbox-entrypoints] sync invocation canonical suite"
Invoke-GoTest "./integration" "-run" "^TestSyncInvocationContractCanonicalMailboxOnlyPublicEntrypoints$" "-count=1"

Write-Host "[canonical-mailbox-entrypoints] async invocation canonical suite"
Invoke-GoTest "./integration" "-run" "^TestAsyncReportingContractLegacyDirectAsyncEntrypointNotSupportedPublicly$" "-count=1"

Write-Host "[canonical-mailbox-entrypoints] mailbox convergence canonical suite"
Invoke-GoTest "./integration" "-run" "^TestMailboxContractCanonicalEntrypointConvergenceGuard$" "-count=1"
