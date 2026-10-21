#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"
echo '[workflow-continuation-contract] source restore, unsupported forms, and read-only replay'
go test ./orchestration/workflow ./orchestration/planbridge ./tool/diagnosticsreplay -count=1
file=tool/diagnosticsreplay/workflow_continuation.go
pattern='"(net/|database/|os"|github.com/FelixSeptem/baymax/(orchestration/workflow|core/checkpoint|core/runner|runtime/|observability/|model/|mcp/))|^[[:space:]]*go[[:space:]]+|\.(Run|Stream|Resume|ClaimResume|MarkResumeRunning|CommitResumeResult|Save|Load)\('
status=0
rg -n -- "$pattern" "$file" && status=0 || status=$?
if [[ "$status" != 1 ]]; then
  echo '[workflow-continuation-contract][ownership] replay must remain reference-only' >&2
  exit 1
fi
if BAYMAX_WORKFLOW_CONTINUATION_FIXTURE=testdata/workflow_continuation.invalid.json go test ./tool/diagnosticsreplay -run '^TestWorkflowContinuationReplayGoldenAndDrift$' -count=1; then
  echo '[workflow-continuation-contract] invalid fixture unexpectedly passed' >&2
  exit 1
fi
echo '[workflow-continuation-contract] passed'
