#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

violations="$(rg -n "\"github.com/FelixSeptem/baymax/mcp/(http|stdio)\"" runtime/config runtime/diagnostics || true)"

if [[ -n "${violations}" ]]; then
  echo "Runtime boundary violation detected:"
  echo "${violations}"
  exit 1
fi

internal_imports="$(rg -n "\"github.com/FelixSeptem/baymax/mcp/internal/\"|\"github.com/FelixSeptem/baymax/mcp/internal" core docs examples integration model observability runtime skill tool || true)"

if [[ -n "${internal_imports}" ]]; then
  echo "MCP internal boundary violation detected:"
  echo "${internal_imports}"
  exit 1
fi

echo "Runtime boundary check passed."
