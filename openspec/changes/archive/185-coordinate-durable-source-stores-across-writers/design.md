## Context

Existing source stores deliberately publish only
`same_host_single_writer_sequential_reopen`. File rename and an owner lock are
not a cross-machine transaction. Checkpoint claim and history append currently
mutate independent in-memory snapshots, so a second writer cannot safely load,
compare, modify and publish the same source. See proposal.md for motivation.

## Goals / Non-Goals

**Goals:**

- Make a source owner opt into a durable coordinator that serializes writer
  lease acquisition and performs revision-checked state replacement.
- Use one immutable source-state envelope per checkpoint store or journal, so
  checkpoint claim/result and journal append/cursor advancement publish only
  from a successful CAS revision.
- Make fence/revision facts portable and auditable in capability DTOs.
- Preserve exact source identities, terminal evidence and replay-only history.

**Non-Goals:**

- No global transaction across different source owners, no scheduler rewrite,
  no automatic migration, no cloud database client, and no claim that an
  arbitrary shared filesystem is strongly consistent.
- No replay-driven recovery, automatic rerun, or fallback from coordinated to
  local execution.

## Decisions

1. **Coordinator is an explicit source-owner dependency.** A `Coordinator`
   operates on one owner-scoped key and returns a lease fence and revision.
   Checkpoint/history owners serialize their full detached snapshot through it.
   This keeps business state and terminal arbitration within the existing owner.
   A generic global store was rejected because it would obscure owner identity
   and create a second runtime.

2. **CAS publishes a whole bounded source snapshot.** Mutation first reads the
   coordinator revision, applies the existing owner validation/state machine,
   then conditionally publishes the complete snapshot. A failed CAS reloads
   facts and returns typed conflict/stale/unavailable; it never retries effects.
   Field-level merge was rejected because it cannot safely merge claims and
   terminal transitions.

3. **Lease fencing is mandatory for cross-writer capability.** Every writer
   obtains an owner namespace + opaque fence + expiry. The coordinator checks
   the latest fence at CAS time. A replacement writer invalidates the old fence
   and stale writes fail closed. Permanent process locks were rejected because
   they provide neither explicit capability evidence nor safe handover.

4. **File coordinator is same-host only.** It persists a bounded envelope and
   uses an OS-visible mutation lock; callers must explicitly configure it. It
   advertises `multi_instance` but never `cross_machine`, because portable file
   locking and rename do not prove distributed filesystem semantics. A future
   network/database backend can implement the same interface and add its own
   evidence without changing owner semantics.

5. **Planbridge remains admission-only.** It inspects owner-projected
   coordinator evidence and rejects mismatch before publish/claim/Run/Stream.
   It neither obtains leases nor coordinates source state.

## Risks / Trade-offs

- [Shared filesystem violates atomic-lock assumptions] → deployment profile is
  explicit; missing or failed lock/CAS becomes unavailable rather than support.
- [CAS conflict after a handler effect] → only source claim/cursor state is
  coordinated; effect dispatch follows existing durable claim safe points and
  conflict returns recovery-required, never automatic execution.
- [Snapshot size growth] → reuse existing continuation/history limits and
  reject over-bound envelopes before coordinator publish.
- [Mixed local/coordinated writer] → owner key and revision format are bound
  into the capability; a local-only store cannot open or advertise the
  coordinated profile.

## Migration Plan

1. Existing stores keep their current constructors and local scope.
2. Hosts explicitly create/configure a coordinator and use the coordinated
   owner constructors/options.
3. Capability manifests and planbridge only admit coordinated requests when
   live coordinator evidence matches.
4. Rollback removes the configured coordinator before new writes; retained
   coordinated source state is not reinterpreted by a local store.

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

## Example Impact Assessment

新增示例

说明：the offline example shows explicit coordinator configuration, a
successful CAS/fence transition, and a rejected local-only distributed request.
