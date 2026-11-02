## Why

Baymax correctly limits its current file-backed source stores to same-host,
single-writer, sequential reopen. That is safe but leaves no source-owned path
for a host that genuinely needs concurrent writers or a shared durable source
across processes/machines: `multi_writer_cas` exists as vocabulary but no
store can evidence it, atomically claim a checkpoint, or advance a cursor.
This change adds that missing coordination boundary without reclassifying local
FileStore as distributed.

## What Changes

- Add a versioned, injectable durable source-store coordination contract with
  writer leases/fences, monotonically versioned compare-and-swap, and durable
  source-owner evidence.
- Provide a durable same-host multi-process coordinator for explicitly
  configured source stores and a deterministic process-local coordinator for
  tests; both reject foreign/stale writers and CAS conflicts. The built-in file
  adapter MUST NOT advertise cross-machine safety.
- Bind coordinated checkpoint claim/result publication and source-history append
  / cursor advancement to one coordinator transaction boundary.
- Advertise same-host multi-instance `multi_writer_cas` only from owners
  configured with an active compatible coordinator; preserve existing local
  FileStore behavior otherwise. Cross-machine remains unsupported without an
  externally supplied backend that proves distributed locking/CAS.
- Make planbridge admission reject a requested distributed profile when its
  bound owner lacks live coordination evidence, before any source activation.

## Capabilities

### New Capabilities

- `durable-source-store-coordination`: source-owned lease, fencing, CAS and
  revision evidence for durable state shared by multiple writers.

### Modified Capabilities

- `durable-native-continuation-restore`: permit a checkpoint store to evidence
  multi-writer atomic resume claim only through a configured coordinator.
- `native-execution-event-history`: permit coordinated append/cursor
  advancement while preserving source-local ordering and replay-only reads.
- `host-runtime-capability-manifest`: publish live coordination capability and
  evidence rather than infer it from file persistence.
- `host-execution-plan-bridge`: validate coordinator-backed deployment claims
  before plan activation.

## Impact

- New `core/sourcecoordination` package plus additive `core/types` contracts.
- `core/checkpoint`, `core/history`, capability projections, planbridge,
  diagnostics/replay fixtures, contract gates and integration documentation.
- No hosted database, global scheduler, queue, control plane, or automatic
  migration of existing file-backed stores.

## Example Impact Assessment

新增示例

说明：add an offline coordinated-source-store example that shows the writer
lease/CAS path and the local-only rejection path without a provider.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`core/history/README.md`、`orchestration/README.md`、`examples/coordinated-source-store/README.md`
- 维护 owner：core source-coordination, checkpoint/history and planbridge maintainers
- 验证证据：focused Go tests、durable-store shell/PowerShell gate、docs consistency gate、OpenSpec validation and full quality verification

## Documentation Impact Assessment

| 表面 | 结果 | 文档路径 | Owner | 验证证据 |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | architecture maintainers | boundary review and docs consistency gate |
| components | 修改文档 | `core/history/README.md`, `orchestration/README.md` | source-owner maintainers | focused coordinator/history/checkpoint tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime configuration maintainers | constructor-only/no-config-key documentation review |
| contract/API | 修改文档 | `README.md`, `docs/guides/integration.md` | core and integration maintainers | deployment, checkpoint, history and planbridge tests |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md`, `docs/runtime-config-diagnostics.md` | diagnostics maintainers | diagnostics replay tests and dedicated gate |
| examples | 新增文档 | `examples/coordinated-source-store/README.md` | example maintainers | `go run ./examples/coordinated-source-store` |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | integration maintainers | integration documentation review and dedicated gate |
| best practices | 修改文档 | `docs/guides/integration.md` | integration maintainers | explicit coordinator/rollback guidance review |
| roadmap | 修改文档 | `docs/development-roadmap.md` | runtime maintainers | roadmap consistency gate |
