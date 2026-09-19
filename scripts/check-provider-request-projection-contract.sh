#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"; cd "${repo_root}"
# Git Bash reports MSYS-style paths (/d/...) which the Go toolchain on Windows
# rejects as "not an absolute path"; normalize through cygpath when present.
if [[ -z "${GOCACHE:-}" ]]; then
  if command -v cygpath >/dev/null 2>&1; then
    GOCACHE="$(cygpath -m "${repo_root}/.gocache")"
  else
    GOCACHE="${repo_root}/.gocache"
  fi
fi
export GOCACHE

fixture="tool/diagnosticsreplay/testdata/model_request_projection.v1.json"
fixture_version="provider_request_projection.v1"
contract_file="model/conformance/request_projection.go"
taxonomy_doc="model/README.md"

fail() {
  echo "$1" >&2
  exit 1
}

echo "[provider-request-projection] fixture presence, version, and bounds"
[[ -f "${fixture}" ]] || fail "provider_request_schema_drift: missing fixture ${fixture}"
[[ "$(wc -c < "${fixture}")" -le 2097152 ]] || fail "provider_request_overflow_drift: fixture exceeds 2 MiB"
grep -q "\"${fixture_version}\"" "${fixture}" || fail "provider_request_schema_drift: fixture does not declare ${fixture_version}"

echo "[provider-request-projection] stable classification taxonomy"
request_projection_codes=(
  provider_request_schema_drift
  provider_request_role_projection_drift
  provider_request_tool_result_native_drift
  provider_request_part_ordering_drift
  provider_request_stable_prefix_drift
  provider_request_tool_order_drift
  provider_request_capability_projection_drift
  provider_request_run_stream_parity_drift
  provider_cache_usage_projection_drift
  provider_request_overflow_drift
  provider_request_contract_drift
)
for code in "${request_projection_codes[@]}"; do
  grep -q "\"${code}\"" "${contract_file}" \
    || fail "provider_request_contract_drift: ${code} missing from ${contract_file}"
  grep -q "${code}" "${taxonomy_doc}" \
    || fail "provider_request_contract_drift: ${code} missing from ${taxonomy_doc}"
done

echo "[provider-request-projection] adapter ownership, provider neutrality, no raw payload"
go test ./tool/contributioncheck -run 'TestProviderRequestProjectionContractBoundary' -count=1

echo "[provider-request-projection] adapter SDK request projection shape"
go test ./model/conformance ./model/openai ./model/anthropic ./model/gemini -run 'RequestProjection|CacheUsageProjection|ProjectionAudit' -count=1

echo "[provider-request-projection] offline replay idempotency"
go test ./tool/diagnosticsreplay -run 'ProviderRequestProjection' -count=2

echo "[provider-request-projection-contract] passed"
