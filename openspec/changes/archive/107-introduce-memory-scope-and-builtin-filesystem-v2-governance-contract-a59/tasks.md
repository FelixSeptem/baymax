## 1. Runtime Memory Governance Config

- [x] 1.1 在 `runtime/config` 增加 `runtime.memory.scope.*`、`runtime.memory.write_mode.*`、`runtime.memory.injection_budget.*`、`runtime.memory.lifecycle.*`、`runtime.memory.search.*` 字段与默认值。
- [x] 1.2 保持 `runtime.memory.mode=external_spi|builtin_filesystem` 现有语义不变，新增字段独立校验。
- [x] 1.3 补齐启动 fail-fast 与热更新原子回滚校验（枚举/范围/冲突组合）。
- [x] 1.4 增加配置单测（`env > file > default`、非法值、热更新回滚）。

## 2. Scope Resolution and Injection Budget

- [x] 2.1 在 `memory/facade` 增加 `session|project|global` deterministic scope 解析逻辑。
- [x] 2.2 实现注入预算裁剪策略与 deterministic 截断顺序。
- [x] 2.3 在响应与诊断映射中补齐 `memory_scope_selected`、`memory_budget_used`。
- [x] 2.4 补齐 scope/budget 单测（命中顺序、override、预算超限）。

## 3. Write Mode Governance

- [x] 3.1 新增 `runtime.memory.write_mode=automatic|agentic` 的运行策略分发。
- [x] 3.2 固化 automatic/agentic 回填窗口与幂等约束，不改变 SPI 入口签名。
- [x] 3.3 增加 write-mode 负向测试（非法模式、组合冲突、热更新回滚）。

## 4. Search Pipeline Governance

- [x] 4.1 在 memory 查询路径实现 `retrieve -> rerank(optional) -> temporal_decay(optional)` 治理链。
- [x] 4.2 增加 hybrid retrieval 配置解析（keyword/vector 权重）与边界校验。
- [x] 4.3 补齐 `memory_hits`、`memory_rerank_stats` 观测字段映射。
- [x] 4.4 增加搜索质量测试（top-k 命中、冗余率、排序稳定性）。

## 5. Lifecycle Governance

- [x] 5.1 实现 `retention|ttl|forget` 策略校验与执行路径。
- [x] 5.2 增加 lifecycle 执行动作记录（`memory_lifecycle_action`）。
- [x] 5.3 增加 lifecycle 单测（TTL 过期、forget 边界、非法策略 fail-fast）。

## 6. Builtin Filesystem v2 Consistency

- [x] 6.1 在 `memory/filesystem_engine` 增加索引增量更新与全量重建触发策略。
- [x] 6.2 引入 snapshot/WAL/index checksum 校验与 drift detect（`recovery_consistency_drift`）。
- [x] 6.3 实现 drift 后 deterministic 恢复流程（增量优先、全量兜底）。
- [x] 6.4 补齐崩溃恢复与并发读写一致性测试。

## 7. Diagnostics and Recorder Additive Fields

- [x] 7.1 在 `runtime/diagnostics` 扩展 memory additive 字段并保持 `additive + nullable + default`。
- [x] 7.2 在 `observability/event.RuntimeRecorder` 接入 memory 字段映射，保持 single-writer idempotency。
- [x] 7.3 增加 QueryRuns 兼容测试（有/无 memory 字段的稳定行为）。

## 8. Replay Fixtures and Drift Taxonomy

- [x] 8.1 在 `tool/diagnosticsreplay` 增加 `memory_scope.v1`、`memory_search.v1`、`memory_lifecycle.v1` fixture loader 与 normalization。
- [x] 8.2 新增 drift 分类断言：`scope_resolution_drift`、`retrieval_quality_regression`、`lifecycle_policy_drift`、`recovery_consistency_drift`。
- [x] 8.3 增加 mixed-fixture 兼容测试（历史 fixtures + memory governance fixtures）。

## 9. Contract Gate and CI Wiring

- [x] 9.1 新增 `scripts/check-memory-scope-and-search-contract.sh/.ps1`。
- [x] 9.2 将 memory contract gate 接入 `scripts/check-quality-gate.*` 阻断路径。
- [x] 9.3 在 CI 暴露独立 required-check 候选（`memory-scope-search-gate`）。
- [x] 9.4 校验 shell/PowerShell 失败传播语义一致。

## 10. Documentation and Validation

- [x] 10.1 更新 `docs/runtime-config-diagnostics.md`（新增 memory governance 字段与默认值）。
- [x] 10.2 更新 `docs/mainline-contract-test-index.md`（A59 fixtures + gate 映射）。
- [x] 10.3 更新 `docs/development-roadmap.md`、`memory/README.md`、`README.md` 的 A59 状态与配置说明。
- [x] 10.4 执行 `go test ./memory ./runtime/config ./runtime/diagnostics ./observability/event -count=1`。
- [x] 10.5 执行 `go test ./tool/diagnosticsreplay ./integration -run 'TestReplayContract.*Memory' -count=1`。
- [x] 10.6 执行 `go test -race ./...`、`golangci-lint run --config .golangci.yml`、`pwsh -File scripts/check-memory-scope-and-search-contract.ps1`、`pwsh -File scripts/check-quality-gate.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`。
