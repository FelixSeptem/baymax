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
  "A20_RUN_TERMINAL"
  "A20_STREAM_TERMINAL"
  "A20_TERMINAL_SUMMARY="
  "A20_SUCCESS"
)

for marker in "${required_markers[@]}"; do
  if ! grep -Fq "${marker}" <<<"${output}"; then
    echo "[example-smoke] missing required marker: ${marker}"
    exit 1
  fi
done

echo "[example-smoke] passed"
