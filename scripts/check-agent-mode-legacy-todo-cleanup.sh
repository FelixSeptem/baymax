#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

echo "[agent-mode-legacy-todo-cleanup] scanning examples for unresolved placeholders"

if ! command -v rg >/dev/null 2>&1; then
  echo "[agent-mode-legacy-todo-cleanup] ripgrep (rg) is required" >&2
  exit 1
fi

set +e
matches_raw="$(rg -n "TODO|TBD|FIXME|待补" examples)"
status=$?
set -e

if [[ ${status} -eq 0 ]]; then
  matches="$(echo "${matches_raw}" | grep -v -E 'examples[\\/]+agent-modes[\\/]+LEGACY_TODO_BASELINE.md' || true)"
fi

if [[ ${status} -eq 0 && -n "${matches:-}" ]]; then
  echo "[agent-mode-legacy-todo-cleanup][legacy-placeholder] unresolved placeholders found:" >&2
  echo "${matches}" >&2
  exit 1
fi
if [[ ${status} -ne 1 ]]; then
  echo "[agent-mode-legacy-todo-cleanup] placeholder scan failed" >&2
  exit 1
fi

echo "[agent-mode-legacy-todo-cleanup] cleanup is complete"
