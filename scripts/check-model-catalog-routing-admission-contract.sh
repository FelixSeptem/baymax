#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
if [[ -z "${GOCACHE:-}" ]]; then export GOCACHE="$repo_root/.gocache"; fi
export GOTELEMETRY="off"

echo "[model-catalog-routing-admission] running contract checks"
go test ./model/catalog ./tool/diagnosticsreplay ./tool/contributioncheck \
  -run 'ModelCatalogRoutingAdmission|RoutingAdmission' -count=1

fixture="tool/diagnosticsreplay/testdata/model_catalog_routing_admission.v1.json"
[[ -f "$fixture" ]] || { echo "[model-catalog-routing-admission] missing fixture: $fixture" >&2; exit 1; }
if [[ "$(wc -c < "$fixture")" -gt 2097152 ]]; then
  echo "[model-catalog-routing-admission] fixture exceeds 2 MiB" >&2
  exit 1
fi
if rg -n -i 'endpoint|sk-|token|raw_response|raw_payload' "$fixture"; then
  echo "[model-catalog-routing-admission] fixture contains forbidden material" >&2
  exit 1
fi
if rg -n --glob '*.go' 'github.com/openai/openai-go|github.com/anthropics/anthropic-sdk-go|google.golang.org/genai|net/http' model/catalog tool/diagnosticsreplay; then
  echo "[model-catalog-routing-admission] forbidden coupling detected" >&2
  exit 1
fi
echo "[model-catalog-routing-admission] passed"
