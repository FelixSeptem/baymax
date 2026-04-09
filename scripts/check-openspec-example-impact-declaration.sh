#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

min_change_id="${BAYMAX_EXAMPLE_IMPACT_MIN_CHANGE_ID:-70}"
if ! [[ "${min_change_id}" =~ ^[0-9]+$ ]]; then
  echo "[invalid-example-impact-value] BAYMAX_EXAMPLE_IMPACT_MIN_CHANGE_ID must be a non-negative integer, got: ${min_change_id}"
  exit 1
fi

if ! command -v openspec >/dev/null 2>&1; then
  echo "[missing-example-impact-declaration] openspec CLI is required but not found in PATH"
  exit 1
fi

allowed_values=(
  "新增示例"
  "修改示例"
  "无需示例变更（附理由）"
)

is_allowed_value() {
  local value="$1"
  for allowed in "${allowed_values[@]}"; do
    if [[ "${value}" == "${allowed}" ]]; then
      return 0
    fi
  done
  if [[ "${value}" == "无需示例变更（附理由）："* || "${value}" == "无需示例变更（附理由）:"* ]]; then
    local reason="${value#无需示例变更（附理由）}"
    reason="${reason#：}"
    reason="${reason#:}"
    reason="$(echo "${reason}" | xargs || true)"
    if [[ -n "${reason}" ]]; then
      return 0
    fi
  fi
  return 1
}

extract_change_id() {
  local change_name="$1"
  local id
  id="$(grep -oE -- '-a[0-9]+($|-[a-z0-9]+$|-[a-z0-9-]+$)' <<< "${change_name}" | tail -n 1 | grep -oE '[0-9]+' || true)"
  if [[ -n "${id}" ]]; then
    printf '%s' "${id}"
  fi
}

extract_declaration_value() {
  local proposal_path="$1"
  awk '
    BEGIN {
      in_section = 0
    }
    function trim(s) {
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", s)
      return s
    }
    {
      line = $0
      normalized = trim(line)
      lower = tolower(normalized)

      if (!in_section) {
        if (lower ~ /^##[[:space:]]*example[[:space:]]+impact[[:space:]]+assessment[[:space:]]*$/ || normalized ~ /^##[[:space:]]*示例影响评估[[:space:]]*$/) {
          in_section = 1
        }
        next
      }

      if (normalized ~ /^##[[:space:]]+/) {
        exit
      }
      if (normalized == "") {
        next
      }

      sub(/^-+[[:space:]]*/, "", normalized)
      sub(/^\*+[[:space:]]*/, "", normalized)
      if (normalized ~ /^\[[xX ]\][[:space:]]+/) {
        normalized = substr(normalized, 4)
      }
      normalized = trim(normalized)
      gsub(/^`|`$/, "", normalized)
      normalized = trim(normalized)
      if (normalized == "") {
        next
      }
      print normalized
      exit
    }
  ' "${proposal_path}"
}

mapfile -t active_changes < <(
  openspec list --json | awk '
    /"name"[[:space:]]*:/ {
      if (match($0, /"name"[[:space:]]*:[[:space:]]*"([^"]+)"/, m)) {
        name = m[1]
      }
      next
    }
    /"status"[[:space:]]*:/ {
      if (match($0, /"status"[[:space:]]*:[[:space:]]*"([^"]+)"/, m)) {
        status = m[1]
        if (status == "in-progress" && name != "") {
          print name
        }
      }
      name = ""
    }
  ' | sort -u
)

issues=()

for change in "${active_changes[@]}"; do
  [[ -z "${change}" ]] && continue

  change_id="$(extract_change_id "${change}")"
  if [[ -n "${change_id}" ]] && (( change_id < min_change_id )); then
    continue
  fi

  proposal_path="openspec/changes/${change}/proposal.md"
  if [[ ! -f "${proposal_path}" ]]; then
    issues+=("[missing-example-impact-declaration] ${change}: missing proposal file ${proposal_path}")
    continue
  fi

  declaration_value="$(extract_declaration_value "${proposal_path}")"
  if [[ -z "${declaration_value}" ]]; then
    issues+=("[missing-example-impact-declaration] ${change}: missing Example Impact Assessment declaration in ${proposal_path}")
    continue
  fi

  if ! is_allowed_value "${declaration_value}"; then
    issues+=("[invalid-example-impact-value] ${change}: unsupported declaration \"${declaration_value}\" in ${proposal_path}")
  fi
done

if (( ${#issues[@]} > 0 )); then
  for issue in "${issues[@]}"; do
    echo "${issue}"
  done
  echo "hint: add section '## Example Impact Assessment' in proposal.md and use one of: 新增示例 | 修改示例 | 无需示例变更（附理由）"
  echo "hint: this gate only enforces changes with numeric suffix >= a${min_change_id}."
  exit 1
fi

echo "[openspec-example-impact-declaration] passed"
