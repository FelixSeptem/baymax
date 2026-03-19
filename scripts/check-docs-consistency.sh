#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

readme="$(cat README.md)"
mapfile -t readme_doc_refs < <(grep -oE 'docs/[A-Za-z0-9\-]+\.md' README.md || true)

missing=()
for ref in "${readme_doc_refs[@]}"; do
  [[ -z "${ref}" ]] && continue
  if [[ ! -f "${ref}" ]]; then
    missing+=("${ref}")
  fi
done

if (( ${#missing[@]} > 0 )); then
  echo "Missing docs references in README: ${missing[*]}"
  exit 1
fi

cfg_doc="$(cat docs/runtime-config-diagnostics.md)"
if [[ "${cfg_doc}" != *"迁移映射"* ]]; then
  echo "docs/runtime-config-diagnostics.md must include migration mapping section."
  exit 1
fi

boundary_doc="$(cat docs/runtime-module-boundaries.md)"
if [[ "${boundary_doc}" != *"依赖方向"* ]]; then
  echo "docs/runtime-module-boundaries.md must include dependency direction section."
  exit 1
fi

adapter_issues=()
adapter_docs=(
  "docs/external-adapter-template-index.md"
  "docs/adapter-migration-mapping.md"
)
for path in "${adapter_docs[@]}"; do
  if [[ ! -f "${path}" ]]; then
    adapter_issues+=("missing required adapter doc file: ${path}")
    continue
  fi
  if [[ "${readme}" != *"${path}"* ]]; then
    adapter_issues+=("README missing adapter doc link: ${path}")
  fi
done

api_ref="$(cat docs/api-reference-d1.md)"
for path in "${adapter_docs[@]}"; do
  if [[ "${api_ref}" != *"${path}"* ]]; then
    adapter_issues+=("docs/api-reference-d1.md missing adapter doc link: ${path}")
  fi
done

for marker in "MCP adapter template" "Model provider adapter template" "Tool adapter template"; do
  if [[ "${api_ref}" != *"${marker}"* ]]; then
    adapter_issues+=("docs/api-reference-d1.md missing adapter onboarding category navigation.")
    break
  fi
done

template_index="$(cat docs/external-adapter-template-index.md)"
for marker in "MCP adapter template" "Model provider adapter template" "Tool adapter template" "onboarding skeleton" "check-adapter-conformance.ps1" "check-adapter-conformance.sh"; do
  if [[ "${template_index}" != *"${marker}"* ]]; then
    adapter_issues+=("docs/external-adapter-template-index.md missing marker: ${marker}")
  fi
done

mapping_doc="$(cat docs/adapter-migration-mapping.md)"
for marker in "capability-domain" "code-snippet" "previous pattern" "recommended pattern" "compatibility notes" "additive + nullable + default + fail-fast" "check-adapter-conformance.ps1" "check-adapter-conformance.sh"; do
  if [[ "${mapping_doc}" != *"${marker}"* ]]; then
    adapter_issues+=("docs/adapter-migration-mapping.md missing marker: ${marker}")
  fi
done

for path in "scripts/check-adapter-conformance.sh" "scripts/check-adapter-conformance.ps1"; do
  if [[ ! -f "${path}" ]]; then
    adapter_issues+=("missing adapter conformance script: ${path}")
  fi
done

if (( ${#adapter_issues[@]} > 0 )); then
  echo "[adapter-docs] missing or stale adapter template/mapping entries: ${adapter_issues[*]}"
  exit 1
fi

go test ./tool/contributioncheck -run '^(TestMainlineContractIndexReferencesExistingTests|TestAdapterOnboardingDocsConsistency)$' -count=1

echo "Docs consistency check passed."
