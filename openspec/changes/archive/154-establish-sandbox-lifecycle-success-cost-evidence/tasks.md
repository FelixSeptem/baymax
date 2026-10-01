## 1. Contract Model and Normalization

- [x] 1.1 Add failing focused tests for `sandbox_lifecycle_success_cost_evidence.v1` validation, including required facts, supported enums, bounded collections/buckets, opaque references, prohibited-field rejection, and canonical digest stability; verify the focused test initially fails for the missing contract behavior.
- [x] 1.2 Implement the bounded evidence input, normalized projection, canonical digest, and deterministic validation-reason taxonomy under the offline tooling boundary; verify focused tests cover valid, malformed, oversized, unknown-additive, and privacy-negative inputs.
- [x] 1.3 Implement immutable lifecycle continuity normalization for acquire, launch, execute, retry, release, recover, and terminal outcome, including cold launch, per-session reuse, recovery/resume, invalid ordering, duplicate release, invalid retry, and reuse-after-terminal; verify focused unit tests cover each positive and negative path.
- [x] 1.4 Implement terminal-success-only bounded cost summary and three-verdict evaluation (`within-baseline`, `lifecycle-gap-confirmed`, `insufficient-evidence`) using only supplied buckets and baseline facts; verify tests prove failures, missing facts, and contradictions never fabricate a successful-task cost.

## 2. Replay and Parity Evidence

- [x] 2.1 Add failing replay tests and versioned `sandbox_lifecycle_success_cost_evidence.v1` fixtures for canonical within-baseline, lifecycle gap, insufficient evidence, launch/recovery/cleanup continuity, and terminal outcome accounting; verify they initially expose absent replay dispatch/evaluation behavior.
- [x] 2.2 Integrate the pure evaluator into `tool/diagnosticsreplay` without changing historical fixture handlers; verify valid fixtures return stable projection, verdict, reason ordering, bounded cost summary, and canonical digest offline.
- [x] 2.3 Add replay drift assertions for `sandbox_lifecycle_phase_drift`, `sandbox_lifecycle_continuity_drift`, `sandbox_success_cost_bucket_drift`, `sandbox_lifecycle_baseline_drift`, `sandbox_lifecycle_terminal_outcome_drift`, `sandbox_lifecycle_run_stream_parity_drift`, and `sandbox_lifecycle_privacy_drift`; verify each intentional mismatch fails with its stable classification.
- [x] 2.4 Add repeated-replay, unknown-additive-field, and mixed historical-fixture compatibility coverage; verify equivalent Run and Stream evidence has equivalent output and all supported historical fixtures remain deterministic.

## 3. Reference-Only Boundary and Contract Gates

- [x] 3.1 Add contribution boundary tests that reject executor/backend/platform-driver/proxy/service-account/credential/filesystem-policy/network-policy/global-session-store/runtime-config/diagnostics-persistence/admission-state ownership in the evidence path; verify a representative forbidden dependency or identifier is detected deterministically.
- [x] 3.2 Add `scripts/check-sandbox-lifecycle-success-cost-evidence-contract.sh` that runs the focused evidence, replay, parity, privacy, mixed-fixture, and boundary suites without launching an OS sandbox or requiring platform privileges; verify its success and injected-failure behavior.
- [x] 3.3 Add the equivalent `scripts/check-sandbox-lifecycle-success-cost-evidence-contract.ps1` and register both paths in the repository quality gate; verify shell and PowerShell execute equivalent assertions and have equivalent pass/fail semantics on the same repository state.
- [x] 3.4 Confirm the implementation reuses `integration/sandboxconformance` only as test support where needed and does not promote it to a production session manager; verify the boundary test and targeted package tests pass.

## 4. Documentation and Example Assessment

- [x] 4.1 Update `docs/development-roadmap.md` to record the evidence baseline scope, later runtime/platform trigger, owners, non-goals, and rollback point; verify roadmap status consistency succeeds.
- [x] 4.2 Update `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`, and `README.md` with the offline evidence contract, privacy boundary, replay/gate mapping, and no-runtime-config/no-persistence statement; verify documentation consistency succeeds.
- [x] 4.3 Record the Example Impact Assessment as `无需示例变更（附理由）` in proposal, design, and this task list: this offline evidence-only change does not alter `examples/agent-modes` configuration, runtime path, or expected markers; verify `check-openspec-example-impact-declaration` passes and do not change agent-mode examples.
- [x] 4.4 Document the Anthropic `sandbox-runtime@5d196e0` assessment as a boundary reference only, retaining lifecycle/correlation lessons while excluding platform executors, proxies, policy drivers, accounts, and credentials; verify the documentation names no imported implementation dependency.

## 5. Verification and Rollback Evidence

- [x] 5.1 Run focused `tool/diagnosticsreplay`, `tool/contributioncheck`, and sandbox conformance/support package tests with `-count=1`; verify all evidence, replay, privacy, parity, and boundary cases pass.
- [x] 5.2 Run `go test ./... -count=1 -timeout 15m`, `go test -race ./... -timeout 20m`, and `golangci-lint run --config .golangci.yml`; verify no regression, race, or lint failure is introduced.
- [x] 5.3 Run `pwsh -File scripts/check-sandbox-lifecycle-success-cost-evidence-contract.ps1`, its shell counterpart, `pwsh -File scripts/check-quality-gate.ps1`, and `pwsh -File scripts/check-docs-consistency.ps1`; verify all blocking gates pass without sandbox privileges or live platform execution.
- [x] 5.4 Run `openspec validate establish-sandbox-lifecycle-success-cost-evidence --strict`, `openspec validate --all`, `pwsh -File scripts/check-openspec-example-impact-declaration.ps1`, `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1`, and `git diff --check`; verify the change is archive-ready and retains the documented no-migration rollback of evaluator, fixtures, replay dispatch, gates, and documentation.

## Example Impact Assessment

无需示例变更（附理由）：本 change 仅新增离线、宿主或测试支持层提供的 sandbox lifecycle evidence 评估与回放门禁，不改变 `examples/agent-modes` 的配置、runtime path、expected markers 或实际 sandbox executor 行为。任何后续 confirmed gap 若要引入 sandbox session、平台 driver 或执行策略，必须先以独立 change 完成 `MATRIX.md` 和受影响模式 README 的文档基线。
