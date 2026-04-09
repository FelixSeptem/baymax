#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

echo "[agent-mode-real-logic-contract] validating real runtime entrypoints"

mapfile -t main_files < <(find examples/agent-modes -mindepth 3 -maxdepth 3 -type f -name main.go | sort)
if (( ${#main_files[@]} == 0 )); then
  echo "[agent-mode-real-logic-contract] no agent-mode main.go found" >&2
  exit 1
fi

simulated_dependency_violations=()
placeholder_regressions=()
missing_runtime_path=()

for file in "${main_files[@]}"; do
  if grep -q "examples/agent-modes/internal/agentmode" "${file}"; then
    simulated_dependency_violations+=("${file}")
  fi

  if ! grep -q "github.com/FelixSeptem/baymax/core/runner" "${file}" ||
    ! grep -q "github.com/FelixSeptem/baymax/tool/local" "${file}" ||
    ! grep -q "github.com/FelixSeptem/baymax/runtime/config" "${file}" ||
    ! grep -q "runner.New(" "${file}"; then
    missing_runtime_path+=("${file}")
  fi

  if ! grep -q "verification.mainline_runtime_path=" "${file}" ||
    ! grep -q "result.final_answer=" "${file}" ||
    ! grep -q "result.signature=" "${file}"; then
    placeholder_regressions+=("${file}")
  fi
done

if (( ${#simulated_dependency_violations[@]} > 0 )); then
  echo "[agent-mode-real-logic-contract][agent-mode-simulated-engine-dependency] prohibited simulation dependency detected:" >&2
  printf '  - %s\n' "${simulated_dependency_violations[@]}" >&2
fi
if (( ${#placeholder_regressions[@]} > 0 )); then
  echo "[agent-mode-real-logic-contract][agent-mode-placeholder-output-regression] required real runtime output markers missing:" >&2
  printf '  - %s\n' "${placeholder_regressions[@]}" >&2
fi
if (( ${#missing_runtime_path[@]} > 0 )); then
  echo "[agent-mode-real-logic-contract][agent-mode-missing-mainline-runtime-path] required runtime wiring missing:" >&2
  printf '  - %s\n' "${missing_runtime_path[@]}" >&2
fi

if (( ${#simulated_dependency_violations[@]} > 0 || ${#placeholder_regressions[@]} > 0 || ${#missing_runtime_path[@]} > 0 )); then
  exit 1
fi

echo "[agent-mode-real-logic-contract] all agent-mode entrypoints satisfy real runtime contract"
