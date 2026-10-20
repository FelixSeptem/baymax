#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
export GOCACHE="${BAYMAX_CANDIDATE_COMPARISON_GOCACHE:-$repo_root/.gocache-candidate-comparison}"
if [[ $# -gt 0 ]]; then
  export BAYMAX_CANDIDATE_COMPARISON_FIXTURE="$1"
fi

echo '[candidate-comparison-contract] bounded identity, isolation, selection and cost replay'
go test ./runtime/evalcontract ./tool/diagnosticsreplay -run 'Test(NormalizeCandidateExperiment|CompareCandidateIdentity|AdmitCandidateAndSelect|BuildCandidateComparison|CandidateComparisonReplay|EvalContractFixtureSuccessAndLegacyShape)' -count=1
pattern='github.com/FelixSeptem/baymax/(model/|core/|runtime/runner)|\.(Run|Stream|Retry|Resume|Execute)\('
scan_status=0
rg -n -- "$pattern" tool/diagnosticsreplay/candidate_comparison.go || scan_status=$?
if [[ $scan_status -eq 0 ]]; then
  echo '[candidate-comparison-contract] replay owns an executable surface' >&2
  exit 1
fi
if [[ $scan_status -ne 1 ]]; then
  echo '[candidate-comparison-contract] ownership scan failed' >&2
  exit "$scan_status"
fi
echo '[candidate-comparison-contract] passed'
