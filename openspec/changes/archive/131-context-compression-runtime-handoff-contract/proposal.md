## Why

长任务在上下文压缩或进程恢复后，当前运行态容易只保留摘要文本，导致下一步动作、文件事实、失败尝试以及 Artifact/Checkpoint/Session History 关联无法被可靠恢复。现有 Context Assembler、压缩质量治理和 snapshot/history 能力已经具备基础，但缺少一个可校验、可回放、与 Run/Stream 等价的“运行交接单”合同；现在补齐该合同，可以把压缩从“生成摘要”收敛为“可恢复的结构化事实交接”，并为后续性能与恢复治理提供稳定边界。

## What Changes

- 新增上下文压缩运行交接单合同：定义合法消息切点、受保护证据、结构化 handoff schema、版本与 nullable/default 兼容规则。
- 记录任务目标、已完成/未完成事项、失败尝试、文件变更、工具结果、policy/sandbox/admission 状态，以及 Artifact、Checkpoint、Session History 引用和可恢复的下一步动作。
- 明确事实、推断、待确认事项的边界，并为消息/状态设置不可丢弃与引用完整性规则。
- 将压缩、交接、恢复接入既有 Context Assembler、reference-first、spill/swap-back、lifecycle tiering 与 snapshot/history owner，不复制完整消息历史或建立第二套存储。
- 增加确定性的质量门槛与 fallback：压缩质量不足或生成失败时保留主 Run 可继续运行，并输出可诊断、可回放的 fallback 分类；禁止静默产生不可恢复的 handoff。
- 扩展 diagnostics replay、contract fixtures 和 shell/PowerShell gate，检测事实丢失、引用丢失、schema 漂移及 Run/Stream 语义不等价。
- 更新 context/recovery 学习示例和运行文档，展示压缩前状态、handoff、恢复、引用保持与 fallback/replay drift。

## Capabilities

### New Capabilities

- `context-compression-runtime-handoff`: 定义压缩运行交接单的数据模型、生成/校验/恢复语义、质量门槛、fallback、诊断与回放合同。

### Modified Capabilities

- `context-assembler-memory-pressure-control`: 增加压缩输出必须满足 handoff 证据保留、合法切点与失败回退要求。
- `context-assembler-production-convergence`: 将 handoff 质量评估、fallback 原因和恢复可用性纳入既有生产压缩治理。
- `jit-context-organization-and-reference-first-assembly-contract`: 规定 handoff 与 reference-first/isolate handoff/swap-back/lifecycle tiering 的衔接与引用保持。
- `unified-state-and-session-snapshot-contract`: 明确 handoff 只引用既有 snapshot/checkpoint 事实源，并在恢复时保持引用稳定。
- `session-history-checkpoint-replay`: 增加 handoff 恢复对 Session History、Checkpoint 与 replay 的一致性要求。
- `diagnostics-replay-tooling`: 增加 handoff 事实完整性、引用完整性、fallback 与压缩漂移的 replay 分类和 fixture 要求。

## Impact

- 受影响模块：`context/*` 压缩与装配流程、runtime runner/recovery 接缝、snapshot/session-history/checkpoint 引用投影、`observability`/diagnostics replay 与 contract gate。
- 需要新增可选、可版本化的 handoff 结构和诊断字段；现有 QueryRuns、snapshot、history 字段保持 additive + nullable + default 兼容。
- 不改变既有 Run/Stream 终态语义，不引入 provider SDK、远程持久化、外部消息总线或平台化控制面；诊断写入继续只通过 `RuntimeRecorder`。
- 配置若扩展 handoff/quality/fallback 选项，遵循 `env > file > default`、非法值 fail-fast 与热更新原子回滚。

## Example Impact Assessment

修改示例

扩展已有 context/recovery 示例，展示压缩前运行状态、结构化 handoff、从 handoff 恢复以及 Artifact/Checkpoint/Session History 引用保持；同时加入质量不足 fallback 和 replay drift 分类示例。
