Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[native-execution-event-contract] projection, observer, integration and offline replay"
Invoke-NativeStrict -Label "native execution suites" -Command {
    go test ./core/types ./host/eventadapter ./integration ./tool/diagnosticsreplay ./tool/contributioncheck -run 'TestNativeExecution|TestObserver' -count=1
}

# Scan implementation code, not comments or fixture/test assertions. These
# three files may not introduce execution, transport, persistence or workers.
$files = @("core/types/native_execution_event.go", "host/eventadapter/observer.go", "tool/diagnosticsreplay/native_execution_event.go")
$pattern = '"(net/|database/|os"|github.com/FelixSeptem/baymax/(core/runner|runtime/|observability/|orchestration/|model/|mcp/))|^[\t ]*go[\t ]+|make\([\t ]*(chan|map\[)|\.(Run|Stream|Resume|Cancel|Retry|Record|AppendEvent)\('
# Pure replay's local bounded maps are allowed; observer/projector caches are not.
foreach ($file in $files) {
    $effective = if ($file -like "tool/*") { $pattern.Replace('make\([\t ]*(chan|map\[)', 'make\([\t ]*chan') } else { $pattern }
    $matches = @(rg -n -- $effective $file)
    if ($LASTEXITCODE -eq 0) { throw "[native-execution-event-contract][ownership] $($matches -join '; ')" }
    if ($LASTEXITCODE -ne 1) { throw "[native-execution-event-contract] ownership scan failed" }
}

$smoke = @()
Invoke-NativeStrict -Label "native execution example" -Command { $script:smoke = @(go run ./examples/native-execution-events) }
foreach ($marker in @("native-events run=completed", "native-events stream=completed", "observer=healthy", "pause-is-terminal=false")) {
    if ($smoke -notcontains $marker) { throw "[native-execution-event-contract] missing marker $marker" }
}
Write-Host "[native-execution-event-contract] passed"
