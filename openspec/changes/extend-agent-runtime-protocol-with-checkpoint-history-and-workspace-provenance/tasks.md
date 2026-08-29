## 1. Core Protocol Model

- [ ] 1.1 Add additive checkpoint relation, lineage, branch, restore-source, replay, and workspace provenance DTOs in `core/types`.
- [ ] 1.2 Add finite enums, bounded normalization, cross-reference validation, and deterministic provenance reason codes.
- [ ] 1.3 Add positive, negative, boundary, JSON compatibility, lineage-chain, branch-conflict, replay-idempotency, and workspace-drift unit tests.

## 2. Snapshot and Runtime Projection

- [ ] 2.1 Extend `orchestration/snapshot` pure manifest projection helpers to accept recovery and workspace context without changing manifest storage or restore semantics.
- [ ] 2.2 Wire source-owned composer/recovery context into protocol projection for fresh, resume, and cross-session restore.
- [ ] 2.3 Add strict/compatible restore, cross-session, and Run/Stream provenance parity integration tests.
- [ ] 2.4 Add control-plane, second-store, workspace-content, and source-mutation absence assertions.

## 3. Diagnostics, Replay, and Gates

- [ ] 3.1 Add nullable checkpoint/workspace provenance diagnostics and low-cardinality OTel fields through `RuntimeRecorder` only.
- [ ] 3.2 Add `agent_runtime_protocol_checkpoint_provenance.v1` fixtures for success, lineage/schema/branch/replay/workspace drift, and malformed inputs.
- [ ] 3.3 Extend replay parsing, normalized output, and drift taxonomy while preserving historical fixture compatibility.
- [ ] 3.4 Extend shell/PowerShell contract gates for provenance validation, replay idempotency, Run/Stream parity, single-writer, and ownership boundaries.

## 4. Example and Documentation

- [ ] 4.1 **Example Impact Assessment: 新增示例** - update `examples/agent-modes/MATRIX.md` and the checkpoint/workspace provenance README first with semantic anchor, runtime path, expected markers, restore/drift behavior, and rollback notes.
- [ ] 4.2 Add executable minimal and production-ish example smoke paths only after the documentation baseline is complete.
- [ ] 4.3 Update README, runtime diagnostics, module boundaries, contract index, and roadmap references.

## 5. Verification and Delivery

- [ ] 5.1 Run affected package and integration tests, including positive, negative, boundary, replay, ownership, and Run/Stream parity cases.
- [ ] 5.2 Run `go test ./...` and `go test -race ./...`.
- [ ] 5.3 Run `golangci-lint run --config .golangci.yml`.
- [ ] 5.4 Run `pwsh -File scripts/check-quality-gate.ps1` and `pwsh -File scripts/check-docs-consistency.ps1`.
- [ ] 5.5 Review additive compatibility, source ownership, rollback, diagnostics cardinality, Example Impact Assessment, and archive readiness evidence.
