#!/usr/bin/env bash
set -euo pipefail
if [[ -z "${GOCACHE:-}" ]]; then export GOCACHE="$(pwd)/.gocache"; fi
export GOSUMDB="${GOSUMDB:-sum.golang.org}"
export GOPROXY="${GOPROXY:-https://proxy.golang.org,direct}"
echo "[extension-lifecycle-replay] running deterministic extension lifecycle checks"
go test ./integration/extensionlifecycle -run '^TestReplayContract' -count=1
echo "[extension-lifecycle-replay] passed"
