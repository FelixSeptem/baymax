#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
if [[ -z "${GOCACHE:-}" ]]; then export GOCACHE="$repo_root/.gocache"; fi
export GOSUMDB="${GOSUMDB:-sum.golang.org}"
export GOTELEMETRY="${GOTELEMETRY:-off}"

echo "[provider-model] running admission contract checks"
go test ./model/catalog ./runtime/config ./runtime/diagnostics ./observability/event ./orchestration/composer ./tool/diagnosticsreplay \
  -run 'ProviderCatalog|ProviderModel|ProviderAdmission|RunRecordProviderAdmission|RuntimeRecorderProjectsRedacted' -count=1

if rg -n -i --glob '*.go' 'remote discovery|credential material|background refresh|global mutable registry' model/catalog runtime/config orchestration/composer; then
  echo "[provider-model] forbidden admission coupling detected" >&2
  exit 1
fi
if rg -n 'github.com/.*/(anthropic|openai|gemini|genai)' context | grep -v 'context/assembler/embedding_adapter.go'; then
  echo "[provider-model] Provider SDK import found under context/*" >&2
  exit 1
fi
echo "[provider-model] passed"
