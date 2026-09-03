#!/usr/bin/env bash
set -euo pipefail

echo "[tool-lifecycle-failure-isolation-gate] focused contract suites"
go test ./core/types ./tool/local ./runtime/diagnostics ./tool/diagnosticsreplay -run 'ToolLifecycle|DispatcherProjectsToolLifecycle' -count=1

echo "[tool-lifecycle-failure-isolation-gate] library-first ownership assertions"
if rg -n 'http\.Listen|ListenAndServe|websocket|NewServer\(|global.*queue|external.*event.*store|terminal.*state.*machine' core/types/tool_lifecycle.go tool/local/lifecycle_projection.go tool/diagnosticsreplay/tool_lifecycle.go; then
  echo "tool lifecycle contract must remain library-first and reuse existing terminal ownership" >&2
  exit 1
fi
echo "[tool-lifecycle-failure-isolation-gate] done"
