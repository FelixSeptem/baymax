## 1. Gap Fixtures and Test-First Baseline

- [x] 1.1 Add failing `runtime/evalcontract` tests for canonical first-error normalization, equivalent input ordering, missing identity, invalid confidence, collection/size bounds, action-boundary conflict, missing required evidence, duplicate evidence and body-bearing privacy rejection; verify the focused tests fail for the intended missing-contract reasons before implementation.
- [x] 1.2 Add failing comparison tests for step/kind/prefix, owner/cause, action/evidence, recoverability/confidence, correlation and Run/Stream parity drift; verify each expected canonical reason is asserted independently without last-write-wins behavior.
- [x] 1.3 Add versioned `eval_first_error_attribution.v1` success, semantic-drift, memory-application and privacy fixtures plus failing `tool/diagnosticsreplay` tests; verify the fixtures demonstrate that the existing result-only Badcase/replay path cannot express the required attribution boundary.

## 2. Bounded Attribution Contract

- [x] 2.1 Add semantic attribution types, supported version, fixed top-level kind/owner taxonomy, recoverability enum, integer confidence range, bounded cause/evidence/action limits and canonical reason constants in `runtime/evalcontract`; verify positive, negative and boundary tests cover every validator branch.
- [x] 2.2 Implement pure deterministic normalization for first-error identity, ranked causes, prefix digest, decision-boundary sets and source-owned evidence references; verify equivalent unordered inputs produce identical normalized output and digest while malformed or conflicting input fails fast.
- [x] 2.3 Implement side-effect-free baseline/candidate comparison with canonical step, kind, prefix, owner, cause, action, evidence, recoverability, confidence, correlation and parity drift classifications; verify repeated comparison is idempotent and neither input is mutated.
- [x] 2.4 Add explicit privacy and side-effect guards proving attribution code does not persist bodies or call provider, tool, memory backend, resolver, Git, workspace or runtime execution paths; verify focused tests and static gate checks reject forbidden dependencies/content.

## 3. Evaluation Associations and Replay

- [x] 3.1 Add additive + nullable attribution references to Badcase, experiment result and feedback recommendation contracts while preserving historical JSON/default behavior; verify archived payload fixtures retain existing reproduction, aggregation, continuity and approval semantics.
- [x] 3.2 Validate corpus item, Badcase, experiment, run and first-error-step correlation without rewriting either source record; verify matching associations succeed and mismatches emit deterministic attribution-correlation drift.
- [x] 3.3 Keep evidence-linked feedback review-only and non-actionable; verify approved recommendations cannot mutate prompt, Skill, tool, policy, memory, runtime configuration, code, tests, gates or execution decisions.
- [x] 3.4 Implement `eval_first_error_attribution.v1` parsing and evaluation in `tool/diagnosticsreplay` by reusing the shared pure contract; verify success, drift, malformed, privacy and memory-application fixtures pass without live connectivity or side effects.
- [x] 3.5 Run mixed historical and new replay fixtures and verify all archived fixture generations remain parseable, deterministic and backward compatible with no aggregate or taxonomy regression.

## 4. Contract Gates and Documentation

- [x] 4.1 Add semantic shell and PowerShell contract gates for first-error attribution and wire both into `scripts/check-quality-gate.sh/.ps1`; verify both scripts run focused tests, scan forbidden side effects/body persistence and report the same pass/fail semantics.
- [x] 4.2 Update `docs/mainline-contract-test-index.md` with the attribution contract, positive/negative tests, replay fixture and gate mapping; verify every referenced test and script path exists.
- [x] 4.3 Update `docs/development-roadmap.md` from active proposal through delivered/archived state only when implementation evidence exists, preserving the Eval → Provider → Budget ordering and avoiding a second progress source; verify `scripts/check-openspec-roadmap-status-consistency.ps1` passes at each status transition.
- [x] 4.4 Confirm Example Impact Assessment remains `无需示例变更（附理由）`: the change is offline Eval/replay/gate only and does not alter `examples/agent-modes` configuration, runtime path or expected markers; verify no example file changes are present, or stop and complete the mandatory `MATRIX.md` and mode README baseline before any example implementation.

## 5. Verification and Archive Readiness

- [ ] 5.1 Run `gofmt`/`goimports` on changed Go files and verify `git diff --check` reports no whitespace errors.
- [ ] 5.2 Run focused verification with `go test ./runtime/evalcontract -count=1` and `go test ./tool/diagnosticsreplay -count=1`, then run the new shell and PowerShell contract gates; verify all commands pass with deterministic repeated output.
- [ ] 5.3 Run repository gates `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1` and `pwsh -File scripts/check-docs-consistency.ps1`; record any platform waiver explicitly and do not mark completion without required evidence.
- [ ] 5.4 Run `openspec validate --all`, review proposal/design/spec/tasks for placeholder, contradiction, ambiguity and scope drift, and verify the change remains offline, reference-only and apply-ready.
- [ ] 5.5 After every task and gate is complete, archive only through `pwsh -File scripts/openspec-archive-seq.ps1 -ChangeName "establish-eval-first-error-attribution-and-trajectory-boundary-contract"`; verify archive index, roadmap, README milestone and contract-test index are synchronized before merging the feature branch into the latest `master`.
