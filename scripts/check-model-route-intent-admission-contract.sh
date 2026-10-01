#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
if [[ -z "${GOCACHE:-}" ]]; then export GOCACHE="$repo_root/.gocache"; fi
export GOTELEMETRY="off"

echo "[model-route-intent-admission] running contract checks"
go test ./model/catalog ./tool/diagnosticsreplay ./tool/contributioncheck \
  -run 'ModelRouteIntentAdmission|RouteIntentAdmission' -count=1

fixture="tool/diagnosticsreplay/testdata/model_route_intent_admission.v1.json"
[[ -f "$fixture" ]] || { echo "[model-route-intent-admission] missing fixture" >&2; exit 1; }
[[ "$(wc -c < "$fixture")" -le 2097152 ]] || { echo "[model-route-intent-admission] fixture exceeds 2 MiB" >&2; exit 1; }
if rg -n -i 'endpoint|sk-|token|raw_response|raw_payload|password|secret' "$fixture"; then
  echo "[model-route-intent-admission] fixture contains forbidden material" >&2
  exit 1
fi
if rg -n --glob '*.go' 'github.com/openai/openai-go|github.com/anthropics/anthropic-sdk-go|google.golang.org/genai|net/http' model/catalog tool/diagnosticsreplay; then
  echo "[model-route-intent-admission] forbidden coupling detected" >&2
  exit 1
fi
echo "[model-route-intent-admission] passed"
