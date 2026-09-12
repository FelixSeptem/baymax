Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[embedded-host-contract-gate] focused contract suites"
Invoke-NativeStrict -Label "host contract suites" -Command {
    go test ./core/types ./core/runner ./host/... ./observability/event ./tool/diagnosticsreplay -run 'Host|Realtime|RuntimeRecorder' -count=1
}

function Assert-PatternAbsent {
    param([Parameter(Mandatory = $true)][string]$Assertion, [Parameter(Mandatory = $true)][string]$Pattern)
    $rg = Get-Command rg -ErrorAction Stop
    $matches = @(& $rg.Source -n --glob 'host/**/*.go' --glob 'host/*.go' --glob '!host/**/*_test.go' --glob '!host/*_test.go' -- $Pattern . 2>$null)
    if ($LASTEXITCODE -eq 0) {
        throw "[embedded-host-contract-gate][$Assertion] forbidden host match: $($matches -join '; ')"
    }
    if ($LASTEXITCODE -gt 1) {
        throw "[embedded-host-contract-gate][$Assertion] rg scan failed with exit $LASTEXITCODE"
    }
}

Write-Host "[embedded-host-contract-gate] ownership and non-goal assertions"
Assert-PatternAbsent -Assertion "source_owner" -Pattern 'runtime/diagnostics|mcp/(http|stdio)|net/http|gorilla/websocket|database/sql'
Assert-PatternAbsent -Assertion "non_goals" -Pattern 'steering|follow[-_ ]?up|hosted listener|remote Session|Artifact store|global queue|terminal state machine'
Assert-PatternAbsent -Assertion "bounded_state" -Pattern '^[[:space:]]*var[[:space:]]+[A-Za-z0-9_]+[[:space:]]*=[[:space:]]*make\((map|chan)'

Write-Host "[embedded-host-contract-gate] replay and framing markers"
# embedded_host_protocol.v1 is the versioned replay contract marker.
if (-not (Select-String -Path (Join-Path $repoRoot "tool\diagnosticsreplay\host_transcript*") -Pattern "embedded_host_protocol.v1" -Quiet)) {
    throw "[embedded-host-contract-gate][replay] missing embedded_host_protocol.v1 marker"
}
if (-not (Select-String -Path (Join-Path $repoRoot "host\jsonl\*.go") -Pattern "LF|CRLF|Frame|Negotiat|MaxFrame" -Quiet)) {
    throw "[embedded-host-contract-gate][framing] missing framing markers"
}
Write-Host "[embedded-host-contract-gate] done"
