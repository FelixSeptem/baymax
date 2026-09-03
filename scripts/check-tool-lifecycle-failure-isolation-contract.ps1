$ErrorActionPreference = "Stop"
Write-Host "[tool-lifecycle-failure-isolation-gate] focused contract suites"
go test ./core/types ./tool/local ./runtime/diagnostics ./tool/diagnosticsreplay -run 'ToolLifecycle|DispatcherProjectsToolLifecycle' -count=1

Write-Host "[tool-lifecycle-failure-isolation-gate] library-first ownership assertions"
$forbidden = Get-ChildItem core/types/tool_lifecycle.go,tool/local/lifecycle_projection.go,tool/diagnosticsreplay/tool_lifecycle.go -ErrorAction SilentlyContinue | Select-String -Pattern 'http\.Listen|ListenAndServe|websocket|NewServer\(|global.*queue|external.*event.*store|terminal.*state.*machine'
if ($forbidden) { throw "tool lifecycle contract must remain library-first and reuse existing terminal ownership" }
Write-Host "[tool-lifecycle-failure-isolation-gate] done"
