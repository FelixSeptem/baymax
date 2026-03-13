#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

banned_pattern='(\.go\.[0-9]+$|\.tmp$|\.bak$|~$)'

candidates="$(git ls-files | grep -E "${banned_pattern}" || true)"
if [[ -n "${candidates}" ]]; then
  echo "[repo-hygiene] found banned temporary/backup artifacts:" >&2
  echo "${candidates}" >&2
  exit 1
fi

echo "[repo-hygiene] passed"
