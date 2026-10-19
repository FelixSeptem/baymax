#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

echo '[mailbox-plan-bridge-contract] owner delegation, lifecycle, parity and reference-only replay'
go test ./orchestration/planbridge ./orchestration/mailbox ./integration ./tool/diagnosticsreplay -run 'Test(MailboxPlan|HostPlanOutcome|HostExecutionPlanMailboxAsync)' -count=1

pattern='github.com/FelixSeptem/baymax/(observability|mcp/)|^[[:space:]]*go[[:space:]]+|\.(Run|Stream|Resume|Retry|Cancel|Record|AppendEvent)\('
if rg -n -- "$pattern" tool/diagnosticsreplay/mailbox_plan_bridge.go; then
  echo '[mailbox-plan-bridge-contract] replay owns an executable surface' >&2
  exit 1
fi

if BAYMAX_MAILBOX_PLAN_BRIDGE_FIXTURE=testdata/mailbox_plan_bridge.invalid.json go test ./tool/diagnosticsreplay -run TestMailboxPlanBridgeReplayGoldenAndDrift -count=1; then
  echo '[mailbox-plan-bridge-contract] invalid replay fixture unexpectedly passed' >&2
  exit 1
fi

echo '[mailbox-plan-bridge-contract] passed'
