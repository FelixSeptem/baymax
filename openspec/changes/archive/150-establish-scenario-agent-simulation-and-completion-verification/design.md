## Context

现有 `integration/fakes`、Run/Stream contract、completion safe-point、`runtime/evalcontract` 和 `tool/diagnosticsreplay` 已分别拥有可复用的测试事实，但没有一个跨边界的场景模型来描述事件编排、故障注入、恢复和完成验证。实现必须继续遵守现有 owner：`core/runner` 拥有运行终态，completion safe-point 拥有后台完成 promotion，`RuntimeRecorder` 是诊断单写入口，eval/replay 只做离线归一化，业务 Outcome 由宿主提供。

本设计只增加测试支持和 reference-only 审计投影。它不改变生产执行循环、provider adapter、配置语义、事件排序、终态优先级或任何现有事实存储。

## Goals / Non-Goals

**Goals:**

- 提供稳定、版本化、可规范化的 Scenario profile 和 bounded Run Result。
- 用同一个场景描述驱动 Run 与 Stream 的等价测试，并支持受控审批、取消、Stream 截断、恢复和 completion promotion 组合。
- 将 execution、evidence、business outcome、release admission 四个结论分开建模，明确 `indeterminate`。
- 为成功、schema drift、causation drift、evidence conflict、late/duplicate completion、privacy violation 和 admission mismatch 提供 deterministic replay 分类。
- 让 Shell/PowerShell gate、fixture、replay 和 contract index 能够追溯到同一 capability contract。

**Non-Goals:**

- 不提供生产运行时 Simulation API、通用 scenario scheduler 或新的 terminal state machine。
- 不连接 live provider、网络、Git、workspace、hosted artifact/state 或外部 resolver。
- 不替代业务 Outcome owner、RuntimeRecorder、completion safe-point、evaluation corpus 或 diagnostics replay 的既有职责。
- 不修改 `examples/agent-modes`，不增加 runtime 配置键，不建设 registry/marketplace/动态下载。

## Decisions

### 1. 使用 Scenario → Event Plan → Run Result → Verification 的四段式数据流

Scenario 是输入契约，包含稳定 identity/version、受控参与者引用、事件计划、故障/恢复动作和预期证据引用。规范化后生成 scenario digest，并将计划事件按显式 sequence/causation 标识排序。

测试 builder 只负责把规范化 Scenario 转换成既有 Run/Stream 测试入口可消费的 fake 输入；它不拥有运行状态或终态。执行输出只收集 bounded event references、terminal/completion classifications 和 source-owned correlation。

Verifier 将输出归一化为四个独立维度：

1. execution：是否达到 source-owned terminal；
2. evidence：要求的引用是否存在、匹配且不冲突；
3. business outcome：宿主是否提供可验证的结果引用；
4. release admission：场景声明的发布条件是否满足。

缺失或冲突时使用 `indeterminate`/明确 drift reason，不让一个维度替代另一个维度。

**替代方案：**直接把场景字段加进 runtime 状态机会减少测试胶水，但会引入生产耦合和第二套事实源；仅保留独立 fixture 则无法消除重复 setup。因此采用测试 builder + reference-only verifier 的组合。

### 2. Scenario 和 Result 使用 bounded reference-first schema

所有身份、事件、证据和结果字段都使用稳定 ID、digest、version、owner、correlation 和 reason code；对事件总数、证据数量、字符串长度和序列化大小设置上限。未知 additive 字段安全忽略，历史 payload 缺少新字段时使用 nullable/default。

禁止字段包括 raw reasoning、transcript、provider/tool/memory/workspace body、credential 和无界 map。证据只表达“由哪个 owner 的哪个引用支持结论”，不在仿真结果中复制源内容。

**替代方案：**保存完整模拟 transcript 便于调试，但会破坏隐私边界、增加 digest 漂移和存储成本；采用引用加 fixture 片段可以保留可审计性而不建立新事实源。

### 3. Builder 只做确定性事件编排，不做策略决策

builder 提供测试专用的声明式动作：fake model/SSE chunk、tool result、approval decision、cancel、fault injection、stream truncation 和 recovery replay。每个动作必须绑定稳定 identity、目标 owner 和预期 causation；builder 不计算 policy、approval、terminal precedence 或 release decision，这些仍由现有 runtime/contract owner 产生。

