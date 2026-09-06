## Context

Baymax 已有 Context Assembler 的压力分区、受保护证据、spill/swap provenance、语义压缩质量门槛与确定性 fallback，也已有 Context JIT 的 reference-first/isolate handoff、swap-back、lifecycle tiering，以及 unified snapshot、session history、checkpoint 和 diagnostics replay。缺口在于这些能力之间没有一个面向“压缩后继续执行”的稳定交接合同：摘要可以生成，但无法统一判断哪些事实必须保留、哪些引用必须可验证、恢复后如何证明与未压缩路径等价。

本设计把 handoff 视为一次有版本的、可校验的运行态投影，而不是新的消息存储或摘要服务。所有事实仍由现有 Run/Step/Event、Artifact、Checkpoint、Session History、snapshot manifest 和 RuntimeRecorder owner 提供；handoff 只保存有界结构化字段与稳定引用。

## Goals / Non-Goals

**Goals:**

- 定义可版本化、可校验的 handoff schema，覆盖目标、进度、失败尝试、文件/工具事实、策略状态、引用和下一步动作。
- 规定合法压缩切点、不可丢弃证据、事实/推断/待确认分层及引用完整性。
- 将 handoff 生成、质量评估、fallback 和恢复接入既有 context/snapshot/history owner。
- 让 Run 与 Stream 在压缩、恢复、终态与诊断上保持语义等价。
- 提供 deterministic replay fixture、drift 分类和 shell/PowerShell gate。

**Non-Goals:**

- 不复制完整消息历史，不新建 session store、snapshot store 或远程持久化服务。
- 不替代 session history、checkpoint、snapshot manifest、Artifact registry 的事实所有权。
- 不引入 hosted gateway、平台控制面、外部 MQ 或 provider 官方 SDK。
- 不重新定义既有终态、policy precedence、sandbox allowlist、budget admission 或 ReAct 决策语义。

## Decisions

### 1. Handoff 是引用优先的有界结构

定义 `handoff.v1` 结构，至少包含：`run_id`、`source_checkpoint_id`、`cut`、`objective`、`completed`、`pending`、`failed_attempts`、`file_changes`、`tool_results`、`policy_state`、`sandbox_state`、`admission_state`、`references`、`next_actions`、`facts`、`inferences`、`needs_confirmation`、`quality` 与 `fallback`。字段均采用 additive + nullable/default 兼容方式，并带 schema version、source event/checkpoint id 和生成时间。

选择引用优先而非内嵌全文，是为了避免第二事实源和无界内存；引用必须可由既有 owners 校验，无法解析时 handoff 不得标记为可恢复。

### 2. 合法切点与保护规则复用 Context Assembler

handoff 只能在已完成事件、工具调用 finalize、checkpoint 或等价的稳定边界生成；禁止在半个 tool call、未提交 artifact、未结束 policy decision 或未 flush 的 stream delta 中切断。critical/immutable evidence、失败尝试、文件写入结果、终态事件和引用元数据优先级高于可压缩叙述。

选择复用现有 pressure/eligibility/provenance 规则，而不是另建 handoff 专用裁剪器，以避免 Run/Stream 或 memory/file 路径分叉。

### 3. 三类知识分层并强制可解释

`facts` 只能来自已观测事件、工具结果、文件/Artifact/Checkpoint 元数据；`inferences` 必须带来源引用和置信度；`needs_confirmation` 用于无法由来源证明的假设。生成器不得把推断写入事实区，也不得把缺失字段静默填成完成状态。

### 4. 质量门槛与 fallback

质量评估至少检查 schema 校验、必需事实覆盖、受保护证据保留、引用可解析、下一步动作可执行性和事实/推断分层。低于阈值时按既有 compaction mode 选择确定性 fallback：优先保留原始受保护窗口，其次执行规则化裁剪；若仍无法形成可恢复 handoff，则返回明确 fallback reason 并让主 Run 继续使用未压缩上下文（fail-fast 配置除外）。任何失败都必须写入 RuntimeRecorder，禁止静默降级。

