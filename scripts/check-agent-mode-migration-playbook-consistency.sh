#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

MATRIX_PATH="examples/agent-modes/MATRIX.md"
PLAYBOOK_PATH="examples/agent-modes/PLAYBOOK.md"

echo "[agent-mode-migration-playbook-consistency] validating matrix/playbook/readme consistency"

if [[ ! -f "${MATRIX_PATH}" ]]; then
  echo "[agent-mode-migration-playbook-consistency][missing-checklist] missing matrix: ${MATRIX_PATH}" >&2
  exit 1
fi
if [[ ! -f "${PLAYBOOK_PATH}" ]]; then
  echo "[agent-mode-migration-playbook-consistency][missing-checklist] missing playbook: ${PLAYBOOK_PATH}" >&2
  exit 1
fi

missing_checklist=()
missing_gate=()

while IFS= read -r row; do
  [[ -n "${row}" ]] || continue
  pattern="$(echo "${row}" | awk -F '|' '{print $2}' | xargs)"
  pattern="${pattern#\`}"
  pattern="${pattern%\`}"
  gates_cell="$(echo "${row}" | awk -F '|' '{print $6}' | xargs)"
  readme_path="examples/agent-modes/${pattern}/production-ish/README.md"

  if [[ ! -f "${readme_path}" ]]; then
    missing_checklist+=("${pattern}:missing-production-ish-readme")
  else
    if ! grep -q "^## Prod Delta Checklist$" "${readme_path}"; then
      missing_checklist+=("${pattern}:missing-prod-delta-checklist")
    fi
  fi

  if ! grep -Fq "\`${pattern}\`" "${PLAYBOOK_PATH}"; then
    missing_checklist+=("${pattern}:missing-playbook-pattern-mapping")
  fi

  mapfile -t gate_tokens < <(echo "${gates_cell}" | grep -oE '`[^`]+`' | tr -d '`')
  for gate in "${gate_tokens[@]}"; do
    [[ -n "${gate}" ]] || continue
    if ! grep -Fq "${gate}" "${PLAYBOOK_PATH}"; then
      missing_gate+=("${pattern}:playbook-missing-gate:${gate}")
    fi
    if [[ -f "${readme_path}" ]] && ! grep -Fq "${gate}" "${readme_path}"; then
      missing_gate+=("${pattern}:production-ish-missing-gate:${gate}")
    fi
  done
done < <(grep -E '^\| `[^`]+` \|' "${MATRIX_PATH}")

if (( ${#missing_checklist[@]} > 0 )); then
  echo "[agent-mode-migration-playbook-consistency][missing-checklist] inconsistencies found:" >&2
  printf '  - %s\n' "${missing_checklist[@]}" >&2
fi

if (( ${#missing_gate[@]} > 0 )); then
  echo "[agent-mode-migration-playbook-consistency][missing-gate] inconsistencies found:" >&2
  printf '  - %s\n' "${missing_gate[@]}" >&2
fi

if (( ${#missing_checklist[@]} > 0 || ${#missing_gate[@]} > 0 )); then
  exit 1
fi

echo "[agent-mode-migration-playbook-consistency] consistency is complete"
