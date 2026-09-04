## 1. Contract and ownership baseline

- [x] 1.1 Audit current Session, Run, Step, Event, snapshot, checkpoint, provenance, and diagnostics replay owners and record the source-of-truth mapping in the design notes.
- [x] 1.2 Define bounded session-history, history-leaf, branch/fork, restore-operation, and replay-operation types with additive nullable/defaultable fields.
- [x] 1.3 Implement deterministic validation for history roots, parent continuity, monotonic positions, session association, and serialized-size limits.
- [x] 1.4 Add unit tests covering valid chains, missing parents, position regressions, conflicting identifiers, and backward-compatible empty history.

## 2. Protocol and checkpoint projections

- [x] 2.1 Extend Agent Runtime Protocol checkpoint projection with optional history leaf, branch/fork, and replay associations without changing existing manifest ownership.
- [x] 2.2 Extend checkpoint/workspace provenance normalization to validate session-history association and preserve existing workspace integrity semantics.
- [x] 2.3 Add branch/fork projection that creates a distinct Run lineage and never mutates a terminal parent Run.
- [x] 2.4 Add protocol and provenance tests for matching associations, cross-session mismatch, missing branch parent, and terminal Run immutability.

## 3. Snapshot restore boundary

- [x] 3.1 Add pre-restore validation for history/checkpoint lineage, schema compatibility, and required references in both `strict` and `compatible` modes.
- [x] 3.2 Preserve existing snapshot manifest and segment source ownership while recording bounded compatible-mode downgrade metadata.
- [x] 3.3 Add atomic restore-operation identity handling and deterministic conflict classification for repeated or conflicting imports.
- [x] 3.4 Add restore tests for strict rejection, compatible downgrade, duplicate idempotency, conflicting identity, and no-mutation-on-failure.

## 4. Diagnostics replay and fixtures

- [x] 4.1 Extend diagnostics replay tooling with a versioned session-history/checkpoint fixture schema and canonical normalization digest.
- [x] 4.2 Add valid, malformed, lineage-gap, branch-conflict, restore-conflict, side-effect, and Run/Stream parity drift fixtures.
- [x] 4.3 Enforce offline read-only replay: no provider/tool execution, workspace mutation, snapshot mutation, branch creation, or parallel diagnostics writer.
- [x] 4.4 Add replay unit tests for historical fixture compatibility, deterministic conflict taxonomy, side-effect rejection, and duplicate replay idempotency.
- [x] 4.5 Update the mainline contract index and add shell/PowerShell replay gate scripts with semantic parity.

## 5. Example documentation baseline

- [x] 5.1 Update `examples/agent-modes/MATRIX.md` with a session/checkpoint branch and offline replay mode, including semantic anchor, runtime path, expected markers, and rollback notes.
- [x] 5.2 Add the corresponding mode `README.md` documenting history leaf selection, branch lineage, restore policy, replay safety, and expected output markers.
- [x] 5.3 Add a minimal example implementation only after the documentation baseline is complete, demonstrating branch creation and side-effect-free replay.
- [x] 5.4 Add a production-ish example variant covering incompatible restore, replay conflict, and Run/Stream parity assertions.
- [x] 5.5 Add agent-mode smoke assertions for history continuity, distinct branch Run correlation, restore classification, and replay side-effect prohibition.

## 6. Observability, configuration, and compatibility

- [x] 6.1 Add bounded additive diagnostics fields for history leaf, branch/fork, restore operation, and replay classification through `RuntimeRecorder` only.
- [x] 6.2 Document field defaults, compatibility behavior, conflict taxonomy, and retention ownership in `docs/runtime-config-diagnostics.md`.
- [x] 6.3 Verify module boundaries and explicitly document that no session store, hosted gateway, provider SDK, or artifact content service is introduced.
- [x] 6.4 Update `README.md` and `docs/development-roadmap.md` with the proposal scope, status, dependencies, and non-goals.

## 7. Verification and delivery

- [x] 7.1 Run focused unit and integration suites for core types, protocol projections, snapshot restore, diagnostics replay, and examples.
- [x] 7.2 Run `go test ./...`, `go test -race ./...`, and `golangci-lint run --config .golangci.yml`.
- [x] 7.3 Run the session-history contract gate, all OpenSpec validation, `check-quality-gate.*`, and `check-docs-consistency.*` with shell/PowerShell parity.
- [x] 7.4 Review `git diff --check`, confirm additive compatibility and no proposal-number leakage into code names, then prepare the change for archive.

## Example Impact Assessment

新增示例

Documentation and smoke coverage must precede implementation for the new session/checkpoint branch and offline replay examples.
