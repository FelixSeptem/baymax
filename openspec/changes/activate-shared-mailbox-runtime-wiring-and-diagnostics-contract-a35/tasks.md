## 1. Preconditions and Scope Guard

- [ ] 1.1 Confirm A34 canonical invoke entrypoint scope is fixed and avoid overlapping API-surface edits in A35.
- [ ] 1.2 Inventory all managed-path `NewInMemoryMailboxBridge()` call sites and classify migration ownership (`composer`/`scheduler`/`collab`).
- [ ] 1.3 Freeze fallback policy and diagnostics fields for mailbox runtime wiring (`file -> memory` with deterministic reason).

## 2. Shared Mailbox Runtime Wiring

- [ ] 2.1 Add managed mailbox runtime holder in composer (instance + configured backend + effective backend + fallback flag/reason + config signature).
- [ ] 2.2 Wire mailbox initialization from effective `runtime/config` mailbox domain (`enabled/backend/path/retry/ttl/query`).
- [ ] 2.3 Implement mailbox config signature refresh for next attempt, aligned with existing scheduler/recovery refresh style.
- [ ] 2.4 Ensure `mailbox.enabled=false` still provides shared memory mailbox runtime wiring (no direct-path bypass).

## 3. Managed Path Migration

- [ ] 3.1 Introduce bridge/provider injection path for collab/scheduler invoke usage on managed orchestration flows.
- [ ] 3.2 Replace managed per-call bridge creation with shared runtime mailbox bridge usage.
- [ ] 3.3 Preserve non-managed/testing local bridge construction path to keep isolated unit test ergonomics.

## 4. Diagnostics and Query Alignment

- [ ] 4.1 Emit mailbox diagnostics records from command/result/delayed publish paths in managed execution flow.
- [ ] 4.2 Record mailbox backend fallback metadata into diagnostics for query/aggregate traceability.
- [ ] 4.3 Verify `runtime/config.Manager.QueryMailbox` and `MailboxAggregates` reflect real managed orchestration traffic.

## 5. Contract Tests and Gates

- [ ] 5.1 Add/adjust integration suites for mailbox wiring activation under `mailbox.enabled=false/true`.
- [ ] 5.2 Add/adjust integration suites for `backend=file` init-failure fallback-to-memory semantics.
- [ ] 5.3 Add Run/Stream equivalence and memory/file parity assertions under shared mailbox runtime wiring.
- [ ] 5.4 Extend shared multi-agent gate scripts to include mailbox runtime wiring suites as blocking checks.
- [ ] 5.5 Ensure quality gate path includes mailbox runtime wiring regression checks with deterministic non-zero failures.

## 6. Documentation Synchronization

- [ ] 6.1 Update `README.md` to describe mailbox runtime wiring behavior and fallback semantics.
- [ ] 6.2 Update `docs/runtime-config-diagnostics.md` with mailbox enabled/disabled wiring semantics and diagnostics visibility guarantees.
- [ ] 6.3 Update `docs/mainline-contract-test-index.md` with mailbox runtime wiring coverage and gate mappings.
- [ ] 6.4 Update `docs/development-roadmap.md` and `orchestration/README.md` to remove mailbox middle-state wording.

## 7. Validation

- [ ] 7.1 Run `go test ./...`.
- [ ] 7.2 Run `go test -race ./...`.
- [ ] 7.3 Run `golangci-lint run --config .golangci.yml`.
- [ ] 7.4 Run `pwsh -File scripts/check-multi-agent-shared-contract.ps1`.
- [ ] 7.5 Run `pwsh -File scripts/check-quality-gate.ps1`.
- [ ] 7.6 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [ ] 7.7 Run `openspec validate activate-shared-mailbox-runtime-wiring-and-diagnostics-contract-a35 --strict`.
