#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

roadmap_path="docs/development-roadmap.md"
archive_index_path="openspec/changes/archive/INDEX.md"

trim_line() {
  local value="$1"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  printf '%s' "${value}"
}

extract_roadmap_slug() {
  local line="$1"
  if [[ "${line}" == *"\`"* ]]; then
    local rest="${line#*\`}"
    local slug="${rest%%\`*}"
    slug="$(trim_line "${slug}")"
    if [[ -n "${slug}" ]]; then
      printf '%s' "${slug}"
      return 0
    fi
  fi

  local slug
  slug="$(grep -oE '[a-z0-9]+(-[a-z0-9]+)+' <<< "${line}" | head -n 1 || true)"
  if [[ -n "${slug}" ]]; then
    printf '%s' "${slug}"
    return 0
  fi
  return 1
}

if [[ ! -f "${roadmap_path}" ]]; then
  echo "[roadmap-status-drift] missing required roadmap file: ${roadmap_path}"
  exit 1
fi
if [[ ! -f "${archive_index_path}" ]]; then
  echo "[roadmap-status-drift] missing required archive index file: ${archive_index_path}"
  exit 1
fi

if ! command -v openspec >/dev/null 2>&1; then
  echo "[roadmap-status-drift] openspec CLI is required but not found in PATH"
  exit 1
fi

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

mapfile -t archived_changes < <(
  awk -F'->' '
    /^\s*-[[:space:]]+[0-9]+[[:space:]]*->/ {
      slug=$2
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", slug)
      if (slug != "") {
        print slug
      }
    }
  ' "${archive_index_path}" | sort -u
)

mapfile -t roadmap_in_progress < <(
  awk '
    function trim(s) {
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", s)
      return s
    }
    function emit_slug(source, mode, normalized, rest, candidate, n, i, m) {
      normalized = trim(source)
      if (index(normalized, "`") > 0) {
        rest = substr(normalized, index(normalized, "`") + 1)
        if (index(rest, "`") > 0) {
          candidate = trim(substr(rest, 1, index(rest, "`") - 1))
          if (candidate != "") {
            print candidate
            return
          }
        }
      }
      n = split(normalized, m, /[^a-z0-9-]+/)
      for (i = 1; i <= n; i++) {
        if (m[i] ~ /^[a-z0-9]+(-[a-z0-9]+)+$/) {
          print m[i]
          return
        }
      }
    }
    BEGIN {
      in_status = 0
      mode = ""
    }
    {
      line = $0
      trimmed = trim(line)
      if (!in_status) {
        if (trimmed ~ /^##[[:space:]]*当前状态/) {
          in_status = 1
        }
        next
      }
      if (trimmed ~ /^##[[:space:]]+/) {
        exit
      }

      if (trimmed ~ /^-[[:space:]]*进行中：/) {
        mode = "in-progress"
        next
      }
      if (trimmed ~ /^-[[:space:]]*已归档：/) {
        mode = "archived"
        next
      }
      if (trimmed ~ /^-[[:space:]]*候选：/) {
        mode = "candidate"
        next
      }

      if (mode != "in-progress") {
        next
      }
      if (trimmed !~ /^-/) {
        next
      }
      emit_slug(trimmed, mode)
    }
  ' "${roadmap_path}" | sort -u
)

mapfile -t roadmap_archived < <(
  awk '
    function trim(s) {
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", s)
      return s
    }
    function emit_slug(source, mode, normalized, rest, candidate, n, i, m) {
      normalized = trim(source)
      if (index(normalized, "`") > 0) {
        rest = substr(normalized, index(normalized, "`") + 1)
        if (index(rest, "`") > 0) {
          candidate = trim(substr(rest, 1, index(rest, "`") - 1))
          if (candidate != "") {
            print candidate
            return
          }
        }
      }
      n = split(normalized, m, /[^a-z0-9-]+/)
      for (i = 1; i <= n; i++) {
        if (m[i] ~ /^[a-z0-9]+(-[a-z0-9]+)+$/) {
          print m[i]
          return
        }
      }
    }
    BEGIN {
      in_status = 0
      mode = ""
    }
    {
      line = $0
      trimmed = trim(line)
      if (!in_status) {
        if (trimmed ~ /^##[[:space:]]*当前状态/) {
          in_status = 1
        }
        next
      }
      if (trimmed ~ /^##[[:space:]]+/) {
        exit
      }

      if (trimmed ~ /^-[[:space:]]*进行中：/) {
        mode = "in-progress"
        next
      }
      if (trimmed ~ /^-[[:space:]]*已归档：/) {
        mode = "archived"
        next
      }
      if (trimmed ~ /^-[[:space:]]*候选：/) {
        mode = "candidate"
        next
      }

      if (mode != "archived") {
        next
      }
      if (trimmed !~ /^-/) {
        next
      }
      emit_slug(trimmed, mode)
    }
  ' "${roadmap_path}" | sort -u
)

declare -A active_set=()
declare -A archived_set=()
declare -A roadmap_in_progress_set=()
declare -A roadmap_archived_set=()

for change in "${active_changes[@]}"; do
  [[ -z "${change}" ]] && continue
  active_set["${change}"]=1
done
for change in "${archived_changes[@]}"; do
  [[ -z "${change}" ]] && continue
  archived_set["${change}"]=1
done
for change in "${roadmap_in_progress[@]}"; do
  [[ -z "${change}" ]] && continue
  roadmap_in_progress_set["${change}"]=1
done
for change in "${roadmap_archived[@]}"; do
  [[ -z "${change}" ]] && continue
  roadmap_archived_set["${change}"]=1
done

issues=()

for change in "${active_changes[@]}"; do
  [[ -z "${change}" ]] && continue
  if [[ -z "${roadmap_in_progress_set[${change}]:-}" ]]; then
    issues+=("[roadmap-status-drift] roadmap missing in-progress change from openspec list: ${change}")
  fi
done

mapfile -t roadmap_in_progress_keys < <(printf '%s\n' "${!roadmap_in_progress_set[@]}" | sed '/^$/d' | sort)
for change in "${roadmap_in_progress_keys[@]}"; do
  if [[ -n "${active_set[${change}]:-}" ]]; then
    continue
  fi
  if [[ -n "${archived_set[${change}]:-}" ]]; then
    issues+=("[roadmap-status-drift] roadmap marks archived change as in-progress: ${change}")
    continue
  fi
  issues+=("[roadmap-status-drift] roadmap in-progress entry is not active in openspec list: ${change}")
done

mapfile -t roadmap_archived_keys < <(printf '%s\n' "${!roadmap_archived_set[@]}" | sed '/^$/d' | sort)
for change in "${roadmap_archived_keys[@]}"; do
  if [[ -n "${archived_set[${change}]:-}" ]]; then
    continue
  fi
  if [[ -n "${active_set[${change}]:-}" ]]; then
    issues+=("[roadmap-status-drift] roadmap marks active change as archived: ${change}")
    continue
  fi
  issues+=("[roadmap-status-drift] roadmap archived entry is not present in archive index: ${change}")
done

if (( ${#issues[@]} > 0 )); then
  for issue in "${issues[@]}"; do
    echo "${issue}"
  done
  echo "hint: sync docs/development-roadmap.md current status with openspec list --json and openspec/changes/archive/INDEX.md."
  echo "hint: expected deterministic status authority sources are active=in-progress changes and archive index."
  exit 1
fi

echo "[openspec-roadmap-status-consistency] passed"
