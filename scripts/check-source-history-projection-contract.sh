#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
echo '[source-history-projection-contract] committed facts, projection idempotency and recovery-required'
go test ./core/history ./core/types ./orchestration/planbridge ./core/runner ./orchestration/composer ./orchestration/teams ./orchestration/workflow -run '^TestProjectionLedger|^TestJournalAppend.*ProjectionLedger|^TestJournalProjectionRecovery|^TestJournalCapabilitiesDeclareProjectionRecoveryOnlyForDurableLedger|^Test.*BindsSourceProjectionLedger|^TestCapabilityManifestHistoryProjectionRecoveryRequiresDurableOwnerLedger$' -count=1
test -f tool/diagnosticsreplay/testdata/source-history-projection.v1.json
rg -q '"version": "source-history-projection.v1"' tool/diagnosticsreplay/testdata/source-history-projection.v1.json
rg -q '"outcome": "recovery_required"' tool/diagnosticsreplay/testdata/source-history-projection.v1.json
echo '[source-history-projection-contract] passed'
