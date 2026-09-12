Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[runtime-input-contract-gate] focused contract suites"
Invoke-NativeStrict -Label "runtime input suites" -Command {
    go test ./core/types ./core/runner ./host ./observability/event ./tool/diagnosticsreplay -run 'RuntimeInput|Host.*Input|RuntimeRecorderProjectsRuntimeInput|HostTranscript' -count=1
}

function Assert-Contains([string]$Path, [string]$Pattern, [string]$Label) {
    if (-not (Select-String -Path (Join-Path $repoRoot $Path) -Pattern $Pattern -Quiet)) {
        throw "[runtime-input-contract-gate][$Label] missing marker $Pattern in $Path"
    }
}

Assert-Contains "core\runner\runtime_input.go" "pendingSteering|followUpLimit|DrainRuntimeInputSafePoint" "bounded-lanes"
Assert-Contains "core\runner\control.go" "applyRuntimeInputSafePoint" "safe-point"
Assert-Contains "core\runner\runtime_input_parity_test.go" "RunStream|CancelAndTerminal" "parity"
Assert-Contains "tool\diagnosticsreplay\host_transcript.go" "embedded_host_protocol.v1" "replay"
Assert-Contains "host\host.go" "RuntimeInputControl" "source-ownership"

$forbidden = Get-Item (Join-Path $repoRoot "core\runner\runtime_input.go") | Select-String -Pattern 'provider\.|global.*queue|mcp/(http|stdio)|net/http'
if ($forbidden) {
    throw "[runtime-input-contract-gate][provider-injection] direct provider/global queue marker detected"
}
Write-Host "[runtime-input-contract-gate] passed"
