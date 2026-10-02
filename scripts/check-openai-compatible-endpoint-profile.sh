#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"
if [[ -z "${GOCACHE:-}" ]]; then
  if command -v cygpath >/dev/null 2>&1; then
    GOCACHE="$(cygpath -m "${repo_root}/.gocache-openai-compatible-profile")"
  else
    GOCACHE="${repo_root}/.gocache-openai-compatible-profile"
  fi
fi
export GOCACHE
fixture="tool/diagnosticsreplay/testdata/openai_endpoint_profile_conformance.v1.json"
# ReplayOpenAIEndpointProfileFixtureJSON is exercised by the package tests below.
[[ -f "${fixture}" ]] || { echo "openai_endpoint_profile_schema_drift: missing fixture ${fixture}" >&2; exit 1; }
[[ "$(wc -c < "${fixture}")" -le 1048576 ]] || { echo "openai_endpoint_profile_overflow_drift: fixture exceeds 1 MiB" >&2; exit 1; }
grep -q '"openai-compatible-endpoint-profile-conformance.v1"' "${fixture}" || { echo "openai_endpoint_profile_schema_drift: unsupported fixture version" >&2; exit 1; }
if grep -Eiq 'raw_prompt|raw_reasoning|raw_payload|credential|password|bearer token' "${fixture}"; then
  echo "openai_endpoint_profile_privacy_drift: sensitive fixture marker" >&2
  exit 1
fi
go test ./model/conformance ./model/openai ./tool/diagnosticsreplay ./tool/contributioncheck -run 'OpenAIEndpointProfile|EndpointProfile|Profile' -count=1
echo "[openai-compatible-endpoint-profile] offline fixture, capability, parity, privacy, and review-only gate passed (no network)"
