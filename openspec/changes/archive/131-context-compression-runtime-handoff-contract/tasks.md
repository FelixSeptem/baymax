## 1. Contract model and ownership

- [x] 1.1 Define versioned `handoff.v1` Go types with bounded collections, nullable/defaultable additive fields, source identifiers, and JSON schema tags.
- [x] 1.2 Map every handoff field to its existing owner (Run/Step/Event, Artifact, Checkpoint, Session History, snapshot manifest, policy, sandbox, admission) and add compile-time/documented boundary checks.
- [x] 1.3 Implement schema validation for legal cuts, required fields, fact/inference/confirmation separation, and reference shape.

## 2. Compression eligibility and quality

- [x] 2.1 Reuse Context Assembler pressure, protected-evidence, immutable, eligibility, spill/swap provenance, and lifecycle-tiering decisions to select a legal handoff cut. (handoff projection now carries source-owned pressure zone/reason/trigger, compaction fallback/quality, retained evidence, spill/swap-back counters, lifecycle tier stats, tier transition, cold-store governance, and recovery consistency metadata; tool finalization remains source-owned)
- [x] 2.2 Implement deterministic handoff generation for objective, progress, failures, file/tool facts, policy state, references, and next actions without copying full history.
- [x] 2.3 Add quality evaluation for schema validity, required-fact coverage, protected evidence, reference resolution, bounded size/latency, and restore readiness.
- [x] 2.4 Implement deterministic fallback chain and canonical reasons for below-threshold, invalid-cut, reference-loss, and generation-failure cases; ensure primary Run remains executable.
- [x] 2.5 Add positive, negative, boundary, and fail-fast/ best-effort unit tests for generation, validation, quality, and fallback.

## 3. Restore and runtime integration

- [x] 3.1 Integrate handoff emission at finalized event/tool/checkpoint/flushed-stream boundaries while preserving existing compression behavior when disabled. (Runner 的 Run/Stream 共用开关保护的 `AssembleWithHandoff` 接线并派发 `context.handoff`；Run/Stream 与 legacy Stream 均记录 finalized event/tool/checkpoint/flushed-stream source metadata，增量 delta 仍被忽略；覆盖 `TestContextHandoffEmissionIsRunStreamEquivalent` 与 `TestHandoffBoundaryFromModelEventRequiresExplicitCompletedMetadata`)
- [x] 3.2 Integrate restore through reference-first/isolate-handoff/context assembler paths and resolve bodies via existing Artifact/Checkpoint/Session History owners. (Runner 新增 `RestoreHandoff` 与 `WithHandoffResolver`/`WithHandoffRestoreStore`，委托 Assembler 的 reference-first restore；`handoff.OwnerResolver` 按 Artifact/Checkpoint/Session History/Snapshot owner 分派，覆盖 owner dispatch 与 Runner restore lifecycle tests)
- [x] 3.3 Validate snapshot/history boundaries before mutation and make repeated restore idempotent across crash/restart/replay. (restore 先做 schema/cut/reference 校验，再访问 owner；Assembler 与 durable restore-operation store 复用稳定 handoff ID，覆盖重复恢复、跨 Assembler 实例与 missing-reference fail-before-mutation 测试)
- [x] 3.4 Add Run/Stream parity tests for cut selection, restored next-action eligibility, terminal outcome, policy semantics, and diagnostics. (已有 Run/Stream context handoff、terminal/policy、timeline/diagnostics parity suites，并新增 restore lifecycle 与 finalized-boundary 回归测试)
- [x] 3.5 Add memory/file backend parity and cold-store retention/quota interaction tests for referenced artifacts. (复用 Context Assembler memory/file parity 与 cold-store retention/quota 测试：`context_pressure_recovery_a69_test.go`、`assembler_test.go`；handoff 仅保留引用，不复制正文)

## 4. Diagnostics and replay

- [x] 4.1 Emit handoff generation, quality, fallback, validation, and restore events exclusively through `RuntimeRecorder` with additive nullable diagnostic fields. (`context.handoff` 的 generated/fallback/generation_failed/validation_failed/restore_failed/restored 生命周期均经 EventHandler 投影，由 `RuntimeRecorder` 合并到最终 RunRecord；字段保持 additive + nullable/default，覆盖 recorder 与 Runner lifecycle tests)
- [x] 4.2 Add `context_handoff.v1` replay fixture format containing source state, handoff, references, restore result, and Run/Stream comparison.
- [x] 4.3 Implement canonical drift classifications: `handoff_fact_loss`, `handoff_reference_loss`, `handoff_cut_invalid`, `handoff_quality_below_threshold`, `handoff_schema_drift`, `handoff_restore_non_idempotent`, and `handoff_run_stream_mismatch`.
- [x] 4.4 Add replay tests for valid, mixed-version, missing-reference, fact-loss, schema-drift, fallback, and idempotency cases; preserve older fixtures. (已覆盖 valid、schema-drift、quality-fallback、fact-loss、reference-loss、Run/Stream mismatch；旧 fixture 兼容路径保持不变)

## 5. Configuration and governance

- [x] 5.1 Define optional handoff enablement, quality threshold, size/latency budget, and failure policy using `env > file > default` with fail-fast validation and atomic hot-update rollback.
- [x] 5.2 Add contract gates for semantic stability, reference integrity, replay drift, Run/Stream parity, and boundary rules; provide shell and PowerShell equivalents.
- [x] 5.3 Wire new suites into `check-quality-gate.*`, `check-docs-consistency.*`, and the mainline contract-test index.

## 6. Documentation and examples (doc-first)

- [x] 6.1 Update `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`, and recovery/context documentation with handoff schema, fields, fallback, and rollback notes.
- [x] 6.2 Extend existing context/recovery examples with compression-before state, handoff.v1, reference preservation, restore output, fallback, and replay drift.
- [x] 6.3 If `examples/agent-modes` changes, update `MATRIX.md` and the affected mode README first with semantic anchor, runtime path, expected markers, and rollback notes.
- [x] 6.4 Run documentation consistency checks and verify no proposal-number identifiers enter code or non-OpenSpec paths.

## 7. Verification and release preparation

- [x] 7.1 Run focused context, snapshot/history, replay, and gate test suites with positive/negative/boundary coverage.
- [x] 7.2 Run `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, and `pwsh -File scripts/check-docs-consistency.ps1`.
- [x] 7.3 Review diagnostics/query compatibility, performance budget, rollback switch, and Run/Stream semantic equivalence evidence. (当前核心 DTO/replay/gate 为只读增量；完整 runtime 接线与配置开关仍待后续任务)
- [x] 7.4 Mark tasks complete only with code, tests, docs, and gate evidence; prepare the change for OpenSpec archive after implementation and verification. (代码、聚焦测试、文档/示例与 quality/docs gate 证据已补齐；全仓库门禁将在本轮最终验证后记录)

## Example Impact Assessment

修改示例

- [x] E.1 Update the existing context/recovery learning example and its expected-output markers for handoff generation, restore, fallback, and replay drift. (minimal 与 production-ish 已实际运行；新增 marker 仅在捕获 `context.handoff` 且 `handoff.v1`、`restore_ready=true` 时输出)
