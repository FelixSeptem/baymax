## 1. Memory SPI and Runtime Config Baseline

- [x] 1.1 新增 `memory` 领域 canonical SPI（`Query/Upsert/Delete`）与统一 request/response/error taxonomy。
- [x] 1.2 新增 `runtime.memory.*` 配置域（`mode`、provider/profile、builtin filesystem、fallback、contract version）。
- [x] 1.3 实现启动与热更新校验（`env > file > default`、fail-fast、原子回滚）。
- [x] 1.4 补齐配置优先级与非法配置单测（含热更新回滚断言）。

## 2. Builtin Filesystem Engine and External SPI Profiles

- [x] 2.1 实现内置文件系统 memory 引擎（append-only WAL、索引快照、原子 compaction）。
- [x] 2.2 实现 `mem0|zep|openviking|generic` profile-pack 解析与 external SPI facade。
- [x] 2.3 实现 memory fallback 策略（`fail_fast|degrade_to_builtin|degrade_without_memory`）与标准 reason code。
- [x] 2.4 补齐并发与崩溃恢复测试（读写竞争、compaction 中断恢复、重复回放幂等）。

## 3. Context Assembler Integration

- [x] 3.1 将 CA2 Stage2 memory 访问统一接入 memory facade，避免主流程 provider-specific 分支。
- [x] 3.2 保持 `fail_fast|best_effort` 现有 stage policy 语义不变，并对齐 memory fallback 策略。
- [x] 3.3 补充 Run/Stream 等价性集成测试（external_spi 与 builtin_filesystem 双路径）。
- [x] 3.4 提供从现有 file-based memory 路径迁移到 SPI/builtin 模式的兼容适配层与断言。

## 4. Observability, Readiness, and Replay

- [x] 4.1 扩展 RuntimeRecorder/diagnostics，新增 memory additive 字段与 bounded-cardinality 约束。
- [x] 4.2 扩展 readiness preflight `memory.*` findings 与 strict/non-strict 映射。
- [x] 4.3 新增 `memory.v1` replay fixtures 与 drift 分类断言。
- [x] 4.4 验证 mixed fixtures 回放兼容（A52 及更早版本 + `memory.v1`）。

## 5. Adapter Contract, Template, and Migration

- [x] 5.1 扩展 adapter manifest memory 字段（provider/profile/contract_version/operations）与兼容校验。
- [x] 5.2 在 external adapter conformance harness 增加 memory matrix suites（mem0/zep/openviking/generic）。
- [x] 5.3 更新 adapter onboarding 模板与迁移映射，覆盖 external SPI 与 builtin filesystem 开关路径。
- [x] 5.4 为模板与迁移条目绑定 conformance case id，防止文档与实现漂移。

## 6. Quality Gate and CI Wiring

- [x] 6.1 新增 `check-memory-contract-conformance.sh/.ps1` 并接入 `check-quality-gate.*`。
- [x] 6.2 将 memory contract gate 作为独立 required-check 候选暴露到 CI。
- [x] 6.3 校验 shell/PowerShell gate parity（失败传播、退出码、阻断语义一致）。
- [x] 6.4 补齐 memory contract suites 在主线 gate 的最小 smoke 与完整 matrix 执行策略。

## 7. Documentation and Roadmap Sync

- [x] 7.1 更新 `docs/runtime-config-diagnostics.md`（memory 配置、诊断字段、fallback 语义）。
- [x] 7.2 更新 `docs/external-adapter-template-index.md` 与 `docs/adapter-migration-mapping.md`（memory onboarding 与迁移）。
- [x] 7.3 更新 `docs/mainline-contract-test-index.md`（memory conformance/replay/gate 索引）。
- [x] 7.4 更新 `docs/development-roadmap.md` 与 `README.md`，记录 A54 范围、验收口径与迁移影响。

## 8. Validation

- [x] 8.1 执行 `go test ./...`。
- [x] 8.2 执行 `go test -race ./...`。
- [x] 8.3 执行 `golangci-lint run --config .golangci.yml`。
- [x] 8.4 执行 `pwsh -File scripts/check-quality-gate.ps1` 与 `pwsh -File scripts/check-docs-consistency.ps1`。
