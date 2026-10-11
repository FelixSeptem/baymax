#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"; cd "$repo_root"
issues=(); add_issue(){ issues+=("$1: $2"); }
mapfile -t changed < <({ git diff --name-only HEAD; git ls-files --others --exclude-standard; } | sed 's#\\#/#g' | sed '/^$/d' | sort -u)
base="$(git merge-base HEAD master 2>/dev/null || true)"
if [[ -n "$base" ]]; then
  mapfile -t changed < <({ printf '%s\n' "${changed[@]}"; git diff --name-only --diff-filter=ACMRTUXB "$base..HEAD"; } | sed 's#\\#/#g' | sed '/^$/d' | sort -u)
fi
areas=(architecture components configuration 'contract/API' diagnostics examples 'CLI/integration' 'best practices' roadmap)
mapfile -t active < <(find openspec/changes -mindepth 1 -maxdepth 1 -type d ! -name archive -print | sort)
if (( ${#active[@]} == 0 )); then echo 'No active OpenSpec changes; documentation impact gate has no proposal assessments to inspect.'; fi
for dir in "${active[@]}"; do
  name="$(basename "$dir")"
  declare -A commitments=()
  for file in proposal.md design.md tasks.md; do
    path="$dir/$file"
    [[ -f "$path" ]] || { add_issue missing-artifact "$name/$file"; continue; }
    section="$(awk '/^## Public Semantic Documentation Commitment[[:space:]]*$/{found=1; next} found && /^## /{exit} found{print}' "$path")"
    [[ -n "$section" ]] || { add_issue missing-public-semantic-documentation-commitment "$name/$file"; continue; }
    semantic="$(printf '%s\n' "$section" | sed -n 's/^[[:space:]]*-[[:space:]]*对外语义变化：[[:space:]]*//p' | head -n1)"
    status="$(printf '%s\n' "$section" | sed -n 's/^[[:space:]]*-[[:space:]]*文档同步状态：[[:space:]]*//p' | head -n1)"
    paths="$(printf '%s\n' "$section" | sed -n 's/^[[:space:]]*-[[:space:]]*维护文档：[[:space:]]*//p' | head -n1)"
    owner="$(printf '%s\n' "$section" | sed -n 's/^[[:space:]]*-[[:space:]]*维护 owner：[[:space:]]*//p' | head -n1)"
    evidence="$(printf '%s\n' "$section" | sed -n 's/^[[:space:]]*-[[:space:]]*验证证据：[[:space:]]*//p' | head -n1)"
    if [[ -z "$semantic" || -z "$status" || -z "$paths" || -z "$owner" || -z "$evidence" ]]; then
      add_issue invalid-public-semantic-documentation-commitment "$name/$file:fields"
    elif [[ "$semantic" != 是 && "$semantic" != 否 ]]; then
      add_issue invalid-public-semantic-documentation-commitment "$name/$file:对外语义变化"
    elif [[ "$semantic" == 是 ]]; then
      [[ "$status" == 已同步 ]] || add_issue public-semantic-doc-sync-missing "$name/$file:文档同步状态"
      [[ "$owner" != — && "$evidence" != — ]] || add_issue public-semantic-doc-sync-missing "$name/$file"
      mapfile -t doc_paths < <(printf '%s\n' "$paths" | grep -oE '`[^`]+\.md`' | tr -d '`' || true)
      if (( ${#doc_paths[@]} == 0 )); then add_issue public-semantic-doc-sync-missing "$name/$file:维护文档"; fi
      for doc_path in "${doc_paths[@]}"; do
        if [[ ! -f "$doc_path" ]] || ! printf '%s\n' "${changed[@]}" | grep -Fxq -- "$doc_path"; then add_issue public-semantic-doc-sync-missing "$name/$file:$doc_path"; fi
      done
    else
      if [[ "$status" != '无需同步（附理由）：'* && "$status" != '无需同步（附理由）:'* ]] || (( ${#status} < 20 )); then add_issue invalid-public-semantic-documentation-commitment "$name/$file:文档同步状态"; fi
    fi
    [[ "$owner" != — && ${#owner} -ge 2 && "$evidence" != — && ${#evidence} -ge 8 ]] || add_issue invalid-public-semantic-documentation-commitment "$name/$file:owner/evidence"
    commitments["$file"]="$semantic|$status|$paths|$owner|$evidence"
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
  [[ "${commitments[proposal.md]:-}" == "${commitments[design.md]:-}" ]] || add_issue public-semantic-documentation-drift "$name:design.md"
  [[ "${commitments[proposal.md]:-}" == "${commitments[tasks.md]:-}" ]] || add_issue public-semantic-documentation-drift "$name:tasks.md"
done
behavior=0; for path in "${changed[@]}"; do [[ ( "$path" =~ (^|/).*\.go$ && ! "$path" =~ _test\.go$ ) || "$path" =~ ^examples/ ]] && behavior=1; done
if (( behavior )); then
  for dir in "${active[@]}"; do
    name="$(basename "$dir")"; proposal="$dir/proposal.md"
    for area in configuration 'contract/API' diagnostics examples; do grep -qF "| $area |" "$proposal" || add_issue surface-without-doc-task "$name:$area"; done
  done
fi
while read -r ref; do [[ -z "$ref" ]] && continue; [[ -f "$ref" ]] || add_issue stale-link "$ref"; done < <(grep -oE '(docs|openspec)/[A-Za-z0-9_./-]+\.md' README.md | sort -u || true)
if (( ${#issues[@]} )); then printf '%s\n' "${issues[@]}" | sort; exit 1; fi
echo 'Documentation impact gate passed.'