为防止误用，builder 放在 integration/test-support 作用域，API 不被 runtime、model/provider、context 或生产 binary 引用；gate 通过静态依赖检查和离线执行保证这一点。

**替代方案：**在生产 `runtime` 包中暴露 simulation hooks 会让宿主误以为可用来驱动真实执行；拒绝该方案以保持 library-first 和模块边界。

### 4. Replay 采用单向规范化和冲突拒绝

replay 输入为版本化 scenario/result fixture。流程为：schema/bound 检查 → canonical normalize → source-owned correlation 校验 → expected/observed digest 比较 → drift classification。相同 identity 与 digest 的重复输入幂等忽略；相同 identity 的不同 digest 直接返回 duplicate conflict，不采用 last-write-wins。

Replay 只读 fixture 和 bounded references，不恢复 runtime、不调用 provider/tool/memory、不写 diagnostics。若需要诊断投影，使用现有 RuntimeRecorder contract fixture 验证单写路径，而不是在 replay 内新建 writer。

### 5. Run/Stream parity 以语义归一化为准

同一 Scenario 分别通过 Run 和 Stream 执行，比较 execution/evidence/completion/terminal/admission 维度以及 correlation、reason 和 digest。Stream 特有的增量 chunk、heartbeat 或 event ordering 先按既有规范归一化，再比较语义字段；不要求字节级事件序列相同。

截断、取消、晚到 completion 和重复 completion 由现有 safe-point/terminal owner 决策，verifier 只检查其结果是否符合既有分类、是否只 promotion 一次以及是否恢复后不 resurrect terminal Run。

### 6. Gate 和文档保持双平台、可追溯

新增独立的 Go contract/replay suite、版本化 fixture、PowerShell 与 Shell gate。gate 输出固定 success/drift taxonomy，并检查 offline/privacy/no-runtime-dependency 约束。`docs/mainline-contract-test-index.md` 映射 capability → tests → replay → gates；`docs/runtime-harness-architecture.md` 只补充 test-support boundary；roadmap 记录进行中状态和触发理由。

## Risks / Trade-offs

- **[Risk] Builder 逐渐复制生产运行时逻辑。** → builder 只编排输入动作，不实现策略、终态或恢复算法；增加静态依赖和 contract gate。
- **[Risk] `indeterminate` 被调用方当作失败或成功。** → 在 schema 中将四个结论分开，要求 verifier 和 fixture 显式断言每个维度；文档标明业务 Outcome 仍由宿主决定。
- **[Risk] Fixture 过大或包含敏感内容。** → 只允许 bounded references/digests，拒绝 body-bearing evidence，执行大小/数量上限和 privacy negative cases。
- **[Risk] Run/Stream 测试出现平行语义。** → 复用同一 Scenario，比较既有 source-owned classifications，并禁止 builder 维护第二套事件序列或 terminal 状态。
- **[Risk] 新 gate 增加质量门禁耗时。** → 首阶段 fixture-only、限定受影响包，并在 gate 中记录耗时和 ROI；超预算只走既有门禁降级/回滚路径，不放宽阈值。
- **[Risk] 历史 payload 无法解析。** → 所有新字段 additive + nullable + default，replay 保留旧 fixture 套件并加入 legacy payload 测试。

## Migration Plan

1. 先实现 schema normalizer、bounded limits、reason taxonomy 和纯函数 verifier，并以 fixture 锁定正负路径。
2. 在测试支持作用域接入 builder，先覆盖单步和组合的 Run/Stream parity，再覆盖 truncation、approval wait、cancel、recovery、late/duplicate completion。
3. 接入 diagnostics replay 和双平台 gate，补齐 contract index、harness architecture、roadmap 和提案任务证据。
4. 运行完整 quality gate；若发现 builder 越界或语义漂移，回滚新测试支持、fixture、replay、gate 和文档，不需要数据迁移或 runtime 回滚。

## Example Impact Assessment

`无需示例变更（附理由）`：设计只作用于离线测试支持和审计 replay，不改变 `examples/agent-modes` 的配置、runtime path、expected markers 或业务 Outcome；若实施阶段发现必须修改示例，必须先完成 `MATRIX.md` 与对应 README 文档基线并重新评估。
