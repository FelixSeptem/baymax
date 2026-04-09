#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

echo "[agent-mode-readme-runtime-sync-contract] validating README runtime sync for agent-mode examples"

mapfile -t main_files < <(find examples/agent-modes -mindepth 3 -maxdepth 3 -type f -name main.go | sort)
if (( ${#main_files[@]} == 0 )); then
  echo "[agent-mode-readme-runtime-sync-contract] no agent-mode main.go found" >&2
  exit 1
fi

required_sections=(
  "## Run"
  "## Prerequisites"
  "## Real Runtime Path"
  "## Expected Output/Verification"
  "## Failure/Rollback Notes"
)

missing_required_sections=()
for main_file in "${main_files[@]}"; do
  readme_file="$(dirname "${main_file}")/README.md"
  if [[ ! -f "${readme_file}" ]]; then
    missing_required_sections+=("${readme_file}:missing-readme")
    continue
  fi
  for section in "${required_sections[@]}"; do
    if ! grep -Fq "${section}" "${readme_file}"; then
      missing_required_sections+=("${readme_file}:missing-section:${section}")
    fi
  done
done

changed_main_files=()
changed_readme_files=()
while IFS= read -r path; do
  [[ -n "${path}" ]] || continue
  if [[ "${path}" == examples/agent-modes/*/main.go ]]; then
    changed_main_files+=("${path}")
  fi
  if [[ "${path}" == examples/agent-modes/*/README.md ]]; then
    changed_readme_files+=("${path}")
  fi
done < <(git status --porcelain -- examples/agent-modes | awk '{print $NF}')

readme_not_updated=()
for main_file in "${changed_main_files[@]}"; do
  readme_file="$(dirname "${main_file}")/README.md"
  found=0
  for changed_readme in "${changed_readme_files[@]}"; do
    if [[ "${changed_readme}" == "${readme_file}" ]]; then
      found=1
      break
    fi
  done
  if [[ ${found} -eq 0 ]]; then
    readme_not_updated+=("${main_file} -> ${readme_file}")
  fi
done

if (( ${#readme_not_updated[@]} > 0 )); then
  echo "[agent-mode-readme-runtime-sync-contract][agent-mode-readme-runtime-desync] main.go changed without matching README update:" >&2
  printf '  - %s\n' "${readme_not_updated[@]}" >&2
fi

if (( ${#missing_required_sections[@]} > 0 )); then
  echo "[agent-mode-readme-runtime-sync-contract][agent-mode-readme-required-sections-missing] required README sections missing:" >&2
  printf '  - %s\n' "${missing_required_sections[@]}" >&2
fi

if (( ${#readme_not_updated[@]} > 0 || ${#missing_required_sections[@]} > 0 )); then
  exit 1
fi

echo "[agent-mode-readme-runtime-sync-contract] passed"