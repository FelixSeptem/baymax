## ADDED Requirements

### Requirement: Source-owner history expansion SHALL declare actual boundaries

For runner, isolated child, team, workflow, and mailbox sources, Baymax SHALL
publish a source-scoped history declaration identifying the event identity
namespace, whether a source-local sequence exists and its ordering scope,
retention window and floor, opaque cursor scope/version, terminal recovery
semantics, and current availability. A missing declaration SHALL remain
unsupported rather than being inferred from callbacks, DTOs, package presence,
Run IDs, timestamps, or host observation order.

#### Scenario: Unsupported source is explicit
- **WHEN** a caller requests history for a source without a retained journal
- **THEN** the source returns `ErrHistoryUnsupported` before reading or
  activating execution
- **AND** its capability declaration includes a stable unsupported reason

#### Scenario: Source availability is explicit
- **WHEN** a declared retained source cannot be opened or its store is
  unavailable
- **THEN** the reader returns `ErrSourceUnavailable`
- **AND** it does not downgrade the source to unsupported or start a new run

### Requirement: Source-owner replay SHALL retain bounded cursor semantics

A supported source SHALL provide bounded, detached pages after an optional
opaque cursor, including events, an optional next cursor, an optional retention
floor, and a complete flag. Cursors SHALL be interpretable only by their source
namespace and source instance. Malformed, foreign, expired, and gapped cursors
MUST return `ErrCursorMalformed`, `ErrCursorForeign`, `ErrCursorExpired`, or
`ErrHistoryGap` respectively; the reader MUST NOT silently use latest or the
retention floor after an error.

#### Scenario: Consumer restart continues from a confirmed cursor
- **WHEN** a new consumer process reads a retained source with its last
  confirmed cursor
- **THEN** it receives only the subsequent bounded source-local events and a
  valid next cursor
- **AND** no source lifecycle transition is performed

#### Scenario: Retention loss fails closed
- **WHEN** a cursor points before the retained floor or a retained sequence has
  an unrecoverable gap
- **THEN** the reader returns the corresponding typed expiry or gap error
- **AND** it does not return an unrelated latest page

### Requirement: Source-owner replay SHALL remain projection-safe

History reads SHALL expose source-owned facts without invoking Run, Stream,
model, tool, handler, retry, resume, restore, ack, nack, lease, requeue, DLQ,
scheduler, business commit, or diagnostics writes. Source journal progress and
consumer projection progress SHALL be independent. Overlapping pages SHALL be
idempotently deduplicated by stable source event identity; conflicting payloads
MUST fail closed.

#### Scenario: Projection crashes before consuming terminal evidence
- **WHEN** the source has retained a terminal event but its consumer crashes
  before projection
- **THEN** the next read after the confirmed cursor returns the same terminal
  evidence
- **AND** replay does not execute or mutate the source

#### Scenario: Overlap and payload conflict are handled safely
- **WHEN** a consumer reads an overlapping page twice
- **THEN** identical source events are accepted once
- **AND** the same source identity with a different payload fails closed

### Requirement: Source-local ordering SHALL not imply a global order

Each source SHALL state the exact scope of its monotonic sequence. Parallel
child or team sources MUST retain their own local order without claiming a
global natural order, and a history cursor MUST NOT be accepted as an execution
checkpoint or restore authority.

#### Scenario: Parallel sources remain independently ordered
- **WHEN** two child or team member sources emit events concurrently
- **THEN** each source can be replayed in its declared local order
- **AND** the runtime does not synthesize a cross-source sequence

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`core/types/README.md`、`orchestration/mailbox/README.md`、`orchestration/planbridge/README.md`、`examples/native-execution-events/README.md`
- 维护 owner：source-owner、runtime-core、mailbox、host-integration、documentation 与 examples maintainers
- 验证命令：`openspec validate retain-run-and-composite-source-history --strict`、history contract gates、`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`

## Example Impact Assessment

修改示例。示例需展示 source matrix、cursor restart、typed cursor errors 和无副作用 replay；只修改 `examples/native-execution-events/README.md` 及其离线示例。
