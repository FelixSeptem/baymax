## Why

Baymax already persists source-owned journal facts and can optionally bind a
`ProjectionLedger`, but the two stores are not one atomic transaction. When a
journal fact is durable and ledger persistence fails, the owner returns
`ErrProjectionRecoveryRequired`; after restart the source fact is readable, yet
there is no public, source-bound reconciliation operation that can safely
rebuild the missing ledger record and continue idempotent projection. This
leaves a recoverable source fact permanently stranded or encourages an unsafe
effect rerun.

## What Changes

- Add an explicit source-owned journal-to-ledger reconciliation API that uses
  only an already injected `history.Journal` and `ProjectionLedger`; it never
  creates a journal, queue, worker, scheduler, or execution loop.
- Reconcile bounded source facts by source identity, sequence, event identity,
  digest, terminal evidence, retention floor, and cursor scope. Identical
  overlap is idempotent; conflicting evidence, gaps, foreign or expired
  cursors, malformed input, and unavailable stores fail closed with typed
  errors.
- Make recovery of a journal-committed/ledger-missing fact possible in a fresh
  process, while keeping source replay read-only with respect to execution and
  allowing writes only to the explicitly supplied source projection ledger.
- Expose the same recovery boundary through runner, child, team, and workflow
  owner integration evidence. Mailbox continues to use its existing owner
  history and does not become a generic ledger or execution-recovery owner.
- Keep capability cells separate for retained history, source commit
  persistence, projection recovery, and consumer acknowledgement. Only a
  durable journal plus a matching durable ledger may advertise the new
  recovery evidence; package or DTO presence remains invalid evidence.
- Add crash-window, fresh-process, overlap, conflict, retention, unavailable,
  no-side-effect, replay-fixture, documentation, and shell/PowerShell gate
  coverage.

## Capabilities

### New Capabilities

None. This change extends existing source-history and projection transaction
contracts without adding another history store or runtime owner.

### Modified Capabilities

- `source-history-projection-transaction`: add an explicit source-fact
  reconciliation operation and idempotent recovery result for a ledger that
  missed a source commit.
- `native-execution-event-history`: clarify that source replay can feed an
  explicitly injected projection ledger through a bounded recovery operation,
  while remaining distinct from execution resume and callback replay.
- `host-runtime-capability-manifest`: distinguish durable retained history,
  source commit evidence, projection reconciliation, and consumer
  acknowledgement, including the same-host single-writer boundary.

## Impact

- Affected code: `core/history`, `core/types`, runner/composer/team/workflow
  history owner adapters, planbridge capability projection, diagnostics
  replay fixtures, examples, contract gates, and public integration docs.
- Public API changes are additive: existing journal, ledger, history reader,
  and acknowledgement calls retain their behavior; callers gain an explicit
  recovery method and typed duplicate/conflict outcomes.
- Recovery mutates only the explicitly supplied source projection ledger. It
  never mutates the source execution owner, starts execution, retries an
  effect, advances a mailbox lease, writes diagnostics, or commits a business
  projection.
- Deployment remains same-host, single-writer, sequential reopen. Multi-writer
  CAS, cross-machine coordination, cross-database atomicity, and execution
  restore remain unsupported.

## Example Impact Assessment

修改示例

理由：the offline native execution events example must show a journal-committed
fact recovered into an explicitly injected projection ledger after a simulated
consumer crash, plus duplicate/conflict and unsupported boundaries. It remains
offline and must not call a provider, model, tool, business database, or
execution owner.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`,
  `docs/guides/integration.md`, `docs/runtime-module-boundaries.md`,
  `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`,
  `core/history/README.md`, `core/types/README.md`,
  `examples/native-execution-events/README.md`
- 维护 owner：runtime-core/source-history owners, planbridge capability owner,
  diagnostics-replay and examples maintainers
- 验证证据：strict OpenSpec validation, documentation/example-impact and
  roadmap gates, source-history contract/replay gates, focused owner tests,
  `go test ./...`, `go test -race ./...`, golangci-lint, quality gate, and
  docs consistency verification recorded in `verification.md`.

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runtime-core/source owners | no-second-store and ownership gate |
| components | 修改文档 | `core/history/README.md`, `core/types/README.md` | history/types owners | recovery API tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime config owner | no-new-config gate |
| contract/API | 修改文档 | `docs/guides/integration.md` | source-history owner | strict spec and integration tests |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | replay maintainer | fixture/gate index tests |
| examples | 修改文档 | `examples/native-execution-events/README.md` | examples maintainer | offline smoke test |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | source/planbridge owners | fail-closed recovery tests |
| best practices | 修改文档 | `README.md` | documentation owner | docs consistency gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap owner | roadmap status consistency gate |