### 5. 恢复只消费 handoff 引用，不取得事实所有权

恢复流程先校验 schema、source checkpoint/history 边界和引用，再通过现有 context assembler/reference-first 注入 handoff 内容；需要的正文由 Artifact/Checkpoint/Session History owner 按引用加载。恢复失败在 source mutation 前返回确定性错误，重复恢复保持幂等。

恢复协调器按引用类型委托现有 Artifact、Checkpoint、Session History 和 snapshot owner；handoff 不复制正文。稳定 handoff ID 可由调用方提供给 durable restore-operation store，使不同 Assembler 实例、crash/restart 和 replay 使用同一 operation identity 并在 source owner 访问前返回幂等结果。未显式标记为 finalized 的模型 delta 不得产生 checkpoint 或 flushed-stream 切点；仅在完成事件上的 source-owned `checkpoint_id` 与 `stream_flushed` metadata 才可进入下一次 assembler 请求。

### 6. Replay 与诊断

新增 `context_handoff.v1` fixture，记录压缩前输入摘要、handoff、引用清单、恢复结果和 Run/Stream 对照。drift 至少区分 `handoff_fact_loss`、`handoff_reference_loss`、`handoff_cut_invalid`、`handoff_quality_below_threshold`、`handoff_schema_drift`、`handoff_restore_non_idempotent` 与 `handoff_run_stream_mismatch`。诊断事件统一通过 RuntimeRecorder，QueryRuns 字段只做 additive 扩展。

### 7. 文档与示例先行

更新 context/recovery 运行文档和现有示例，给出压缩前后结构、引用解析、fallback 和 replay 输出；若触及 `examples/agent-modes`，先更新 MATRIX.md 与对应 README 的 semantic anchor、runtime path、expected markers、rollback notes，再实现代码。

## Risks / Trade-offs

- [事实覆盖评估可能误报] → 采用来源引用与必需字段清单双重校验，并在 fixture 中覆盖空值、冲突和未知来源。
- [handoff 字段膨胀导致性能回退] → 结构有界、正文引用优先，质量 gate 同时记录大小/延迟预算并复用 lifecycle tiering。
- [新旧恢复路径产生语义分叉] → 只在既有 assembler/reference-first 接缝注入，强制 Run/Stream parity 与 replay 对照。
- [引用指向已清理的 cold artifact] → handoff 生成时校验 retention/quota 状态；恢复前再次解析，失败则确定性 fallback/错误，不返回“可恢复”假象。
- [配置错误造成不可预测降级] → 新增配置遵循 env > file > default、fail-fast 和热更新原子回滚，并补充正负配置测试。

## Migration Plan

1. 先落地类型、schema validator、quality/fallback reason taxonomy 与 replay fixtures，默认关闭 handoff 恢复接线。
2. 在 context assembler 的稳定切点生成 handoff，并将诊断接入 RuntimeRecorder；保留原有压缩路径作为 fallback。
3. 接入 snapshot/history/checkpoint 引用校验和 reference-first 恢复，完成 Run/Stream、memory/file、crash/replay 对等测试。
4. 更新文档、示例和 gate；在观测到质量与性能满足阈值后再逐步启用配置。
5. 回滚时关闭 handoff 生成/恢复开关，继续使用既有压缩与恢复路径；保留已生成 handoff 作为只读诊断数据，不迁移或删除既有事实源。

## Open Questions

- 默认质量阈值与 handoff 大小预算应沿用 a69 的现有配置，还是增加独立但同源的 alias？实现阶段需通过现有配置审查确定，禁止重复定义成本口径。
- `next_actions` 是否需要结构化 action kind 白名单，还是先限定为引用式文本加可验证 source id？优先采用后者以控制首版范围。

## Example Impact Assessment

修改示例

需要扩展已有 context/recovery 示例，展示压缩前运行状态、handoff.v1、引用保持、恢复结果、质量不足 fallback 和 replay drift 分类；不新增独立示例事实源。
