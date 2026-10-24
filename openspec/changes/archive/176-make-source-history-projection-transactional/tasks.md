## 1. Source Commit and Projection Ledger

- [x] 1.1 Add bounded public source-commit, projection acknowledgement, and recovery status types plus typed unavailable/conflict validation; verify unit tests cover malformed identity, changed digest, and invalid cursor/reference bounds.
- [x] 1.2 Implement an explicit source-owned in-memory/file-backed commit ledger with canonical payload-omitted digests, atomic duplicate projection acknowledgement, restart reopen, and retention-floor behavior; verify fresh-process reopen and conflict tests pass.
- [x] 1.3 Integrate explicit ledger commits with `core/history.Journal` without changing default no-ledger behavior; verify journal append/ledger failure windows return committed evidence or typed recovery-required/unavailable and never expose fabricated facts.

## 2. Recovery Reads and Source Capability Evidence

- [x] 2.1 Expose bounded read-after-commit and projection acknowledgement APIs that are read-only with respect to execution; verify restart-before-projection, overlap idempotency, terminal recovery, unavailable store, and no-side-effect tests pass.
- [x] 2.2 Extend history capability declarations with source commit/projection recovery scope and deployment limits; verify retained-history-only owners remain explicit and unsupported for recovery.
- [x] 2.3 Extend the host capability manifest projection/validation for source commit recovery evidence; verify it cannot be inferred from history, continuation DTOs, or file reopen alone.

## 3. Owner Integration and Contract Evidence

- [x] 3.1 Wire runner, child, team, and workflow explicit history owners through the ledger option without creating implicit stores; verify owner append health remains distinct from commit/projection recovery and Run/Stream results retain parity.
- [x] 3.2 Add source commit/recovery fixtures and equivalent PowerShell/shell contract gates for crash-before-projection, duplicate acknowledgement, conflicting payload, unavailable owner, retention expiry, and no execution side effects.

## 4. Public Documentation and Verification

- [x] 4.1 Update README, integration guide, type/history READMEs, runtime boundaries/diagnostics, contract-test index, and roadmap status; verify proposal/design/tasks public semantic documentation commitments remain consistent.
- [x] 4.2 Run focused history/owner/manifest tests, `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, dedicated PowerShell/shell contract gates, `pwsh -File scripts/check-quality-gate.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, and `openspec validate --all`; record any environment limitation before archive.

### Verification Record

- Focused history/owner/manifest suites passed: `go test ./core/history ./core/types ./core/runner ./orchestration/composer ./orchestration/teams ./orchestration/workflow ./orchestration/planbridge ./tool/diagnosticsreplay -count=1`.
- Full Go test and race suites passed: `go test ./... -count=1`; `go test -race ./... -count=1`.
- `golangci-lint run --config .golangci.yml` passed with 0 issues.
- Dedicated PowerShell gates passed: source history projection and native event history.
- The equivalent Bash gates were attempted but could not start in this Windows environment (`Bash/Service/CreateInstance/E_ACCESSDENIED` and Git Bash signal-pipe `Win32 error 5`); this is recorded as an environment limitation, not a passing shell result.
- `pwsh -File scripts/check-docs-consistency.ps1`, `pwsh -File scripts/check-openspec-documentation-impact.ps1`, `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1`, `pwsh -File scripts/check-openspec-example-impact-declaration.ps1`, `openspec validate --all` (146 passed, 0 failed), and `git diff --check` passed.
- The first `pwsh -File scripts/check-quality-gate.ps1` attempt hit the script's default 1,800-second total budget while `go test -race ./...` was running, not a test assertion failure. The full quality gate was rerun with `BAYMAX_QUALITY_GATE_TOTAL_TIMEOUT_SECONDS=3600` and `BAYMAX_QUALITY_GATE_STEP_TIMEOUT_SECONDS=900`; its full test, race, lint, contract, and benchmark steps completed without a reported failing check.

## Example Impact Assessment

修改示例

理由：`examples/native-execution-events/README.md` 增加 source-owned ledger/recovery 的语义锚点；不改变正常调用 form、配置键、示例 runtime path 或 expected marker。确定性 fixture、restart test 和 dedicated contract gate 仍是交付证据。

## Documentation Impact Assessment

| Area | Outcome | Document | Owner | Evidence |
|---|---|---|---|---|
| architecture | 修改文档 | `docs/runtime-module-boundaries.md`, `core/history/README.md` | Runtime contracts owner | source-owner commit boundary, read-only recovery and no second journal |
| components | 修改文档 | `core/types/README.md`, `docs/guides/integration.md` | Runtime contracts owner | public commit/acknowledgement/recovery API and typed failures |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | Runtime contracts owner | no configuration key is added; the deployment/recovery boundary is documented |
| contract/API | 修改文档 | `openspec/changes/make-source-history-projection-transactional/specs/native-execution-event-history/spec.md`, `openspec/changes/make-source-history-projection-transactional/specs/host-runtime-capability-manifest/spec.md` | Runtime contracts owner | delta specs, fixtures and manifest tests |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md` | Runtime contracts owner | confirms no RuntimeRecorder schema addition and records gate evidence |
| examples | 修改文档 | `examples/native-execution-events/README.md` | Runtime contracts owner | example documents explicit owner ledger injection and the no-rerun recovery boundary |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | Runtime contracts owner | explicit owner injection and read-only recovery integration are documented |
| best practices | 无需文档变更（附理由） | — | Runtime contracts owner | this change does not alter normal application construction or operational best practices |
| roadmap | 修改文档 | `README.md`, `docs/development-roadmap.md` | Runtime roadmap owner | active state and successor dependency update |

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`, `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, `docs/guides/integration.md`, `docs/mainline-contract-test-index.md`, `core/history/README.md`, `core/types/README.md`, `examples/native-execution-events/README.md`, `openspec/changes/make-source-history-projection-transactional/specs/native-execution-event-history/spec.md`, `openspec/changes/make-source-history-projection-transactional/specs/host-runtime-capability-manifest/spec.md`
- 维护 owner：Runtime contracts owner and source-owner maintainers
- 验证证据：`openspec validate --all`; `pwsh -File scripts/check-openspec-documentation-impact.ps1`; `pwsh -File scripts/check-docs-consistency.ps1`; focused source commit/replay tests and dedicated contract gates
