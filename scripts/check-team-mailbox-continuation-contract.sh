#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

echo '[team-mailbox-continuation-contract] owner restore, lifecycle races, and owner boundaries'
go test ./orchestration/teams ./orchestration/mailbox ./orchestration/planbridge -count=1

files=(
  orchestration/mailbox/continuation.go
  orchestration/teams/continuation.go
)
pattern='^[[:space:]]*go[[:space:]]+|\b(NewMemoryStore|NewFileStore|Publish|Consume)\('
status=0
rg -n -- "$pattern" "${files[@]}" && status=0 || status=$?
if [[ "$status" != 1 ]]; then
  echo '[team-mailbox-continuation-contract][ownership] continuation path created an execution owner or scan failed' >&2
  exit 1
fi

echo '[team-mailbox-continuation-contract] passed'
