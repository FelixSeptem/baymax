## Context

Baymax 已有 collector-first OTel tracing、固定 span topology、协议 correlation、最小 agent eval metric，以及 embedded `local|distributed` 执行和幂等聚合。`RuntimeRecorder` 是诊断单写入口，`tool/diagnosticsreplay` 与 shell/PowerShell gate 负责合同回归。当前缺口集中在评测资产和结果治理：同一场景的输入、工具/策略快照和运行状态没有稳定的版本引用，失败样例没有统一复现标识，跨实验结果缺少可审计比较，人工反馈也没有只读投影合同。

本 change 只扩展现有 OTel/eval 同域能力。运行时仍由各 source runtime 拥有执行事实，snapshot、Artifact、Trace 和 policy/memory/budget 仍是事实来源；新增对象只保存引用、版本和聚合结果，不复制内容存储或控制面。

## Goals / Non-Goals

**Goals:**

- 定义 `evaluation_corpus.v1`、`badcase.v1`、`experiment_comparison.v1` 和 `feedback_recommendation.v1` 的可验证数据合同。
- 通过规范化字段和稳定摘要保证 corpus、metric/rubric、Badcase 复现与实验聚合的确定性和幂等性。
- 复用 `run_id`、`step_id`、Trace、Artifact、Checkpoint、policy/memory/budget 输出，保持 Run/Stream 语义等价。
- 为版本不兼容、引用缺失、不可复现、指标/rubric drift、聚合冲突和审批缺失提供稳定 drift/reason 分类。
- 将新增关联以 additive、nullable、bounded 字段写入 `RuntimeRecorder`，并接入 replay、integration、quality gate 与 `tracing-eval-smoke`。
- 保持 local/distributed evaluator 的相同语义；distributed 只增加 shard 组织，不引入远程 scheduler 或托管控制面。

**Non-Goals:**

- 不实现评测 UI、托管评测服务、队列/调度平台、跨租户控制面或外部 corpus/结果存储。
- 不自动修改 prompt、tool、policy、memory 或 runtime configuration；feedback 只产生人工可审阅建议。
- 不复制 Trace、Artifact、Checkpoint 内容，不改变 snapshot 或 sandbox 的所有权。
- 不改变现有 OTel semantic convention、eval v1 指标含义或历史 replay fixture 的默认结果。

## Decisions

### 1. Reference-first 数据模型

Corpus item 保存 scenario/input/tool/policy/runtime snapshot 的版本与内容摘要/引用，而不是嵌入大 payload。Badcase 保存失败分类、最小复现引用和现有运行对象关联；experiment 保存 corpus/rubric 版本、运行批次及聚合结果；feedback 保存 reviewer、审批状态、建议和依据引用。所有引用字段可为空但一旦声明必须通过格式和版本校验。

**考虑过的替代方案：** 将完整输入与 Trace 内容复制到 eval 专用存储。该方案会产生第二事实源、增加脱敏和保留策略风险，因此不采用。

### 2. 确定性规范化与摘要

对 corpus、metric/rubric、Badcase reproduction input 和 experiment shard 采用固定字段顺序、空值默认、枚举小写化和稳定 JSON 规范化，再计算摘要。聚合按稳定 item key 去重；相同 key 的不同摘要分类为 conflict，不允许静默覆盖。weighted mean 与 worst case 复用现有 aggregation 语义，并将 corpus/rubric 版本作为结果的一部分。

**考虑过的替代方案：** 依赖 map 序列化顺序或 wall-clock 排序。它们会使 replay 和 distributed resume 产生非确定结果，因此不采用。

### 3. 只读人工反馈投影

反馈建议只能引用 Badcase、实验差异和 rubric 结果，状态限定为 `pending|approved|rejected`，缺少审批上下文时分类为 `approval_missing`。系统不执行建议，也不将建议写回 prompt/tool/policy/config；后续若需要自动优化必须另立提案。

**考虑过的替代方案：** 直接把建议应用到运行配置。该方案越过 policy/config fail-fast 和回滚边界，拒绝采用。

### 4. 观测与回放接入

`RuntimeRecorder` 增加 bounded nullable 字段（corpus version/id、item id、badcase id、experiment id、rubric version、comparison status、feedback status），并复用既有 redaction/cardinality。diagnostics replay 新增 success 与 drift fixture；旧 fixture 缺失新字段时使用 documented defaults。gate 同时提供 shell/PowerShell 等价路径。

### 5. 配置与兼容性

新增行为相关配置遵循 `env > file > default`、严格解析、非法热更新 fail-fast + 原子回滚；默认关闭任何可能改变现有 eval 执行的扩展开关。所有新增诊断/QueryRuns 字段遵循 additive + nullable + default，未知未来字段忽略而非破坏历史读取。

## Risks / Trade-offs

- **[语料引用失效]** 外部引用可能不可读取 → 只在 replay/compare 时报告 `corpus_reference_unavailable`，保留原始引用和摘要，不伪造通过结果。
- **[版本漂移]** corpus 或 rubric 版本不兼容 → 在 admission/replay 前 fail-fast，返回确定性 drift 分类；兼容窗口由显式版本声明控制。
- **[distributed 聚合差异]** shard 重试或重复提交导致结果变化 → 以稳定 item/shard key 去重，冲突摘要拒绝合并，并测试 local/distributed parity。
- **[隐私与高基数]** input/feedback 可能携带敏感数据 → 仅记录 bounded reference/digest，经既有 RuntimeRecorder 脱敏和 cardinality budget；不把原文写入 OTel 属性。
- **[范围膨胀]** 使用方要求自动修复或托管评测 → 通过 non-goals 和独立 capability gate 拒绝，另立后续 change。

## Migration Plan

1. 先添加数据类型、规范化、校验和 replay 解析器；旧 payload 按空值默认处理。
2. 接入 `RuntimeRecorder`、OTel additive projection、integration/replay fixture 和 contract gate。
3. 更新 `tracing-eval-smoke` 文档与可执行断言，验证 corpus → run/trace → result → feedback 链路。
4. 默认保持扩展关闭；回滚时移除新 projection/gate fixture 即可，既有 eval v1 和 OTel span 不受影响。

## Open Questions

- 首期 corpus/reference 是否只支持本地文件与内存 provider，还是同时定义通用 URI scheme；实现前应冻结最小集合。
- Badcase 分类是否采用固定枚举加可选 `custom_code`，需要在 spec 中限定未知值的兼容行为。
- comparison 输出是否只支持现有两种 aggregation，还是允许 rubric 自定义聚合；首期建议保持现有 `weighted_mean|worst_case`。
