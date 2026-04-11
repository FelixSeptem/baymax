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

required_sections=(
  "## Run"
  "## Prerequisites"
  "## Real Runtime Path"
  "## Expected Output/Verification"
  "## Failure/Rollback Notes"
)

missing_checklist=()
missing_gate=()

header_line="$(grep -E '^\|[[:space:]]*pattern[[:space:]]*\|' "${MATRIX_PATH}" | head -n 1 || true)"
if [[ -z "${header_line}" ]]; then
  echo "[agent-mode-migration-playbook-consistency][missing-checklist] missing matrix header row: ${MATRIX_PATH}" >&2
  exit 1
fi

IFS='|' read -r -a header_cols <<<"${header_line}"
pattern_idx=-1
gates_idx=-1
for i in "${!header_cols[@]}"; do
  col_name="$(echo "${header_cols[$i]}" | xargs)"
  if [[ "${col_name}" == "pattern" ]]; then
    pattern_idx="${i}"
  fi
  if [[ "${col_name}" == "gates" ]]; then
    gates_idx="${i}"
  fi
done

if (( pattern_idx < 0 || gates_idx < 0 )); then
  echo "[agent-mode-migration-playbook-consistency][missing-checklist] missing required columns in matrix header (pattern/gates)" >&2
  exit 1
fi

if ! grep -Fq "## Production Migration Checklist" "${PLAYBOOK_PATH}"; then
  missing_checklist+=("playbook:missing-production-migration-checklist")
fi

while IFS= read -r row; do
  [[ -n "${row}" ]] || continue
  IFS='|' read -r -a row_cols <<<"${row}"
  if (( ${#row_cols[@]} <= pattern_idx || ${#row_cols[@]} <= gates_idx )); then
    continue
  fi

  pattern="$(echo "${row_cols[$pattern_idx]}" | xargs)"
  pattern="${pattern#\`}"
  pattern="${pattern%\`}"
  [[ -n "${pattern}" ]] || continue

  gates_cell="$(echo "${row_cols[$gates_idx]}" | xargs)"
  gates_cell="${gates_cell#\`}"
  gates_cell="${gates_cell%\`}"
  readme_path="examples/agent-modes/${pattern}/production-ish/README.md"

  if [[ ! -f "${readme_path}" ]]; then
    missing_checklist+=("${pattern}:missing-production-ish-readme")
  else
    for section in "${required_sections[@]}"; do
      if ! grep -Fq "${section}" "${readme_path}"; then
        missing_checklist+=("${pattern}:missing-section:${section}")
      fi
    done
  fi

  if ! grep -Fq "\`${pattern}\`" "${PLAYBOOK_PATH}"; then
    missing_checklist+=("${pattern}:missing-playbook-pattern-mapping")
  fi

  IFS=';' read -r -a gate_tokens <<<"${gates_cell}"
  for gate in "${gate_tokens[@]}"; do
    gate="$(echo "${gate}" | xargs)"
    [[ -n "${gate}" ]] || continue
    if [[ "${gate}" == "-" ]]; then
      continue
    fi
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
