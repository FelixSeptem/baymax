#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
if [[ -z "${GOCACHE:-}" ]]; then export GOCACHE="$repo_root/.gocache"; fi
export GOTELEMETRY="off"

echo "[sandbox-lifecycle-success-cost-evidence] running offline contract checks"
go test ./tool/diagnosticsreplay ./tool/contributioncheck \
  -run 'SandboxLifecycleSuccessCostEvidence|SandboxLifecycleSuccessCost|QualityGateIncludesSandboxLifecycle' -count=1

fixture="tool/diagnosticsreplay/testdata/sandbox_lifecycle_success_cost_evidence.v1.json"
[[ -f "$fixture" ]] || { echo "[sandbox-lifecycle-success-cost-evidence] missing fixture" >&2; exit 1; }
[[ "$(wc -c < "$fixture")" -le 1048576 ]] || { echo "[sandbox-lifecycle-success-cost-evidence] fixture exceeds 1 MiB" >&2; exit 1; }
if rg -n -i 'endpoint|sk-|token|raw_response|raw_payload|password|secret|command' "$fixture"; then
  echo "[sandbox-lifecycle-success-cost-evidence] fixture contains forbidden material" >&2
  exit 1
fi
if rg -n --glob '*.go' 'sandbox-exec|bubblewrap|net/http|os/exec|security\.sandbox' tool/diagnosticsreplay/sandbox_lifecycle_success_cost_evidence.go; then
  echo "[sandbox-lifecycle-success-cost-evidence] forbidden execution coupling detected" >&2
  exit 1
fi
echo "[sandbox-lifecycle-success-cost-evidence] passed"
