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

echo "Runtime boundary check passed."

