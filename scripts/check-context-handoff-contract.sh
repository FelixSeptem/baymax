#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"
GOTOOLCHAIN=local go test ./context/handoff ./tool/diagnosticsreplay -count=1
echo "[context-handoff-contract-gate] done"
