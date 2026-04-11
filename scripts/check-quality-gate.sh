#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

if [[ "${OSTYPE:-}" == msys* || "${OSTYPE:-}" == cygwin* || "${MSYSTEM:-}" == MINGW64 || "${MSYSTEM:-}" == MINGW32 ]]; then
  if command -v pwsh >/dev/null 2>&1; then
    echo "[quality-gate] windows bash detected; delegating to scripts/check-quality-gate.ps1 for timeout and parallel semantics"
    exec pwsh -File scripts/check-quality-gate.ps1
  fi
fi

is_writable_dir() {
  local path="${1:-}"
  [[ -n "${path}" ]] || return 1
  mkdir -p "${path}" 2>/dev/null || return 1
  local probe="${path}/._write_probe_$$"
  : > "${probe}" 2>/dev/null || return 1
  rm -f "${probe}" 2>/dev/null || true
  return 0
}

ensure_writable_cache_env() {
  local env_name="$1"
  local fallback_path="$2"
  local current="${!env_name:-}"
  if is_writable_dir "${current}"; then
    return 0
  fi
  if ! is_writable_dir "${fallback_path}"; then
    echo "[quality-gate] unable to prepare writable cache directory for ${env_name} at ${fallback_path}" >&2
    exit 1
  fi
  export "${env_name}=${fallback_path}"
}

ensure_writable_cache_env "GOCACHE" "${REPO_ROOT}/.gocache"
ensure_writable_cache_env "GOLANGCI_LINT_CACHE" "${REPO_ROOT}/.gocache/golangci-lint"

if [[ -z "${GOPROXY:-}" ]]; then
  export GOPROXY="https://proxy.golang.org,direct"
fi
if [[ -z "${GOSUMDB:-}" ]]; then
  export GOSUMDB="sum.golang.org"
fi
if [[ -z "${GOVULNDB:-}" ]]; then
  export GOVULNDB="https://vuln.go.dev"
fi
if [[ -z "${CGO_ENABLED:-}" ]]; then
  export CGO_ENABLED=1
fi
if [[ "${GODEBUG:-}" != *"goindex="* ]]; then
  if [[ -z "${GODEBUG:-}" ]]; then
    export GODEBUG="goindex=0"
  else
    export GODEBUG="${GODEBUG},goindex=0"
  fi
fi

echo "[quality-gate] repo hygiene"
bash scripts/check-repo-hygiene.sh

echo "[quality-gate] docs consistency"
if ! bash scripts/check-docs-consistency.sh; then
  echo "[quality-gate][docs-consistency] docs consistency failed (adapter mapping or pre1 governance drift)"
  exit 1
fi

echo "[quality-gate] openspec example impact declaration"
if ! bash scripts/check-openspec-example-impact-declaration.sh; then
  echo "[quality-gate][openspec-example-impact-declaration] openspec example impact declaration failed"
  exit 1
fi

echo "[quality-gate] agent mode pattern coverage"
if ! bash scripts/check-agent-mode-pattern-coverage.sh; then
  echo "[quality-gate][agent-mode-pattern-coverage] agent mode pattern coverage failed"
  exit 1
fi

echo "[quality-gate] agent mode examples smoke"
if ! bash scripts/check-agent-mode-examples-smoke.sh; then
  echo "[quality-gate][agent-mode-examples-smoke] agent mode examples smoke failed"
  exit 1
fi

echo "[quality-gate] agent mode smoke stability governance"
if ! bash scripts/check-agent-mode-smoke-stability-governance.sh; then
  echo "[quality-gate][agent-mode-smoke-stability-governance] agent mode smoke stability governance failed"
  exit 1
fi

echo "[quality-gate] agent mode migration playbook consistency"
if ! bash scripts/check-agent-mode-migration-playbook-consistency.sh; then
  echo "[quality-gate][agent-mode-migration-playbook-consistency] agent mode migration playbook consistency failed"
  exit 1
