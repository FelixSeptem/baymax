#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
go test ./orchestration/outboxbridge ./orchestration/mailbox ./orchestration/scheduler ./tool/diagnosticsreplay -count=1
a=$(go run ./examples/application-outbox-recovery)
for marker in mailbox_dispatch_accepted scheduler_dispatch_accepted publish_before_outbox_confirmation_reconciled effect_committed_before_target_reconciliation duplicate_converged rollback_preserves_pending_facts; do grep -Fq "$marker" <<<"$a"; done
echo "[application-outbox] contract gate passed"
