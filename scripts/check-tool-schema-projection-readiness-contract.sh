#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
# Contract covers stable-window pressure, subset integrity, privacy, overflow,
# replay determinism, and Run/Stream parity.
cache_dir="${GOCACHE:-$repo_root/.gocache-tool-schema-projection-readiness-gate}"
mkdir -p "$cache_dir"
GOCACHE="$cache_dir" go test ./tool/schemaaudit ./tool/diagnosticsreplay ./tool/contributioncheck -run 'Readiness|ToolSchemaProjection' -count=1
for fixture in tool_schema_projection_readiness.v1.json tool_schema_projection_readiness_overflow.json tool_schema_projection_readiness_privacy.json; do
  test -f "tool/diagnosticsreplay/testdata/$fixture" || { echo "[tool-schema-projection-readiness-gate] missing fixture $fixture" >&2; exit 1; }
done
if rg -n 'github\.com/(openai|anthropics)|google\.golang\.org/genai|tokenizer|ModelRequest|net/http|os\.Open|os\.WriteFile|globalSelector|globalRouter|dynamicDownload|marketplaceResolve|persistRawSchema|persistPrompt|persistModelOutput|persistToolResult|RuntimeRecorder' tool/schemaaudit/readiness*.go tool/diagnosticsreplay/tool_schema_projection_readiness.go >/dev/null; then
  echo '[tool-schema-projection-readiness-gate] forbidden runtime/provider/persistence reference detected' >&2
  exit 1
fi
echo '[tool-schema-projection-readiness-gate] offline readiness contract passed'
