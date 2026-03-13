#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$(pwd)/.gocache"
fi
if [[ -z "${GOLANGCI_LINT_CACHE:-}" ]]; then
  export GOLANGCI_LINT_CACHE="$(pwd)/.gocache/golangci-lint"
fi

echo "[quality-gate] repo hygiene"
bash scripts/check-repo-hygiene.sh

echo "[quality-gate] go test ./..."
go test ./...

echo "[quality-gate] go test -race (exclude examples packages)"
packages="$(go list ./... | grep -v '/examples/' || true)"
if [[ -z "${packages}" ]]; then
  echo "[quality-gate] no packages found for race tests"
  exit 1
fi
go test -race ${packages}

echo "[quality-gate] golangci-lint"
golangci-lint run --config .golangci.yml

echo "[quality-gate] CA4 benchmark regression"
bash scripts/check-ca4-benchmark-regression.sh

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
