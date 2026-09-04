$ErrorActionPreference = 'Stop'
$env:GOSUMDB = 'sum.golang.org'
Write-Host '[session-history-checkpoint-replay-gate] core and snapshot contract suites'
go test ./core/types ./orchestration/snapshot -run 'SessionHistory|CheckpointHistoryLeafAssociation|ProtocolCheckpoint|ImporterValidatesHistory' -count=1
Write-Host '[session-history-checkpoint-replay-gate] replay fixture suites'
go test ./tool/diagnosticsreplay -run SessionHistory -count=1
Write-Host '[session-history-checkpoint-replay-gate] done'
