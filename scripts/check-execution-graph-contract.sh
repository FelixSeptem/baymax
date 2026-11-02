#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
echo '[execution-graph-contract] owner digest stays separate from topology digest'

go test ./orchestration/graph ./orchestration/workflow ./orchestration/teams ./orchestration/composer ./examples/execution-graph -run '^Test(Compile|Canonical|EquivalentTopology|TopologyDrift|ProjectExecutionGraph|ProjectChildExecutionGraph|OfflineExample)' -count=1

if rg -n --glob '*.go' --glob '!**/*_test.go' '"github.com/FelixSeptem/baymax/orchestration/(workflow|teams|composer|scheduler|checkpoint|runner)' orchestration/graph; then
  echo '[execution-graph-contract][ownership] shared graph package imports an owner or executor' >&2
  exit 1
else
  status=$?
  if [[ "$status" -ne 1 ]]; then
    echo '[execution-graph-contract][ownership] import scan failed' >&2
    exit "$status"
  fi
fi

if rg -n --glob 'graph_projection.go' '\.(Run|Stream|Resume|Claim|Save|Commit)\(' orchestration/workflow orchestration/teams orchestration/composer; then
  echo '[execution-graph-contract][ownership] projection invokes execution or restore owner' >&2
  exit 1
else
  status=$?
  if [[ "$status" -ne 1 ]]; then
    echo '[execution-graph-contract][ownership] projection scan failed' >&2
    exit "$status"
  fi
fi

echo '[execution-graph-contract] owner digest and execution boundaries passed'
