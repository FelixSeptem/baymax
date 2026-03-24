#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$(pwd)/.gocache"
fi
if [[ -z "${GOLANGCI_LINT_CACHE:-}" ]]; then
  export GOLANGCI_LINT_CACHE="$(pwd)/.gocache/golangci-lint"
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
if ! go test ./runtime/config ./runtime/diagnostics ./orchestration/composer ./integration -run 'Test(RuntimeReadiness|StoreRunReadiness|ComposerReadiness)' -count=1; then
  echo "[quality-gate][runtime-readiness] runtime readiness contract suites failed"
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

echo "[quality-gate] full-chain example smoke"
bash scripts/check-full-chain-example-smoke.sh

scan_mode="${BAYMAX_SECURITY_SCAN_MODE:-strict}"
govulncheck_enabled="${BAYMAX_SECURITY_SCAN_GOVULNCHECK_ENABLED:-true}"
if [[ "${govulncheck_enabled}" == "true" ]]; then
  echo "[quality-gate] govulncheck (mode=${scan_mode})"
  govuln_cmd=(govulncheck ./...)
  if ! command -v govulncheck >/dev/null 2>&1; then
    govuln_cmd=(go run golang.org/x/vuln/cmd/govulncheck@latest ./...)
  fi
  if ! "${govuln_cmd[@]}"; then
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