fi

echo "[quality-gate] agent mode legacy todo cleanup"
if ! bash scripts/check-agent-mode-legacy-todo-cleanup.sh; then
  echo "[quality-gate][agent-mode-legacy-todo-cleanup] agent mode legacy todo cleanup failed"
  exit 1
fi

echo "[quality-gate] agent mode real runtime semantic contract"
if ! bash scripts/check-agent-mode-real-runtime-semantic-contract.sh; then
  echo "[quality-gate][agent-mode-real-runtime-semantic-contract] agent mode real runtime semantic contract failed"
  exit 1
fi

echo "[quality-gate] agent mode readme runtime sync contract"
if ! bash scripts/check-agent-mode-readme-runtime-sync-contract.sh; then
  echo "[quality-gate][agent-mode-readme-runtime-sync-contract] agent mode README runtime sync contract failed"
  exit 1
fi

echo "[quality-gate] agent mode anti-template contract"
if ! bash scripts/check-agent-mode-anti-template-contract.sh; then
  echo "[quality-gate][agent-mode-anti-template-contract] agent mode anti-template contract failed"
  exit 1
fi

echo "[quality-gate] agent mode doc-first delivery contract"
if ! bash scripts/check-agent-mode-doc-first-delivery-contract.sh; then
  echo "[quality-gate][agent-mode-doc-first-delivery-contract] agent mode doc-first delivery contract failed"
  exit 1
fi

echo "[quality-gate] a64 impacted gate selection"
if ! bash scripts/check-a64-impacted-gate-selection.sh; then
  echo "[quality-gate][a64-impacted-gate-selection] a64 impacted gate selection failed"
  exit 1
fi

echo "[quality-gate] a64 harnessability scorecard"
if ! bash scripts/check-a64-harnessability-scorecard.sh; then
  echo "[quality-gate][a64-harnessability-scorecard] a64 harnessability scorecard failed"
  exit 1
fi

echo "[quality-gate] go file line budget governance"
if ! bash scripts/check-go-file-line-budget.sh; then
  echo "[quality-gate][go-file-line-budget] go file line budget governance failed"
  exit 1
fi

echo "[quality-gate] go split semantic equivalence strong checks"
if ! bash scripts/check-go-split-semantic-equivalence.sh; then
  echo "[quality-gate][go-split-strong-check] go split semantic equivalence strong checks failed"
  exit 1
fi

echo "[quality-gate] semantic labeling governance"
if ! bash scripts/check-semantic-labeling-governance.sh; then
  echo "[quality-gate][semantic-labeling-governance] semantic labeling governance failed"
  exit 1
fi

echo "[quality-gate] canonical mailbox entrypoints"
if ! bash scripts/check-canonical-mailbox-entrypoints.sh; then
  echo "[quality-gate][canonical-mailbox-entrypoints] canonical mailbox invoke guard failed"
  exit 1
fi

echo "[quality-gate] multi-agent shared contract suites"
if ! bash scripts/check-multi-agent-shared-contract.sh; then
  echo "[quality-gate][multi-agent-shared-contract] shared contract suites failed"
  exit 1
fi

echo "[quality-gate] state snapshot contract suites"
if ! bash scripts/check-state-snapshot-contract.sh; then
  echo "[quality-gate][state-snapshot-contract] state snapshot contract suites failed"
  exit 1
fi

echo "[quality-gate] runtime readiness + explainability + version governance contract suites"
if ! go test ./runtime/config ./runtime/diagnostics ./observability/event ./orchestration/composer ./integration -run 'Test(RuntimeReadiness|ReadinessAdmission|ArbitrationVersionGovernanceContract|StoreRunReadiness|StoreRunArbitrationVersionGovernance|RuntimeRecorderReadinessParserCompatibilityAdditiveNullableDefault|RuntimeRecorderArbitrationExplainabilityParserCompatibilityAdditiveNullableDefault|RuntimeRecorderArbitrationVersionGovernanceParserCompatibilityAdditiveNullableDefault|RuntimeRecorderParsesArbitrationVersionGovernanceFields|ComposerReadiness)' -count=1; then
  echo "[quality-gate][runtime-readiness] runtime readiness contract suites failed"
  exit 1
