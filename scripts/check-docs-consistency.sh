#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"
if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="${repo_root}/.gocache"
fi

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

echo "[docs-consistency] semantic labeling governance"
if ! bash scripts/check-semantic-labeling-governance.sh; then
  echo "[docs-consistency][semantic-labeling-governance] semantic labeling governance failed"
  exit 1
fi

echo "[docs-consistency] openspec roadmap status consistency"
if ! bash scripts/check-openspec-roadmap-status-consistency.sh; then
  echo "[docs-consistency][openspec-roadmap-status-consistency] openspec roadmap status consistency failed"
  exit 1
fi

offline_cache_issues=()
mapfile -t offline_tracked < <(git ls-files -- examples/adapters/_a23-offline-work || true)
for path in "${offline_tracked[@]}"; do
  [[ -z "${path}" ]] && continue
  case "${path}" in
    "examples/adapters/_a23-offline-work/.gitkeep"|"examples/adapters/_a23-offline-work/README.md")
      ;;
    *)
      offline_cache_issues+=("${path}")
      ;;
  esac
done
if (( ${#offline_cache_issues[@]} > 0 )); then
  echo "[offline-scaffold-cache] tracked offline cache artifacts must be removed: ${offline_cache_issues[*]}"
  exit 1
fi

repo_hygiene_issues=()
mapfile -t untracked_paths < <(git ls-files --others --exclude-standard || true)
for path in "${untracked_paths[@]}"; do
  [[ -z "${path}" ]] && continue
  case "${path}" in
    *.go.[0-9]*|*.tmp|*.bak|*~)
      repo_hygiene_issues+=("${path}")
      ;;
  esac
done
if (( ${#repo_hygiene_issues[@]} > 0 )); then
  echo "[repo-hygiene] temporary artifacts must be removed: ${repo_hygiene_issues[*]}"
  exit 1
fi

pre1_issues=()
roadmap_doc="$(cat docs/development-roadmap.md)"
versioning_doc="$(cat docs/versioning-and-compatibility.md)"

for marker in "版本阶段口径（延续 0.x）" '不做 `1.0.0` / prod-ready 承诺' "允许新增能力型提案" "新增提案准入规则（0.x 阶段）"; do
  if [[ "${roadmap_doc}" != *"${marker}"* ]]; then
    pre1_issues+=("docs/development-roadmap.md missing marker: ${marker}")
  fi
done

for marker in '`Why now`' "风险" "回滚" "文档影响" "验证命令"; do
  if [[ "${roadmap_doc}" != *"${marker}"* ]]; then
    pre1_issues+=("docs/development-roadmap.md missing proposal admission field marker: ${marker}")
  fi
done

for marker in "契约一致性" "可靠性与安全" "质量门禁回归治理" "外部接入 DX"; do
  if [[ "${roadmap_doc}" != *"${marker}"* ]]; then
    pre1_issues+=("docs/development-roadmap.md missing bounded objective category: ${marker}")
  fi
done

for marker in "长期方向（不进入近期主线）" "平台化控制面" "跨租户全局调度与控制平面" "市场化/托管化 adapter registry 能力"; do
  if [[ "${roadmap_doc}" != *"${marker}"* ]]; then
    pre1_issues+=("docs/development-roadmap.md missing long-term deferral marker: ${marker}")
  fi
done

for marker in 'pre-`1.0.0`' 'does **not** imply `1.0.0/prod-ready` commitments' "Pre-1 Proposal Admission Baseline" 'Capability additions are allowed in `0.x`'; do
  if [[ "${versioning_doc}" != *"${marker}"* ]]; then
    pre1_issues+=("docs/versioning-and-compatibility.md missing marker: ${marker}")
  fi
done

for marker in "版本阶段快照" '`0.x` pre-1 阶段' '不做 `1.0.0/prod-ready` 承诺' '`0.x` 阶段允许新增能力型提案'; do
  if [[ "${readme}" != *"${marker}"* ]]; then
    pre1_issues+=("README.md missing pre-1 release snapshot marker: ${marker}")
  fi
done

if (( ${#pre1_issues[@]} > 0 )); then
  echo "[pre1-governance] missing or stale pre-1 governance entries: ${pre1_issues[*]}"
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
for marker in "MCP adapter template" "Model provider adapter template" "Tool adapter template" "linux_nsjail" "linux_bwrap" "oci_runtime" "windows_job" "onboarding skeleton" "check-adapter-conformance.ps1" "check-adapter-conformance.sh" "check-sandbox-adapter-conformance-contract.ps1" "check-sandbox-adapter-conformance-contract.sh"; do
  if [[ "${template_index}" != *"${marker}"* ]]; then
    adapter_issues+=("docs/external-adapter-template-index.md missing marker: ${marker}")
  fi
done

mapping_doc="$(cat docs/adapter-migration-mapping.md)"
for marker in "capability-domain" "code-snippet" "previous pattern" "recommended pattern" "compatibility notes" "rollback notes" "conformance suite id" "additive + nullable + default + fail-fast" "check-adapter-conformance.ps1" "check-adapter-conformance.sh" "check-sandbox-adapter-conformance-contract.ps1" "check-sandbox-adapter-conformance-contract.sh"; do
  if [[ "${mapping_doc}" != *"${marker}"* ]]; then
    adapter_issues+=("docs/adapter-migration-mapping.md missing marker: ${marker}")
  fi
done

for path in "scripts/check-adapter-conformance.sh" "scripts/check-adapter-conformance.ps1" "scripts/check-sandbox-adapter-conformance-contract.sh" "scripts/check-sandbox-adapter-conformance-contract.ps1"; do
  if [[ ! -f "${path}" ]]; then
    adapter_issues+=("missing adapter conformance script: ${path}")
  fi
done

if (( ${#adapter_issues[@]} > 0 )); then
  echo "[adapter-docs] missing or stale adapter template/mapping entries: ${adapter_issues[*]}"
  exit 1
fi

go test ./tool/contributioncheck -run '^(TestMainlineContractIndexReferencesExistingTests|TestAdapterOnboardingDocsConsistency|TestPre1GovernanceDocsConsistency|TestValidatePre1GovernanceDocsDetectsStageConflict|TestReleaseStatusParityDocsConsistency|TestValidateStatusParityDetectsConflict|TestCoreModuleReadmeRichnessBaseline|TestValidateCoreModuleReadmeRichnessDetectsMissingSection|TestValidateCoreModuleReadmeRichnessDetectsCanonicalPathDrift|TestValidateStatusParitySupportsSlugSnapshotFormat|TestDocsConsistencyRepoHygieneTempArtifacts|TestRoadmapStatusConsistencyGateScriptParity|TestExampleImpactDeclarationGateScriptParity|TestDocsConsistencyIncludesRoadmapStatusConsistencyGate|TestQualityGateIncludesExampleImpactDeclarationGate|TestCIGovernanceRequiredCheckCandidates)$' -count=1

echo "Docs consistency check passed."
