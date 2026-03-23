#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$PWD/.gocache"
fi
mkdir -p "${GOCACHE}"

echo "[canonical-mailbox-entrypoints] contributioncheck"
go test ./tool/contributioncheck -run '^TestCanonicalMailboxInvokeEntrypoints$' -count=1

echo "[canonical-mailbox-entrypoints] sync invocation canonical suite"
go test ./integration -run '^TestSyncInvocationContractCanonicalMailboxOnlyPublicEntrypoints$' -count=1

echo "[canonical-mailbox-entrypoints] async invocation canonical suite"
go test ./integration -run '^TestAsyncReportingContractLegacyDirectAsyncEntrypointNotSupportedPublicly$' -count=1

echo "[canonical-mailbox-entrypoints] mailbox convergence canonical suite"
go test ./integration -run '^TestMailboxContractCanonicalEntrypointConvergenceGuard$' -count=1

