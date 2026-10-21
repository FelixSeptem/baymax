#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
echo '[child-continuation-contract] source-owned child restore and read-only replay'
go test ./orchestration/composer ./orchestration/planbridge ./tool/diagnosticsreplay -count=1
file=tool/diagnosticsreplay/child_continuation.go
pattern='"(net/|database/|os"|github.com/FelixSeptem/baymax/(orchestration/|core/runner|runtime/|observability/))|^[[:space:]]*go[[:space:]]+|\.(Run|Stream|Resume|SpawnChild|Claim|Save|Load|Commit)\('
status=0
rg -n -- "$pattern" "$file" && status=0 || status=$?
if [[ "$status" != 1 ]]; then
  echo '[child-continuation-contract][ownership] runtime/source-owner dependency or scan failure' >&2
  exit 1
fi
echo '[child-continuation-contract] passed'
