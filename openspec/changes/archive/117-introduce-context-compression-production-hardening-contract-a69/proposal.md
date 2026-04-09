## Why

A67-CTX 已完成 `reference-first / isolate handoff / edit gate / relevance swap-back / lifecycle tiering` 的语义合同收敛，但 context 压缩在“生产可用”维度仍有明显缺口：

- 语义压缩质量门槛与失败分级不够稳定，容易出现“压缩执行了但收益不可预测”。
- 冷热分层与 swap-back 仍偏实现细节驱动，缺少可审计的检索排序与一致性契约。
- file 冷存缺少系统化 `retention/quota/cleanup/compact` 治理，长期运行下存在容量与恢复风险。
- crash/restart/replay 下的 spill/swap-back 一致性断言不完整，生产回放难以稳定复现。

A69 的目标不是引入新的 context 语义，而是把既有 context 压缩能力收敛为可回归、可观测、可回滚的生产合同，并作为 a62 `context-governed` 示例收口前置。

## What Changes

- 新增 A69 主合同：`context compression production hardening`，在单提案内收敛语义压缩、冷热分层、冷存治理、一致性回放与强门禁。
- 固化 S1 语义压缩治理：
  - 明确 semantic compaction 质量门槛、失败分类与降级链路；
  - 补齐 rule-based 可压缩对象边界（含“最早工具调用结果”类历史项的可裁剪规则）与证据保留约束。
- 固化 S2 冷热分层与回填治理：
  - 统一 `hot|warm|cold|pruned` 迁移判定；
  - 将 swap-back 检索从顺序读取提升为“相关性优先 + 新近性次级”的确定性排序。
- 固化 S3 file 冷存治理：
  - 增加 `retention/quota/cleanup/compact` 治理策略；
  - 明确默认 file backend 的独立可运行与异常恢复边界。
- 固化 S4 一致性与恢复治理：
  - 补齐 crash/restart/replay 的幂等与去重语义；
  - 保持单一事实源，不引入第二状态源。
- 固化 S5 观测与配置治理：
  - 新增/收敛 config 与 diagnostics additive 字段；
  - 保持 `env > file > default`、fail-fast 与原子回滚语义。
- 固化 S6 强门禁治理：
  - 新增 `check-context-compression-production-contract.sh/.ps1`；
  - 与 replay/benchmark 及 `check-quality-gate.*` 形成阻断闭环。
- 明确与 a62 关系：
  - a62 非 context 示例可并行推进；
  - `a62-T15 context-governed-reference-first` 等 context-governed 子项完成判定以后置 A69 收敛为准。

## Capabilities

### New Capabilities

- `context-compression-production-hardening-contract-a69`: 定义 context 压缩生产可用治理边界、回放分类与门禁阻断。

### Modified Capabilities

- `context-assembler-memory-pressure-control`: 增补语义压缩质量门槛、rule-based 压缩边界、冷热分层与回填治理约束。
- `runtime-config-and-diagnostics-api`: 增补 A69 配置字段与诊断字段的 additive 合同。
- `diagnostics-replay-tooling`: 增补 A69 fixture 与 drift taxonomy。
- `go-quality-gate`: 增补 A69 专项门禁、影响面映射与 shell/PowerShell parity 阻断要求。

## Impact

- 上下文核心路径：`context/assembler`、`context/journal`、`context/provider`。
- 运行时配置与诊断：`runtime/config`、`runtime/diagnostics`、`observability/event`。
- 回放与测试：`integration`、`diagnostics replay fixtures`、contract suites。
- 门禁脚本：`scripts/check-context-compression-production-contract.*`、`scripts/check-quality-gate.*`。
- 文档与索引：`docs/development-roadmap.md`、`docs/mainline-contract-test-index.md`、相关 context 文档。
