#!/usr/bin/env bash
# Gate: redis cross-instance coordination contract (shell/PowerShell parity).
# Covers the coordinator, mailbox redis store, cursor store and instance
# registry contracts on the miniredis substitute. When BAYMAX_TEST_REDIS_ADDR
# is configured, the real-redis integration tests run against that server too.
set -euo pipefail
cd "$(dirname "$0")/.."

go test ./core/sourcecoordination/... ./orchestration/mailbox/... ./orchestration/instanceregistry/... ./core/types/... ./core/history/... ./runtime/config/... -count=1

echo "[redis-coordination-contract] passed"
