#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

echo '[mailbox-agent-pause-restore-contract] owner continuation and replay evidence'
go test ./orchestration/mailbox ./orchestration/planbridge ./tool/diagnosticsreplay -count=1

files=(
  orchestration/planbridge/capability.go
  orchestration/planbridge/compile.go
  orchestration/planbridge/execute.go
  orchestration/planbridge/types.go
  orchestration/planbridge/snapshot.go
)
pattern='mailbox\.NewContinuationWorker|mailbox\.New(File|Memory)Store\(|mailbox\.Consume\(|^[[:space:]]*go[[:space:]]+'
status=0
rg -n -- "$pattern" "${files[@]}" && status=0 || status=$?
if [[ "$status" != 1 ]]; then
  echo '[mailbox-agent-pause-restore-contract][ownership] planbridge created a mailbox worker/store/consume loop or scan failed' >&2
  exit 1
fi

echo '[mailbox-agent-pause-restore-contract] passed'
