#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

echo '[team-continuation-contract] policy binding, provenance, restore, and capability evidence'
go test ./orchestration/teams ./orchestration/planbridge ./core/types -count=1 -run 'TestTeam|TestContinuationCapabilities|TestVersionedPauseReasonMatrixFixtures'
rg -n 'teamPolicyDigest|teamAggregateProvenance|policy_binding|ambiguous_in_flight|run_stream_parity' orchestration/teams/continuation.go orchestration/teams/continuation_test.go
echo '[team-continuation-contract] passed'
