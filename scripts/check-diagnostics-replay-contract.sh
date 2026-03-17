#!/usr/bin/env bash
set -euo pipefail

echo "[diagnostics-replay-gate] replay contract tests"
go test ./tool/diagnosticsreplay -run '^TestReplayContract' -count=1
