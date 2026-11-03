#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
echo '[attention-diagnostics-contract] bounded pure snapshot and replay'
go test ./runtime/diagnostics ./integration ./tool/diagnosticsreplay ./tool/contributioncheck ./examples/attention-diagnostics -run TestAttention -count=1 -timeout 2m
set +e
invalid_output="$(BAYMAX_ATTENTION_FIXTURE=testdata/attention.invalid.json go test ./tool/diagnosticsreplay -run TestAttentionReplayGoldenAndDrift -count=1 2>&1)"
invalid_exit=$?
set -e
if [[ $invalid_exit -eq 0 || "$invalid_output" != *attention.replay_drift* ]]; then
  echo '[attention-diagnostics-contract] invalid fixture lacks attention.replay_drift failure' >&2
  exit 1
fi
echo "[attention-diagnostics-contract] expected drift exit=$invalid_exit classification=attention.replay_drift"
smoke="$(go run ./examples/attention-diagnostics)"
for marker in healthy_retrying_ignored effect_pending_explicit truncation_visible; do
  if ! grep -Fxq "$marker" <<<"$smoke"; then
    echo "[attention-diagnostics-contract] missing marker $marker" >&2
    exit 1
  fi
done
echo '[attention-diagnostics-contract] passed'
