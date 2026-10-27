## 1. Contract and source audit

- [x] 1.1 Audit current Journal/ProjectionLedger append, persistence, cursor, digest and acknowledgement paths; record the journal-committed/ledger-missing crash window with a failing regression test before implementation.
- [x] 1.2 Add bounded recovery DTO/result validation for source identity, sequence, event ID, digest, terminal evidence, cursor, retention floor and duplicate state; verify malformed, foreign, gap, expiry and conflict cases with typed errors.
- [x] 1.3 Update the source-history, projection-transaction and capability-manifest delta evidence so retained history, source commit, reconciliation and acknowledgement remain independent; verify strict OpenSpec and documentation impact declarations.

## 2. Source-owned reconciliation

- [x] 2.1 Implement explicit journal-to-ledger reconciliation using only caller-injected matching stores; verify no implicit store allocation and no mutation before all source fact validation succeeds.
- [x] 2.2 Make identical recovery idempotent and conflicting event identity/digest/terminal/cursor evidence fail closed; verify one retained record, stable cursor and explicit duplicate result.
- [x] 2.3 Preserve retention floor, bounded pages and source-local ordering during recovery; verify expired, gapped, foreign and unavailable source/ledger outcomes never fall back to latest or execute effects.
- [x] 2.4 Prove fresh-process recovery after journal append succeeds and ledger persistence fails, then acknowledge the recovered record idempotently; verify source facts and terminal evidence remain unchanged.

## 3. Owner integrations and manifest boundaries

- [x] 3.1 Exercise the recovery seam through explicitly injected runner, child, team and workflow owner journals/ledgers; verify each source retains its own namespace, sequence and terminal facts.
- [x] 3.2 Preserve mailbox lifecycle ownership and direct/composite unsupported boundaries; verify mailbox history, callback IDs and package/DTO presence cannot promote generic projection recovery.
- [x] 3.3 Update capability projection with per-form journal/ledger evidence, deployment scope and stable unsupported/unavailable reasons; verify foreign owner evidence is rejected before admission or source activation.
- [x] 3.4 Add counting-fake tests proving recovery never calls Run, Stream, retry, resume, restore, model, tool, handler, lease, ack, nack, DLQ, business commit or diagnostics writer.

## 4. Replay, fixtures, examples and documentation

- [x] 4.1 Add versioned reference-only fixtures for journal-committed/ledger-missing recovery, duplicate, conflict, malformed, foreign, expired, gap and unavailable outcomes; verify replay constructs no source owner or execution.
- [x] 4.2 Update `examples/native-execution-events/README.md` before example code and extend the offline example with successful reconciliation plus a rejected conflict/unsupported case; verify expected markers and rollback notes.
- [x] 4.3 Synchronize README, integration, runtime-boundary, runtime-config/diagnostics, contract-index, core history/types README and roadmap documentation; verify every Public Semantic Documentation Commitment path is modified.
- [x] 4.4 Add equivalent shell and PowerShell source-history projection-recovery contract gates and wire them into the quality/replay path; verify both reject latest fallback and execution side effects.

## 5. Verification and delivery

- [x] 5.1 Run focused core/history, core/types, runner, composer, teams, workflow, mailbox, planbridge, eventadapter, integration and diagnostics-replay tests, including race coverage for the recovery path.
- [x] 5.2 Run strict OpenSpec validation, `openspec validate --all`, documentation/example/roadmap gates, source-history gates and `git diff --check`; record exact outputs in `verification.md`.
- [x] 5.3 Run `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml ./...`, `pwsh -File scripts/check-quality-gate.ps1`, and `pwsh -File scripts/check-docs-consistency.ps1`; record exit codes, environment limits and rollback evidence before archive. The widened quality gate completed all runtime/contract/test/lint/performance steps; its final `govulncheck ./...` step returned exit 3 for pre-existing Go standard-library and dependency vulnerabilities, recorded in `verification.md`.

## Example Impact Assessment

修改示例

理由：the offline native execution events example demonstrates explicit
source-fact reconciliation after a simulated consumer crash and keeps
duplicate/conflict/unsupported behavior visible without live providers or
business writes.

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