fi

echo "[quality-gate] diagnostics cardinality contract suites"
if ! go test ./runtime/config ./runtime/diagnostics ./observability/event ./integration -run 'Test(DiagnosticsCardinality|ManagerDiagnosticsCardinality|StoreRunCardinality|CardinalityListGovernance|RuntimeRecorderDiagnosticsCardinalityParserCompatibilityAdditiveNullableDefault|DiagnosticsCardinalityContract)' -count=1; then
  echo "[quality-gate][diagnostics-cardinality] diagnostics cardinality contract suites failed"
  exit 1
fi

echo "[quality-gate] adapter-health contract suites"
if ! go test ./adapter/health ./runtime/config ./runtime/diagnostics ./observability/event ./integration/adapterconformance -run 'Test(RunnerProbe|AdapterHealthConfig|ManagerAdapterHealth|ManagerReadinessPreflightAdapterHealth|StoreRunReadinessAdditiveFieldsPersistAndReplayIdempotent|RuntimeRecorderReadinessParserCompatibilityAdditiveNullableDefault|RuntimeRecorderAdapterHealthGovernanceParserCompatibilityAdditiveNullableDefault|AdapterConformanceHealth(Matrix|Governance))' -count=1; then
  echo "[quality-gate][adapter-health] adapter-health contract suites failed"
  exit 1
fi

echo "[quality-gate] mailbox runtime wiring regression"
if ! go test ./integration -run '^TestComposerContractMailboxRuntimeWiring' -count=1; then
  echo "[quality-gate][mailbox-runtime-wiring] mailbox runtime wiring regression detected"
  exit 1
fi

echo "[quality-gate] timeout resolution contract suites"
if ! go test ./integration -run '^TestTimeoutResolutionContract' -count=1; then
  echo "[quality-gate][timeout-resolution-contract] timeout resolution contract suites failed"
  exit 1
fi

echo "[quality-gate] readiness-timeout-health replay fixture suites"
if ! go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractCompositeFixture|ReplayContractPrimaryReasonArbitrationFixture|ReadinessTimeoutHealthReplayContract|PrimaryReasonArbitrationReplayContract)' -count=1; then
  echo "[quality-gate][readiness-timeout-health-replay] replay fixture suites failed"
  exit 1
fi

echo "[quality-gate] react contract suites"
if ! bash scripts/check-react-contract.sh; then
  echo "[quality-gate][react-contract] react contract suites failed"
  exit 1
fi

echo "[quality-gate] react plan notebook contract suites"
if ! bash scripts/check-react-plan-notebook-contract.sh; then
  echo "[quality-gate][react-plan-notebook-contract] react plan notebook contract suites failed"
  exit 1
fi

echo "[quality-gate] context jit organization contract suites"
if ! BAYMAX_CONTEXT_JIT_SKIP_IMPACTED_CONTRACT_SUITES=1 bash scripts/check-context-jit-organization-contract.sh; then
  echo "[quality-gate][context-jit-organization-contract] context jit organization contract suites failed"
  exit 1
fi

echo "[quality-gate] context compression production contract suites"
if ! BAYMAX_CONTEXT_COMPRESSION_SKIP_IMPACTED_CONTRACT_SUITES=1 bash scripts/check-context-compression-production-contract.sh; then
  echo "[quality-gate][context-compression-production-contract] context compression production contract suites failed"
  exit 1
fi

echo "[quality-gate] realtime protocol contract suites"
if ! bash scripts/check-realtime-protocol-contract.sh; then
  echo "[quality-gate][realtime-protocol-contract] realtime protocol contract suites failed"
  exit 1
