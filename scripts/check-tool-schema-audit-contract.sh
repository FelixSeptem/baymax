#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
# Contract suite covers fixture privacy/bounds, pressure/quality, strategy drift,
# corpus advisory isolation, ReplayJSON idempotency, and Run/Stream parity.
cache_dir="${GOCACHE:-$repo_root/.gocache-tool-schema-audit-gate}"
mkdir -p "$cache_dir"
GOCACHE="$cache_dir" go test ./tool/schemaaudit ./tool/diagnosticsreplay ./tool/contributioncheck -count=1

for fixture in synthetic_within_budget.json synthetic_pressure_only.json synthetic_quality_only.json synthetic_pressure_quality.json synthetic_gold_conflict.json synthetic_overflow.json synthetic_strategy_drift.json corpus_advisory_optional.json; do
  test -f "tool/schemaaudit/testdata/$fixture" || { echo "[schema-audit-gate] missing fixture $fixture" >&2; exit 1; }
done

if rg -n 'github\.com/(openai|anthropics)|google\.golang\.org/genai|tokenizer|ModelRequest|registry\.Register|net/http|os\.Open|os\.WriteFile|globalSelector|globalRouter|dynamicDownload|marketplaceResolve|persistRawSchema|persistPrompt|persistModelOutput|persistToolResult' tool/schemaaudit/*.go tool/diagnosticsreplay/schema_audit.go >/dev/null; then
  echo "[schema-audit-gate] forbidden runtime/provider/persistence reference detected" >&2
  exit 1
fi
echo "[schema-audit-gate] offline schema pressure/selection audit contract passed"
