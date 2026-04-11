#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

echo "[agent-mode-doc-first-delivery-contract] validating doc-first delivery constraints"

matrix_file="examples/agent-modes/MATRIX.md"
playbook_file="examples/agent-modes/PLAYBOOK.md"
baseline_file="examples/agent-modes/doc-baseline-freeze.md"

if [[ ! -f "${matrix_file}" ]]; then
  echo "[agent-mode-doc-first-delivery-contract][agent-mode-doc-first-baseline-missing] missing matrix: ${matrix_file}" >&2
  exit 1
fi
if [[ ! -f "${playbook_file}" ]]; then
  echo "[agent-mode-doc-first-delivery-contract][agent-mode-doc-first-baseline-missing] missing playbook: ${playbook_file}" >&2
  exit 1
fi
if [[ ! -f "${baseline_file}" ]]; then
  echo "[agent-mode-doc-first-delivery-contract][agent-mode-doc-first-baseline-missing] missing baseline freeze: ${baseline_file}" >&2
  exit 1
fi

if ! grep -Fq "doc-baseline-ready" "${matrix_file}" || ! grep -Fq "impl-ready" "${matrix_file}" || ! grep -Fq "failure_rollback_ref" "${matrix_file}"; then
  echo "[agent-mode-doc-first-delivery-contract][agent-mode-doc-first-baseline-missing] matrix missing doc-first columns" >&2
  exit 1
fi

required_sections=(
  "## Run"
  "## Prerequisites"
  "## Real Runtime Path"
  "## Expected Output/Verification"
  "## Failure/Rollback Notes"
)

missing_readme_sections=()
for readme in examples/agent-modes/*/*/README.md; do
  [[ -f "${readme}" ]] || continue
  for section in "${required_sections[@]}"; do
    if ! grep -Fq "${section}" "${readme}"; then
      missing_readme_sections+=("${readme}:missing-section:${section}")
    fi
  done
  if [[ "${readme}" == */production-ish/README.md ]]; then
    if ! grep -Fq "## Variant Delta (vs minimal)" "${readme}"; then
      missing_readme_sections+=("${readme}:missing-section:## Variant Delta (vs minimal)")
    fi
  fi
done

changed_code_files=()
while IFS= read -r path; do
  [[ -n "${path}" ]] || continue
  if [[ "${path}" =~ ^examples/agent-modes/[^/]+/semantic_example\.go$ || "${path}" =~ ^examples/agent-modes/[^/]+/(minimal|production-ish)/main\.go$ ]]; then
    changed_code_files+=("${path}")
  fi
done < <(git status --porcelain -- examples/agent-modes | awk '{print $NF}')

doc_first_baseline_missing=()
if (( ${#changed_code_files[@]} > 0 )); then
  for code_file in "${changed_code_files[@]}"; do
    pattern="$(echo "${code_file}" | awk -F/ '{print $3}')"
    row="$(grep -E "^\| \`${pattern}\` \|" "${matrix_file}" || true)"
    if [[ -z "${row}" ]]; then
      doc_first_baseline_missing+=("${code_file}:missing-matrix-row")
      continue
    fi
    if [[ "${row}" != *"| yes | yes |"* ]]; then
      doc_first_baseline_missing+=("${code_file}:doc-baseline-ready/impl-ready not both yes")
    fi
  done
fi

if (( ${#doc_first_baseline_missing[@]} > 0 )); then
  echo "[agent-mode-doc-first-delivery-contract][agent-mode-doc-first-baseline-missing] doc-first baseline is incomplete for changed code paths:" >&2
  printf '  - %s\n' "${doc_first_baseline_missing[@]}" >&2
fi

if (( ${#missing_readme_sections[@]} > 0 )); then
  echo "[agent-mode-doc-first-delivery-contract][agent-mode-doc-required-sections-missing] required doc sections missing:" >&2
  printf '  - %s\n' "${missing_readme_sections[@]}" >&2
fi

if (( ${#doc_first_baseline_missing[@]} > 0 || ${#missing_readme_sections[@]} > 0 )); then
  exit 1
fi

echo "[agent-mode-doc-first-delivery-contract] passed"
