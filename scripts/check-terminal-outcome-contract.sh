#!/usr/bin/env bash
set -euo pipefail

echo "[terminal-outcome-contract-gate] focused contract suites"
go test ./core/types ./tool/local ./orchestration/composer ./tool/diagnosticsreplay ./integration -run 'TerminalOutcome|RecoveryErrorProjects|ReplayFixture' -count=1

echo "[terminal-outcome-contract-gate] source ownership assertions"
if rg -n 'retry exhausted|resume may be possible' core/runner core/types tool/local --glob '*.go' | rg 'TerminalOutcome|terminal_outcome_projection'; then
  echo "projection must not synthesize retry/resume from free-form messages" >&2
  exit 1
fi
if rg -n 'terminal.*working|working.*terminal' core/runner core/types orchestration --glob '*.go' | rg 'TerminalOutcome|TerminalOutcomeArbiter'; then
  echo "terminal outcome must not transition back to working" >&2
  exit 1
fi

echo "[terminal-outcome-contract-gate] done"
