## 1. Contract and normalization

- [x] 1.1 Define versioned candidate-comparison manifest, candidate-surface constraints, outcome axes, selection record, and cost-ledger DTOs in `runtime/evalcontract`; verify normalization and digest stability with positive, negative, and boundary unit tests.
- [x] 1.2 Implement fixed-identity admission and drift classification for model, authority, data, budget, network policy, rubric, baseline, and candidate digests; verify same-identity conflict fixtures are rejected deterministically.
- [x] 1.3 Implement candidate path/size/naming/static-validation admission and baseline-preserving selection; verify invalid candidates are rejected before probe admission and baseline remains selectable.
- [x] 1.4 Implement selection/scoring separation and explicit `unavailable` / `insufficient-evidence` outcomes; verify selection cannot consume scoring-only data and missing scores are never normalized to zero.
- [x] 1.5 Implement per-attempt cost and status accounting for repeats, retries, cache hits, model calls, and incomplete outcomes; verify totals and outcome axes with deterministic unit tests.

## 2. Replay and contract evidence

- [x] 2.1 Add versioned success, identity-drift, scope-rejection, isolation-unavailable, scoring-unavailable, incomplete-cost, and baseline-selected fixtures; verify each fixture replays without model, tool, network, or production execution calls.
- [x] 2.2 Add diagnostics replay validation for candidate identity, scope, isolation, selection-before-scoring, outcome axes, cost accounting, and idempotency; verify stable classification codes and legacy fixture compatibility.
- [x] 2.3 Add a dedicated contract check script and connect it to quality gates; verify its pass/fail behavior with valid and intentionally drifted fixtures.
- [x] 2.4 Add contract/replay test mappings to `docs/mainline-contract-test-index.md`; verify every new contract requirement maps to a test, fixture, and gate.

## 3. Documentation and verification

- [x] 3.1 Update `docs/development-roadmap.md` with the active bounded candidate-comparison scope, non-goals, and owner-gap follow-up rule; verify it does not expand BXM runtime capabilities.
- [x] 3.2 Document the versioned evaluation inputs, evidence states, isolation boundary, and review-only outputs in the existing evaluation/replay documentation; verify examples do not imply automatic publication or runtime recovery.
- [x] 3.3 Run focused evalcontract and diagnostics replay tests plus the candidate contract gate; verify positive, negative, replay, and idempotency cases pass.
- [x] 3.4 Run `openspec validate evaluate-bounded-harness-candidate-comparisons --strict`, `pwsh -File scripts/check-openspec-documentation-impact.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, and `pwsh -File scripts/check-quality-gate.ps1`; record any environment-blocked command and reason.

## Verification Evidence

- 2026-10-20: `go test ./runtime/evalcontract ./tool/diagnosticsreplay -count=1` passed after the final code changes, including configuration identity drift, nullable zero-quality evidence, conservative baseline selection, and attempt/retry accounting regressions.
- 2026-10-20: `pwsh -NoProfile -File scripts/check-candidate-comparison-contract.ps1` and Git Bash `scripts/check-candidate-comparison-contract.sh` passed against the final code. Both are wired into their platform's quality gate.
- 2026-10-20: an intentionally changed `expected.selected_id` in a temporary fixture made the PowerShell gate exit nonzero with `candidate_comparison_non_actionable`; the temporary fixture was removed afterwards. No production execution was invoked.
- 2026-10-20: independent review found axis-sum overflow and contradictory run/ledger statuses. New `TestBuildCandidateComparisonRejectsAxisOverflow` (including underflow and a valid boundary) and `TestBuildCandidateComparisonRejectsContradictoryAttemptStatus` failed before the fix; both package suites passed after the fix.
- 2026-10-20: OpenSpec strict validation, documentation-impact gate, docs consistency and `git diff --check` passed.
- 2026-10-20: full `pwsh -NoProfile -File scripts/check-quality-gate.ps1` exited 0, completing 83 steps in 1547.37 seconds within its 1800-second budget. It executed `go test ./...` (54.17s), `go test -race ./...` (91.83s), and `golangci-lint run --config .golangci.yml ./...` (29.08s, 0 issues) after the final code fixes. Candidate-comparison contract, 30-mode dual-variant smoke, semantic stability, performance regression, and full-chain smoke all passed.
- 2026-10-20: strict `govulncheck` within the quality gate passed with 0 vulnerabilities affecting called code; it also reported 2 package and 1 module vulnerabilities outside detected call paths. No scan or performance threshold was disabled or weakened.
- Final status synchronization: completing all 13 tasks changes the OpenSpec status to `complete`; the roadmap records this change as completed and awaiting archive, outside its in-progress entries.
- Environment: system `bash` maps to WSL without `/bin/bash`; the shell gate was verified using `C:/Program Files/Git/bin/bash.exe` instead. All required verification completed before archive.
- 2026-10-20: archived using `scripts/openspec-archive-seq.ps1` as sequence 170; four added requirements were synchronized to `openspec/specs/evaluation-corpus-badcase-and-experiment-contract/spec.md`. README and roadmap reflect the archived status.
- Archive verification: the evaluation main spec's inherited Purpose placeholder was replaced with its concrete scope, and its strict validation passed. Full-library strict validation reports 34 passed and 109 failed; all remaining failures are Purpose-placeholder warnings, with the placeholders verified against `master`. These unrelated specs are outside this change. Post-archive `go test ./...` and docs consistency passed.

## Example Impact Assessment

无需示例变更（附理由）：离线评测契约不改变 `examples/agent-modes` 的配置、runtime path、expected markers 或公开运行时行为。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/mainline-contract-test-index.md`、`docs/evaluation-contract.md`
- 维护 owner：evaluation contract maintainers、documentation and contract-test index maintainers
- 验证证据：`openspec validate evaluate-bounded-harness-candidate-comparisons --strict`、`pwsh -File scripts/check-openspec-documentation-impact.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 无需文档变更（附理由） | — | evaluation contract maintainers | 不增加 runtime owner、执行循环或模块依赖 |
| components | 修改文档 | `docs/development-roadmap.md` | roadmap owner | 研究结论与 active proposal 范围已同步 |
| configuration | 修改文档 | `docs/evaluation-contract.md` | evaluation contract maintainers | 记录离线实验配置绑定；不增加生产配置键、默认值或热更新语义 |
| contract/API | 修改文档 | `docs/mainline-contract-test-index.md` | contract-test index maintainers | 增加评测契约与 replay/gate 验证映射 |
| best practices | 修改文档 | `docs/evaluation-contract.md` | evaluation contract maintainers | 明确评测隔离、证据状态与 review-only 边界 |
| diagnostics | 修改文档 | `docs/evaluation-contract.md` | evaluation contract maintainers | 记录 replay-only 证据与 review-only 输出；不改变 RuntimeRecorder 写入 |
| examples | 修改文档 | `docs/evaluation-contract.md` | examples maintainers | 明确无示例变更及不应推断运行时恢复的边界 |
| CLI/integration | 无需文档变更（附理由） | — | integration maintainers | 不新增 CLI 或生产集成入口 |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap owner | Harness² 校准和 proposal 路由已记录 |
| release status | 修改文档 | `README.md` | release documentation maintainers | 同步 active milestone snapshot 与 OpenSpec 状态 |
