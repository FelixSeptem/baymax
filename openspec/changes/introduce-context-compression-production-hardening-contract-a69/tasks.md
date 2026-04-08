## 1. Baseline and Scope Freeze

- [ ] 1.1 （a69-S0-T01）建立 A69 影响面清单：`context/assembler`、`context/journal`、`runtime/config`、`runtime/diagnostics`、`integration`、`scripts`、`docs`。
- [ ] 1.2 （a69-S0-T02）建立 “A69 vs A64 vs A62” 边界映射，明确语义治理、性能优化、示例收口三者职责分离。
- [ ] 1.3 （a69-S0-T03）固化 A69 required suites 基线（contract/replay/benchmark/docs parity），作为后续任务完成前置。

## 2. S1 Semantic Compaction Governance

- [ ] 2.1 （a69-S1-T10）补齐 semantic compaction 质量门槛与结果分级（成功/降级/失败）契约。
- [ ] 2.2 （a69-S1-T11）补齐 rule-based 可压缩对象边界，覆盖工具调用历史项（含最早工具结果）裁剪可用条件与证据保留约束。
- [ ] 2.3 （a69-S1-T12）固化 `best_effort` 与 `fail_fast` 下的 fallback 链路与 deterministic 行为断言。
- [ ] 2.4 （a69-S1-T13）新增/更新单测，覆盖 semantic+rule mixed path 与边界异常分支。

## 3. S2 Tiering and Swap-Back Governance

- [ ] 3.1 （a69-S2-T20）统一 `hot|warm|cold|pruned` 生命周期迁移规则与冲突优先级。
- [ ] 3.2 （a69-S2-T21）将 swap-back 检索升级为“相关性优先 + 新近性次级”排序，并补齐 deterministic tie-break 规则。
- [ ] 3.3 （a69-S2-T22）补齐 tiering/swap-back 的 Run/Stream 语义等价测试与回放断言。

## 4. S3 File Cold-Store Lifecycle Governance

- [ ] 4.1 （a69-S3-T30）新增 file 冷存 `retention/quota/cleanup/compact` 治理策略与默认值。
- [ ] 4.2 （a69-S3-T31）补齐冷存文件损坏、部分写入、超配额等异常处理与 fail-safe 语义。
- [ ] 4.3 （a69-S3-T32）补齐冷存治理单测/integration，验证容量约束与清理行为可预测。

## 5. S4 Recovery and Replay Consistency

- [ ] 5.1 （a69-S4-T40）补齐 crash/restart 场景 spill/swap-back 幂等恢复与去重断言。
- [ ] 5.2 （a69-S4-T41）补齐 replay 一致性断言，确保无第二事实源下语义可复现。
- [ ] 5.3 （a69-S4-T42）新增恢复一致性 integration 场景（中断后恢复、重复恢复、跨模式恢复）。

## 6. S5 Runtime Config and Diagnostics Additive Contract

- [ ] 6.1 （a69-S5-T50）新增/收敛 A69 配置字段（压缩质量门槛、swap-back 排序、cold-store 治理），保持 `env > file > default`。
- [ ] 6.2 （a69-S5-T51）补齐非法配置与非法热更新 fail-fast + 原子回滚校验。
- [ ] 6.3 （a69-S5-T52）新增/收敛 A69 diagnostics 字段（压缩分级、tier 迁移、cold-store 指标、恢复一致性标记），保持 additive + nullable + default。
- [ ] 6.4 （a69-S5-T53）同步更新 `docs/runtime-config-diagnostics.md` 与索引文档中的字段映射。

## 7. S6 Replay and Quality Gate Hardening

- [ ] 7.1 （a69-S6-T60）新增 A69 replay fixture contract（建议 `context_compression_production.v1`）与 canonical fixtures。
- [ ] 7.2 （a69-S6-T61）新增 A69 drift taxonomy（compaction/tiering/swap-back/cold-store/recovery）并接入 replay 解析。
- [ ] 7.3 （a69-S6-T62）新增 `scripts/check-context-compression-production-contract.sh/.ps1` 并接入 `check-quality-gate.*`。
- [ ] 7.4 （a69-S6-T63）将 `check-context-production-hardening-benchmark-regression.*` 纳入 A69 影响面阻断集合并固化 shell/PowerShell parity。

## 8. Validation and Closure

- [ ] 8.1 （a69-S7-T70）执行最小验证：`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`。
- [ ] 8.2 （a69-S7-T71）执行 A69 专项门禁与 replay suites，记录 required-check 对应关系。
- [ ] 8.3 （a69-S7-T72）执行 `pwsh -File scripts/check-docs-consistency.ps1`，若存在未执行项需记录原因与风险。
- [ ] 8.4 （a69-S7-T73）更新 a62 依赖关系说明：context-governed 子项在 A69 收敛后进行最终验收。
