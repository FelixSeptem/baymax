#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

banned_pattern='(\.go\.[0-9]+$|\.tmp$|\.bak$|~$)'

tracked_candidates="$(git ls-files | grep -E "${banned_pattern}" || true)"
untracked_candidates="$(git ls-files --others --exclude-standard | grep -E "${banned_pattern}" || true)"
candidates="$(printf '%s\n%s\n' "${tracked_candidates}" "${untracked_candidates}" | sed '/^$/d' | sort -u || true)"
if [[ -n "${candidates}" ]]; then
  echo "[repo-hygiene] found banned temporary/backup artifacts (tracked or untracked):" >&2
  echo "${candidates}" >&2
  exit 1
fi

echo "[repo-hygiene] passed"
