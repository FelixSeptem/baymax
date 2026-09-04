#!/usr/bin/env bash
set -euo pipefail
export GOSUMDB=sum.golang.org
echo '[session-history-checkpoint-replay-gate] core and snapshot contract suites'
go test ./core/types ./orchestration/snapshot -run 'SessionHistory|CheckpointHistoryLeafAssociation|ProtocolCheckpoint|ImporterValidatesHistory' -count=1
echo '[session-history-checkpoint-replay-gate] replay fixture suites'
go test ./tool/diagnosticsreplay -run SessionHistory -count=1
echo '[session-history-checkpoint-replay-gate] done'
