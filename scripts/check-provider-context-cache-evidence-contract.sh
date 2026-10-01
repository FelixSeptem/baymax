#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
cache_dir="${GOCACHE:-$repo_root/.gocache-provider-context-cache-evidence-gate}"
mkdir -p "$cache_dir"

GOCACHE="$cache_dir" go test ./model/conformance ./tool/diagnosticsreplay ./tool/contributioncheck -run 'ProviderContextCache|AdmitProviderContextCache' -count=1
test -f "tool/diagnosticsreplay/testdata/provider_context_cache_evidence.v1.json" || { echo "[provider-context-cache-evidence-gate] missing fixture" >&2; exit 1; }

if rg -n 'github\.com/(openai|anthropics)|google\.golang\.org/genai|net/http|RuntimeRecorder|ModelRequest|raw_prompt|raw_reasoning' \
  model/conformance/provider_context_cache_evidence.go \
  tool/diagnosticsreplay/provider_context_cache_evidence.go >/dev/null; then
  echo "[provider-context-cache-evidence-gate] forbidden provider/runtime/raw payload reference detected" >&2
  exit 1
fi

echo "[provider-context-cache-evidence-gate] offline evidence, privacy, replay, and review-only route passed"
