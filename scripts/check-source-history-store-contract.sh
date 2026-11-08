#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
echo '[source-history-store-contract] owner binding, fencing, bounded replay and reconciliation'
go test ./core/history ./core/sourcecoordination ./core/types ./tool/diagnosticsreplay -run 'Test(OwnerStore|Journal|CoordinatedJournal|DeploymentScope|ReplayDeploymentScope)' -count=1
fixture=core/history/testdata/owner_store_deployment.v1.json
test -f "$fixture"
rg -q '"version": "owner_store_deployment.v1"' "$fixture"
rg -q '"side_effects": false' "$fixture"
echo '[source-history-store-contract] passed'
