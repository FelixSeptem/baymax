# Offline source adapter lifecycle

Semantic anchor: `source-adapter-lifecycle`, source-bound cursor-after-commit.

Runtime path: `go run ./examples/source-adapter-lifecycle` -> injected offline
source/consumer/store -> `host/sourceadapter`. No network or model credentials.

Expected markers: `sent_confirmed`, `gone_confirmed`, `retryable_preserved`,
`duplicate_no_effect`, `restart_confirmed_cursor`, `abort_preserved`.

The host consumer demonstrates effect CAS with stable external identity.
Pending evidence and confirmed cursor are distinct; MemoryStore is process
local and FileStore only supports same-host single-writer sequential reopen.
The example does not alter a source Run or claim global dedup/terminal ownership.

Rollback notes: stop calling RunOnce/Offer; retain state and host effect evidence
for reconciliation. See [integration guide](../../docs/guides/source-adapter-lifecycle.md).
