#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
echo '[native-event-history-contract] reference-only replay, cursor, retention and ownership'
go test ./core/types ./orchestration/mailbox ./orchestration/planbridge ./host/eventadapter ./tool/diagnosticsreplay ./integration -run '^TestNativeHistory|^TestMailboxHistory|^TestFileMailboxHistory|^TestCallbackAdapterNeverOwnsHistory|^TestRetainedFactCanReachSink' -count=1
file=tool/diagnosticsreplay/native_event_history.go
pattern='"(net/|database/|os"|github.com/FelixSeptem/baymax/(core/runner|runtime/|observability/|orchestration/|model/|mcp/))|^[[:space:]]*go[[:space:]]+|\.(Run|Stream|Resume|Cancel|Retry|Record|AppendEvent|ReadAfter)\('
status=0
rg -n -- "$pattern" "$file" && status=0 || status=$?
if [[ "$status" != 1 ]]; then
  echo '[native-event-history-contract][ownership] runtime/source-owner dependency or scan failure' >&2
  exit 1
fi
echo '[native-event-history-contract] passed'
