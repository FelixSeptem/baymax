#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
echo '[native-continuation-contract] source evidence, unsupported boundaries and read-only crash-window replay'
go test ./core/types ./core/checkpoint ./core/runner ./orchestration/planbridge ./tool/diagnosticsreplay -count=1
file=tool/diagnosticsreplay/native_continuation.go
pattern='"(net/|database/|os"|github.com/FelixSeptem/baymax/(core/runner|core/checkpoint|runtime/|observability/|orchestration/|model/|mcp/))|^[[:space:]]*go[[:space:]]+|\.(Run|Stream|Resume|ResumeFromCheckpoint|Cancel|Retry|Record|AppendEvent|Save|Load|ClaimResume|CommitResumeResult)\('
status=0
rg -n -- "$pattern" "$file" && status=0 || status=$?
if [[ "$status" != 1 ]]; then
  echo '[native-continuation-contract][ownership] runtime/source-owner dependency or scan failure' >&2
  exit 1
fi
if BAYMAX_NATIVE_CONTINUATION_FIXTURE=testdata/native_continuation.invalid.json go test ./tool/diagnosticsreplay -run '^TestNativeContinuationReplayGoldenAndDrift$' -count=1; then
  echo '[native-continuation-contract] invented terminal fixture unexpectedly passed' >&2
  exit 1
fi
echo '[native-continuation-contract] passed'
