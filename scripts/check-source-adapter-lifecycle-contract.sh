#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
echo '[source-adapter-lifecycle-contract] bounded delivery, durable state and pure replay'
go test ./host/sourceadapter ./integration ./tool/diagnosticsreplay ./tool/contributioncheck ./examples/source-adapter-lifecycle -run 'TestSourceAdapter|TestDisposition|TestWholePage|TestPending|TestPoll|TestAbort|TestOperation|TestHostEffect|TestBounds|TestRecent|TestFileStore' -count=1 -timeout 2m
set +e
invalid_output="$(BAYMAX_SOURCE_ADAPTER_FIXTURE=testdata/source_adapter_lifecycle.invalid.json go test ./tool/diagnosticsreplay -run TestSourceAdapterReplayGoldenAndDrift -count=1 2>&1)"
invalid_exit=$?
set -e
if [[ $invalid_exit -eq 0 || "$invalid_output" != *source_adapter.replay_drift* ]]; then
  echo '[source-adapter-lifecycle-contract] invalid fixture lacks source_adapter.replay_drift failure' >&2
  exit 1
fi
echo "[source-adapter-lifecycle-contract] expected drift exit=$invalid_exit classification=source_adapter.replay_drift"
smoke="$(go run ./examples/source-adapter-lifecycle)"
for marker in sent_confirmed gone_confirmed retryable_preserved duplicate_no_effect restart_confirmed_cursor abort_preserved; do
  if ! grep -Fxq "$marker" <<<"$smoke"; then
    echo "[source-adapter-lifecycle-contract] missing marker $marker" >&2
    exit 1
  fi
done
echo '[source-adapter-lifecycle-contract] passed'
