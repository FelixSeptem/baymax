## Example Impact Assessment

无需示例变更（附理由）：本设计只建立离线审计、fixture、replay、benchmark 和 gate，不改变 `examples/agent-modes` 的 runtime path、配置键、expected markers 或工具执行语义。后续若把某个策略接入运行时，必须另开范围评审并先完成示例文档基线。

## Context

参见 `proposal.md` 的 Why。当前 `core/types.Tool` 暴露 `JSONSchema()`，本地 registry、MCP metadata、adapter manifest/capability、allowlist、sandbox 和 tool lifecycle 已分别拥有工具身份与准入事实，但仓库没有一个跨来源、provider-neutral 的 schema 压力与工具选择质量审计边界。

设计必须保持以下硬约束：审计只接收宿主已经准入的 bounded snapshot；不调用 Tool、MCP、Provider 或 tokenizer；不访问网络、文件、时钟或 credential；不写 RuntimeRecorder；不改变 `ModelRequest`、ReAct、scheduler、allowlist、sandbox 或执行路径。

## Goals / Non-Goals

**Goals:**

- 定义 `tool_schema_pressure_selection_audit.v1` 的稳定输入、归一化输出、digest、限制和 reason taxonomy。
- 用同一份已准入工具快照同时计算 schema pressure 与 synthetic task 的 selection quality。
- 允许多个版本化 deterministic strategy 在离线环境中比较，并报告 pressure/quality trade-off。
- 允许 Eval corpus/Badcase 作为缺省可选的 advisory evidence，不影响强制 synthetic gate。
- 为 replay、benchmark、Run/Stream parity、历史默认值和 gate 提供可验证契约。

**Non-Goals:**

- 不向 provider 或 `ModelRequest` 投影工具子集。
- 不实现运行时 selector、router、tool registry、marketplace、动态下载或 credential store。
- 不引入 provider tokenizer、embedding、模型调用或任意用户代码策略。
- 不修改工具准入、allowlist、sandbox、Skill/MCP discovery、ReAct 或执行状态机。

## Decisions

### 1. 使用独立的 snapshot DTO，而不是直接消费 `types.Tool`

审计输入包含工具 identity、source、canonical schema digest/length、capability labels、admission facts 和 priority 等已归一化字段。审计不会调用 `JSONSchema()` 或任何工具方法，避免把执行对象、动态副作用或 provider/MCP 生命周期带入离线路径。相比直接复用 registry，snapshot DTO 牺牲少量组装工作，换取 replay 可移植性、隐私边界和稳定 digest。

### 2. Canonical schema 与 token estimate 都版本化

schema 先执行有界 JSON canonicalization，再计算 SHA-256 digest 和 byte length。token estimate 使用明确的 provider-neutral estimator version（首版为 `utf8_bytes_div4_v1`：`ceil(UTF-8 bytes / 4)`），结果只用于比较，不声称等同任何 provider tokenizer。canonical bytes、estimate、estimator version 都进入输出 digest；超过大小或数量上限时 fail-fast，不产生部分结果。

### 3. 策略采用有限 enum + bounded parameters

首版支持 `full_admitted_set`、`capability_filtered_set`、`priority_top_k`、`source_partitioned_set` 和 `fixture_declared_strategy` 五类策略。策略参数只能来自版本化 fixture：capability label、`k`、source group 与 tie-break policy 都有长度/数量上限。禁止任意脚本、表达式或 callback，确保同一输入总能得到相同候选集合与排序。

### 4. Synthetic quality 是强制证据，corpus 是 advisory

synthetic case 必须声明 expected、allowed、forbidden 和 fallback 集合，并计算 precision、recall、F1、expected-hit、forbidden-hit 与 fallback coverage。Eval corpus/Badcase 只提交 bounded metadata 和 aggregate coverage/trend；缺失、unknown field 或无标注 case 只能降低 evidence sufficiency，不能改变强制 gate 结果。这样保留真实语料价值，同时避免外部 corpus 变动破坏确定性门禁。

### 5. `projection_candidate` 只是一种离线结论

只有 baseline pressure 超阈值、baseline quality 退化且至少一个策略在 pressure 和 quality 上都确定性改善时，才输出 `projection_candidate`。该结论不携带可直接接入 runtime 的 tool subset，不写配置、不触发 routing、不修改 registry；它只为后续新的 OpenSpec change 提供证据。

### 6. Replay 与 gate 采用 additive、nullable、default 兼容

历史 fixture 缺少 strategy、corpus advisory 或 estimator fields 时使用 documented defaults；unknown fields 忽略。Replay 比较 canonical digest、strategy order、metrics、conclusion、generation-like snapshot id 和 Run/Stream parity，并为 schema drift、metric drift、strategy drift、overflow、gold conflict 与 insufficient evidence 返回稳定码。

## Risks / Trade-offs

- [估算 token 与真实 provider tokenizer 不一致] → 明确标注 estimator version，只比较同版本相对趋势，不把 estimate 当作 provider usage。
- [synthetic gold 与真实任务分布不一致] → 强制 gate 使用 synthetic，另以 corpus advisory 输出覆盖率和偏差；不自动把 advisory 升级为门禁结论。
- [策略评分被误用为运行时 selector] → 输出只包含 bounded metrics/reasons，不输出 runtime wiring；contribution gate 扫描 `ModelRequest`、provider、registry 和 runtime config 接线。
- [工具 schema 过大导致 fixture/diagnostics 泄露] → 只持久化 digest、长度、计数和有限标签；raw schema、prompt、模型输出和 tool result body 禁止进入 fixture。
- [Run/Stream 采集路径产生不同快照] → 输入契约显式包含 path marker，replay 要求语义字段等价，差异只允许出现在非语义 timing。

## Migration Plan

这是离线新增能力，没有运行时迁移或回滚步骤。先提交 contract/spec、synthetic fixture、replay、benchmark 和双平台 gate；若结果为 `baseline_sufficient`、`pressure_only`、`quality_only` 或 `insufficient_evidence`，保持当前运行时不变。只有后续独立提案确认 `projection_candidate` 的真实宿主价值后，才评估运行时接线。

## Open Questions

无。estimator、策略枚举、强制/ advisory 分层和 conclusion 条件已在本设计中固定。
