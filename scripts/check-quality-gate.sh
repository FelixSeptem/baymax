#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$(pwd)/.gocache"
fi
if [[ -z "${GOLANGCI_LINT_CACHE:-}" ]]; then
  export GOLANGCI_LINT_CACHE="$(pwd)/.gocache/golangci-lint"
fi
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

echo "[quality-gate] repo hygiene"
bash scripts/check-repo-hygiene.sh

echo "[quality-gate] docs consistency"
if ! bash scripts/check-docs-consistency.sh; then
  echo "[quality-gate][docs-consistency] docs consistency failed (adapter mapping or pre1 governance drift)"
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

echo "[quality-gate] runtime readiness contract suites"
if ! go test ./runtime/config ./runtime/diagnostics ./observability/event ./orchestration/composer ./integration -run 'Test(RuntimeReadiness|ReadinessAdmission|StoreRunReadiness|RuntimeRecorderA40ParserCompatibilityAdditiveNullableDefault|ComposerReadiness)' -count=1; then
  echo "[quality-gate][runtime-readiness] runtime readiness contract suites failed"
  exit 1
fi

echo "[quality-gate] diagnostics cardinality contract suites"
if ! go test ./runtime/config ./runtime/diagnostics ./observability/event ./integration -run 'Test(DiagnosticsCardinality|ManagerDiagnosticsCardinality|StoreRunCardinality|CardinalityListGovernance|RuntimeRecorderA45ParserCompatibilityAdditiveNullableDefault|DiagnosticsCardinalityContract)' -count=1; then
  echo "[quality-gate][diagnostics-cardinality] diagnostics cardinality contract suites failed"
  exit 1
fi

echo "[quality-gate] adapter-health contract suites"
if ! go test ./adapter/health ./runtime/config ./runtime/diagnostics ./observability/event ./integration/adapterconformance -run 'Test(RunnerProbe|AdapterHealthConfig|ManagerAdapterHealth|ManagerReadinessPreflightAdapterHealth|StoreRunReadinessAdditiveFieldsPersistAndReplayIdempotent|RuntimeRecorderA14ParserCompatibilityAdditiveNullableDefault|RuntimeRecorderA46ParserCompatibilityAdditiveNullableDefault|AdapterConformanceHealth(Matrix|Governance))' -count=1; then
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
  # Git Bash on Windows may hit ThreadSanitizer allocation issues; bridge to pwsh keeps the same blocking semantics.
  race_packages="$(echo "${packages}" | tr '\n' ' ')"
  pwsh -NoProfile -Command "\$env:CGO_ENABLED='1'; go test -race ${race_packages}"
else
  go test -race ${packages}
fi

echo "[quality-gate] golangci-lint"
golangci-lint run --config .golangci.yml

echo "[quality-gate] CA4 benchmark regression"
bash scripts/check-ca4-benchmark-regression.sh

echo "[quality-gate] multi-agent mainline benchmark regression"
bash scripts/check-multi-agent-performance-regression.sh

echo "[quality-gate] diagnostics query benchmark regression"
if ! bash scripts/check-diagnostics-query-performance-regression.sh; then
  echo "[quality-gate][diagnostics-query-bench] diagnostics query benchmark regression failed"
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
