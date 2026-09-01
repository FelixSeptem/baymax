#!/usr/bin/env bash
set -euo pipefail

echo "[event-stream-terminal-recovery-gate] focused contract suites"
go test ./core/types ./core/runner ./runtime/diagnostics ./observability/event ./tool/diagnosticsreplay -run 'EventStreamTerminalRecovery' -count=1

echo "[event-stream-terminal-recovery-gate] library-first ownership assertions"
if rg -n 'http\.Listen|ListenAndServe|websocket|NewServer\(' core/types core/runner --glob '*.go'; then
  echo "event-stream recovery must not introduce a transport listener" >&2
  exit 1
fi
if rg -n 'global.*queue|recovery.*queue' core/types core/runner --glob '*event_stream_terminal_recovery*.go'; then
  echo "event-stream recovery must not own a global queue" >&2
  exit 1
fi
echo "[event-stream-terminal-recovery-gate] done"