fi

echo "[quality-gate] hooks + middleware contract suites"
if ! bash scripts/check-hooks-middleware-contract.sh; then
  echo "[quality-gate][hooks-middleware-contract] hooks + middleware contract suites failed"
  exit 1
fi

echo "[quality-gate] security sandbox contract suites"
if ! bash scripts/check-security-sandbox-contract.sh; then
  echo "[quality-gate][security-sandbox-contract] security sandbox contract suites failed"
  exit 1
fi

echo "[quality-gate] sandbox rollout governance contract suites"
if ! bash scripts/check-sandbox-rollout-governance-contract.sh; then
  echo "[quality-gate][sandbox-rollout-governance-contract] sandbox rollout governance contract suites failed"
  exit 1
fi

echo "[quality-gate] sandbox egress + adapter allowlist contract suites"
if ! bash scripts/check-sandbox-egress-allowlist-contract.sh; then
  echo "[quality-gate][sandbox-egress-allowlist-contract] sandbox egress + adapter allowlist contract suites failed"
  exit 1
fi

echo "[quality-gate] policy precedence contract suites"
if ! bash scripts/check-policy-precedence-contract.sh; then
  echo "[quality-gate][policy-precedence-contract] policy precedence contract suites failed"
  exit 1
fi

echo "[quality-gate] runtime budget admission contract suites"
if ! bash scripts/check-runtime-budget-admission-contract.sh; then
  echo "[quality-gate][runtime-budget-admission-contract] runtime budget admission contract suites failed"
  exit 1
fi

echo "[quality-gate] agent eval and tracing interop contract suites"
if ! bash scripts/check-agent-eval-and-tracing-interop-contract.sh; then
  echo "[quality-gate][agent-eval-tracing-interop-contract] agent eval and tracing interop contract suites failed"
  exit 1
fi

echo "[quality-gate] adapter conformance"
if ! bash scripts/check-adapter-conformance.sh; then
  echo "[quality-gate][adapter-conformance] adapter conformance harness failed"
  exit 1
fi

echo "[quality-gate] adapter manifest contract"
if ! bash scripts/check-adapter-manifest-contract.sh; then
  echo "[quality-gate][adapter-manifest-contract] adapter manifest contract check failed"
  exit 1
fi

echo "[quality-gate] adapter capability negotiation contract"
if ! bash scripts/check-adapter-capability-contract.sh; then
  echo "[quality-gate][adapter-capability-contract] adapter capability negotiation contract check failed"
  exit 1
fi

echo "[quality-gate] adapter contract replay"
if ! bash scripts/check-adapter-contract-replay.sh; then
  echo "[quality-gate][adapter-contract-replay] adapter contract replay check failed"
  exit 1
fi

echo "[quality-gate] sandbox adapter conformance contract"
if ! bash scripts/check-sandbox-adapter-conformance-contract.sh; then
  echo "[quality-gate][sandbox-adapter-conformance-contract] sandbox adapter conformance contract check failed"
  exit 1
fi

echo "[quality-gate] memory contract conformance"
if ! bash scripts/check-memory-contract-conformance.sh; then
  echo "[quality-gate][memory-contract] memory contract conformance check failed"
  exit 1
fi

echo "[quality-gate] memory scope and search governance contract"
if ! bash scripts/check-memory-scope-and-search-contract.sh; then
  echo "[quality-gate][memory-scope-search-contract] memory scope and search governance contract check failed"
  exit 1
fi

echo "[quality-gate] observability export and diagnostics bundle contract"
if ! bash scripts/check-observability-export-and-bundle-contract.sh; then
  echo "[quality-gate][observability-export-bundle-contract] observability export and diagnostics bundle contract check failed"
  exit 1
fi

