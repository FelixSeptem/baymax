# Source adapter lifecycle

`host/sourceadapter` is an opt-in, host-injected external delivery library.
It supplies one-page polling (`RunOnce`) and explicit callback pages (`Offer`),
not an automatic service or worker. The source owns immutable facts and opaque
cursor encoding; the host owns trust, authorization, permanent-gone decisions
and effect CAS.

## Delivery and restart

Construct Metadata with SourceIdentity (kind/namespace/ID), IdentityNamespace
and CursorVersion. Inject matching Source, Consumer and Store as constructor
arguments; Options selects operation timeout and page limit.
Native history's closed source kinds remain unchanged; do not label an
external transport as a runner/mailbox solely to obtain a history identity.
Callback Observer cannot supply polling or cursor persistence.

A Fact has ExternalID, Reference, Payload, a matching `sha256:` payload Digest,
and source-bound NextCursor. Pages bind the same Metadata. References are
bounded to 256 UTF-8 bytes (kind 64), payload to 64 KiB, token to 4096 bytes, and
page size and recent confirmed receipt window to 200. References reject
whitespace/control characters; payload is transient and never persisted.

The adapter validates/detaches the full page before effects, then processes it
in source-local order. It persists Pending identity/fingerprint before each
consumer call without advancing Cursor. Only consumer `Sent` or `Gone`
permits confirmation and clears Pending; `Retryable`, errors and cancellation
leave that fact unconfirmed and stop the page. Earlier confirmed facts remain.
`Gone` requires host proof of permanent non-deliverability, not a timeout.
Consumer callbacks receive detached bytes and must cooperate with context.

A consumer effect can succeed before its confirmation write fails. Retry must
use the same external identity and host-owned effect CAS. Pending evidence
detects a changed retry even after process restart; the last 200 confirmed
receipts detect recent conflicts. There is no global dedup guarantee outside
that window and no adapter-owned business-effect ledger.

## Storage and shutdown

MemoryStore is process-local. FileStore is same-host single-writer sequential
reopen only, with bounded strict JSON and atomic temp/rename writes. One live
instance must own a file; independent instances, distributed CAS and power-loss
transaction guarantees are not supplied. A host requiring broader coordination
must inject a store with that contract.

The file JSON envelope is at most 8 MiB (including escaped cursor references),
rejects duplicate/unknown keys, trailing data, excessive nesting, missing schema
or source binding, and stores no payload. Revision exhaustion fails before
consumer invocation by reserving both pending and confirmation writes.

CompareAndSwap advances state Revision exactly once and returns detached state.
Store errors must not be treated as proof of rollback for an unknown external
store. Reload and reconcile state against host effects after an uncertain write.
Corrupt, oversized or foreign files fail closed without overwrite.

The default operation timeout is five seconds. Dispose and Abort only cancel
cooperative lifecycle work and reject later operations; they do not control a
Run, source execution, terminal state, or consumer effect. A store commit
already admitted when cancellation occurs may finish; Load is authoritative.
Health retains finite reason, bounded reference and saturating counters, with no
payload or raw dependency error. Concurrent operations fail explicitly.

## Verification and rollback

Use `go test ./host/sourceadapter -count=1` and
`pwsh -File scripts/check-source-adapter-lifecycle-contract.ps1` (or the
equivalent shell gate). The versioned offline replay is reference-only and
never polls a source or invokes consumers, tools, models or network.
The [offline example](../../examples/source-adapter-lifecycle/README.md)
shows sent/gone/retryable, abort and reopen.

Rollback stops polling/offers and preserves the last confirmed cursor, Pending
and host effect records for reconciliation; never discard evidence to force a
retry. This API adds no runtime configuration key, QueryRuns field,
RuntimeRecorder writer, native history owner or execution terminal owner.
