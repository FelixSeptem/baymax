$ErrorActionPreference = "Stop"
Write-Host "[event-stream-terminal-recovery-gate] focused contract suites"
go test ./core/types ./core/runner ./runtime/diagnostics ./observability/event ./tool/diagnosticsreplay -run 'EventStreamTerminalRecovery' -count=1

Write-Host "[event-stream-terminal-recovery-gate] library-first ownership assertions"
$forbidden = Get-ChildItem core/types,core/runner -Recurse -Filter *.go | Select-String -Pattern 'http\.Listen|ListenAndServe|websocket|NewServer\('
if ($forbidden) { throw "event-stream recovery must not introduce a transport listener" }
$queue = Get-ChildItem core/types,core/runner -Recurse -Filter *.go | Select-String -Pattern 'global.*queue|recovery.*queue' | Where-Object { $_.Path -match 'event_stream_terminal_recovery' }
if ($queue) { throw "event-stream recovery must not own a global queue" }
Write-Host "[event-stream-terminal-recovery-gate] done"