echo "[quality-gate] adapter scaffold drift"
if ! bash scripts/check-adapter-scaffold-drift.sh; then
  echo "[quality-gate][adapter-scaffold-drift] adapter scaffold drift check failed"
  exit 1
fi

echo "[quality-gate] go test ./..."
go test ./...

echo "[quality-gate] go test -race (exclude examples packages)"
if [[ "${CGO_ENABLED}" != "1" ]]; then
  echo "[quality-gate] go test -race requires CGO_ENABLED=1"
  exit 1
fi
packages="$(go list ./... | grep -v '/examples/' || true)"
if [[ -z "${packages}" ]]; then
  echo "[quality-gate] no packages found for race tests"
  exit 1
fi
if [[ "${OSTYPE:-}" == msys* || "${OSTYPE:-}" == cygwin* || "${MSYSTEM:-}" == MINGW64 || "${MSYSTEM:-}" == MINGW32 ]]; then
  # Run race tests through cmd to avoid MSYS process model causing sporadic ThreadSanitizer allocation failures.
  cmd.exe /c "set CGO_ENABLED=1&& go test -race ./..."
else
  go test -race ${packages}
fi

echo "[quality-gate] golangci-lint"
golangci-lint run --config .golangci.yml

echo "[quality-gate] a64 semantic stability gate"
if ! bash scripts/check-a64-semantic-stability-contract.sh; then
  echo "[quality-gate][a64-semantic-stability] a64 semantic stability gate failed"
  exit 1
fi

echo "[quality-gate] a64 performance regression gate"
if ! bash scripts/check-a64-performance-regression.sh; then
  echo "[quality-gate][a64-performance-regression] a64 performance regression gate failed"
  exit 1
fi

echo "[quality-gate] full-chain example smoke"
bash scripts/check-full-chain-example-smoke.sh

scan_mode="${BAYMAX_SECURITY_SCAN_MODE:-strict}"
govulncheck_enabled="${BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED:-true}"
if [[ "${govulncheck_enabled}" == "true" ]]; then
  echo "[quality-gate] govulncheck (mode=${scan_mode})"
  should_bypass_proxy_for_govulncheck() {
    local proxy_value
    proxy_value="$(echo "${1:-}" | tr '[:upper:]' '[:lower:]')"
    [[ "${proxy_value}" == *"127.0.0.1:9"* || "${proxy_value}" == *"localhost:9"* || "${proxy_value}" == *"[::1]:9"* ]]
  }
  govuln_env_prefix=()
  if should_bypass_proxy_for_govulncheck "${HTTP_PROXY:-}" ||
    should_bypass_proxy_for_govulncheck "${HTTPS_PROXY:-}" ||
    should_bypass_proxy_for_govulncheck "${ALL_PROXY:-}" ||
    should_bypass_proxy_for_govulncheck "${http_proxy:-}" ||
    should_bypass_proxy_for_govulncheck "${https_proxy:-}" ||
    should_bypass_proxy_for_govulncheck "${all_proxy:-}"; then
    echo "[quality-gate] detected placeholder proxy for govulncheck; run with direct connection"
    govuln_env_prefix=(env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY -u http_proxy -u https_proxy -u all_proxy -u GIT_HTTP_PROXY -u GIT_HTTPS_PROXY)
  fi
  govuln_cmd=(govulncheck ./...)
  if ! command -v govulncheck >/dev/null 2>&1; then
    govuln_cmd=(go run golang.org/x/vuln/cmd/govulncheck@latest ./...)
  fi
  if ! "${govuln_env_prefix[@]}" "${govuln_cmd[@]}"; then
    if [[ "${scan_mode}" == "warn" ]]; then
      echo "[quality-gate] govulncheck found issues but mode=warn; continue"
    else
      echo "[quality-gate] govulncheck found issues; mode=strict fails"
      exit 1
    fi
  fi
else
  echo "[quality-gate] govulncheck disabled by BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED"
fi

echo "[quality-gate] done"
