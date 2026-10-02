#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"; cd "$repo_root"
issues=(); add_issue(){ issues+=("$1: $2"); }
areas=(architecture components configuration 'contract/API' diagnostics examples 'CLI/integration' 'best practices' roadmap)
mapfile -t active < <(find openspec/changes -mindepth 1 -maxdepth 1 -type d ! -name archive -print | sort)
if (( ${#active[@]} == 0 )); then echo 'No active OpenSpec changes; documentation impact gate has no proposal assessments to inspect.'; fi
for dir in "${active[@]}"; do
  name="$(basename "$dir")"
  for file in proposal.md design.md tasks.md; do
    path="$dir/$file"
    [[ -f "$path" ]] || { add_issue missing-artifact "$name/$file"; continue; }
    grep -qE '^##[[:space:]]+Documentation Impact Assessment[[:space:]]*$' "$path" || { add_issue missing-assessment "$name/$file"; continue; }
    for area in "${areas[@]}"; do
      grep -qF "| $area |" "$path" || add_issue missing-area "$name/$file:$area"
    done
    while IFS='|' read -r _ area outcome paths owner verify _; do
      area="$(echo "$area" | xargs)"; outcome="$(echo "$outcome" | xargs)"; paths="$(echo "$paths" | xargs)"; owner="$(echo "$owner" | xargs)"; verify="$(echo "$verify" | xargs)"
      [[ -z "$area" || "$area" == Area || "$area" == ---* ]] && continue
      if [[ "$outcome" == '无需文档变更（附理由）' && ( "$verify" == '—' || ${#verify} -lt 8 ) ]]; then add_issue missing-reason "$name/$file:$area"; fi
      if [[ "$outcome" == '新增文档' || "$outcome" == '修改文档' ]]; then [[ "$paths" == '—' || "$owner" == '—' || "$verify" == '—' ]] && add_issue missing-evidence "$name/$file:$area"; fi
    done < <(grep -E '^\|.*\|.*\|.*\|.*\|.*\|' "$path" || true)
  done
done
mapfile -t changed < <({ git diff --name-only HEAD; git ls-files --others --exclude-standard; } | sed 's#\\#/#g' | sed '/^$/d' | sort -u)
behavior=0; for path in "${changed[@]}"; do [[ "$path" =~ (^|/).*\.go$ || "$path" =~ ^examples/ ]] && behavior=1; done
if (( behavior )); then
  for dir in "${active[@]}"; do
    name="$(basename "$dir")"; proposal="$dir/proposal.md"
    for area in configuration 'contract/API' diagnostics examples; do grep -qF "| $area |" "$proposal" || add_issue surface-without-doc-task "$name:$area"; done
  done
fi
while read -r ref; do [[ -z "$ref" ]] && continue; [[ -f "$ref" ]] || add_issue stale-link "$ref"; done < <(grep -oE '(docs|openspec)/[A-Za-z0-9_./-]+\.md' README.md | sort -u || true)
if (( ${#issues[@]} )); then printf '%s\n' "${issues[@]}" | sort; exit 1; fi
echo 'Documentation impact gate passed.'
