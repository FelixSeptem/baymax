## 1. Core Protocol Model

- [x] 1.1 Add additive checkpoint relation, lineage, branch, restore-source, replay, and workspace provenance DTOs in `core/types`.
- [x] 1.2 Add finite enums, bounded normalization, cross-reference validation, and deterministic provenance reason codes.
- [x] 1.3 Add positive, negative, boundary, JSON compatibility, lineage-chain, branch-conflict, replay-idempotency, and workspace-drift unit tests.

## 2. Snapshot and Runtime Projection

- [x] 2.1 Extend `orchestration/snapshot` pure manifest projection helpers to accept recovery and workspace context without changing manifest storage or restore semantics.
- [x] 2.2 Wire source-owned composer/recovery context into protocol projection for fresh, resume, and cross-session restore.
- [x] 2.3 Add strict/compatible restore, cross-session, and Run/Stream provenance parity integration tests.
- [x] 2.4 Add control-plane, second-store, workspace-content, and source-mutation absence assertions.

## 3. Diagnostics, Replay, and Gates

- [x] 3.1 Add nullable checkpoint/workspace provenance diagnostics and low-cardinality OTel fields through `RuntimeRecorder` only.
- [x] 3.2 Add `agent_runtime_protocol_checkpoint_provenance.v1` fixtures for success, lineage/schema/branch/replay/workspace drift, and malformed inputs.
- [x] 3.3 Extend replay parsing, normalized output, and drift taxonomy while preserving historical fixture compatibility.
- [x] 3.4 Extend shell/PowerShell contract gates for provenance validation, replay idempotency, Run/Stream parity, single-writer, and ownership boundaries.

## 4. Example and Documentation

- [x] 4.1 **Example Impact Assessment: 新增示例** - update `examples/agent-modes/MATRIX.md` and the checkpoint/workspace provenance README first with semantic anchor, runtime path, expected markers, restore/drift behavior, and rollback notes.
- [x] 4.2 Add executable minimal and production-ish example smoke paths only after the documentation baseline is complete.
- [x] 4.3 Update README, runtime diagnostics, module boundaries, contract index, and roadmap references.

## 5. Verification and Delivery

- [x] 5.1 Run affected package and integration tests, including positive, negative, boundary, replay, ownership, and Run/Stream parity cases.
- [x] 5.2 Run `go test ./...` and `go test -race ./...`.
- [x] 5.3 Run `golangci-lint run --config .golangci.yml`.
- [x] 5.4 Run `pwsh -File scripts/check-quality-gate.ps1` and `pwsh -File scripts/check-docs-consistency.ps1`.
- [x] 5.5 Review additive compatibility, source ownership, rollback, diagnostics cardinality, Example Impact Assessment, and archive readiness evidence.
