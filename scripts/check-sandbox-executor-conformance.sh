#!/usr/bin/env bash
set -euo pipefail

echo "[sandbox-conformance] running offline deterministic harness"
go test ./integration/sandboxconformance -count=1
echo "[sandbox-conformance] passed"
