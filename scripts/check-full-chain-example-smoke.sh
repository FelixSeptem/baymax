#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$(pwd)/.gocache"
fi

echo "[example-smoke] running full-chain example"
if ! output="$(go run ./examples/09-multi-agent-full-chain-reference 2>&1)"; then
  echo "${output}"
  echo "[example-smoke] full-chain example execution failed"
  exit 1
fi
echo "${output}"

required_markers=(
  "CHECKPOINT async_report_succeeded=true"
  "CHECKPOINT delayed_dispatch_claimed=true"
  "CHECKPOINT recovery_replayed=true"
  "CHECKPOINT correlation "
  "CHECKPOINT run_stream_aligned=true"
)

for marker in "${required_markers[@]}"; do
  if ! grep -Fq "${marker}" <<<"${output}"; then
    echo "[example-smoke] missing required marker: ${marker}"
    exit 1
  fi
done

require_marker() {
  local marker="$1"
  if grep -Fq "${marker}" <<<"${output}"; then
    return 0
  fi
  echo "[example-smoke] missing required marker: ${marker}"
  exit 1
}

require_marker "FULL_CHAIN_RUN_TERMINAL"
require_marker "FULL_CHAIN_STREAM_TERMINAL"
require_marker "FULL_CHAIN_TERMINAL_SUMMARY="
require_marker "FULL_CHAIN_SUCCESS"

echo "[example-smoke] passed"
