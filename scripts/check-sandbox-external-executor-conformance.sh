#!/usr/bin/env bash
# Gate: sandbox external executor conformance (shell/PowerShell parity).
set -euo pipefail
cd "$(dirname "$0")/.."

go test ./adapter/sandboxexternal/... ./integration/sandboxconformance/... -count=1

echo "[sandbox-external-executor-conformance] passed"
