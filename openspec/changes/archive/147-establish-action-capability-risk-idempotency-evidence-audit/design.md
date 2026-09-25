## Example Impact Assessment

无需示例变更（附理由）：设计只增加离线审计合同、fixture/replay 与 gate，不改变任何 `examples/agent-modes` 的执行路径、配置、事件标记或回滚说明。

## Context

See `proposal.md` for motivation. 当前 Action Gate、Tool lifecycle、Tool security、Action timeline 和 diagnostics replay 已分别拥有 source-owned 的执行、授权、阶段和事件事实，但这些事实没有被统一成一个用于审计 Action capability 声明的离线输入。仓库硬约束要求：运行时状态不能出现第二套事实源，诊断写入必须经过 `observability/event.RuntimeRecorder`，Run/Stream 必须语义等价，新增 schema 必须 additive + nullable + default，且不应建设平台控制面。

本设计把审计定位为 diagnostics replay 的一个纯函数能力：宿主先提供已经准入的 bounded snapshot，审计只验证声明、引用和有限观察是否自洽，不能自行发现能力、执行动作或判定业务 Outcome。

## Goals / Non-Goals

**Goals:**

- 定义版本化 `action_capability_audit.v1` 输入、归一化输出、审计 verdict 和 drift taxonomy。
- 覆盖 effect、副作用、risk、reversibility、idempotency、preconditions、timeout/retry、owner/scope/version 以及 Preview/Approve/Commit/Verify 证据引用。
- 区分 intent、issued、confirmed 三种事实，识别“已发出但未确认”与“可以安全重试”的差异。
- 复用既有 Tool lifecycle、Action Gate、Security、Timeline 和 Run/Stream projection 的 bounded reference，不复制其状态机或诊断写入路径。
- 交付 deterministic normalizer、正负/边界 fixture、replay、shell/PowerShell 对等 gate 和文档索引。
- 保持历史 fixture 可解析，缺失扩展字段按 nullable/default 兼容处理。

**Non-Goals:**

- 不修改 Runner、Dispatcher、MCP、Provider、Policy、Sandbox、Action Gate 或真实工具执行语义。
- 不建立 Tool/Action registry、marketplace、动态下载器、credential store、全局 action queue 或 hosted execution/session store。
- 不执行 Preview、Approve、Commit、Verify、补偿、撤销、重试或任何外部副作用。
- 不采集或存储 raw prompt、reasoning、凭证、完整命令输出、工具参数正文、响应正文或无界 payload。
- 不把审计 verdict 直接写入 RuntimeRecorder、RunRecord、runtime config、Policy、Skill、Tool registry 或业务 Outcome。

## Decisions

### 1. 选择 offline audit-first，而不是 runtime-integrated enforcement

**决策：** 将能力放在 `tool/diagnosticsreplay` 的离线归一化/审计路径，并由 `tool/contributioncheck` gate 调用。任何 `issued` 未关联 `confirmed` 的操作（含只读动作）都不得判为 compliant；高风险四阶段必须有可接受的正向状态，拒绝、超时、失败或未知均不算阶段完成。

**理由：** 当前触发证据是 Handbook 提出的元数据缺口，而不是已证明的执行器错误。先锁定 fixture、分类和 evidence sufficiency，能够测量 gap 而不改变副作用语义，符合 library-first 与先 fixture/replay/gate 后 runtime 的 roadmap 顺序。

**替代方案：**

- 在 `core/runner` 或 `tool/local` 中强制新增 metadata 并阻断运行：会把尚未证实的声明缺口变成行为变更，扩大回滚和兼容风险，拒绝。
- 新建 Action registry/control plane：会复制 extension/tool ownership，引入动态发现、凭证和平台化状态，违反 roadmap 边界，拒绝。

### 2. 输入使用宿主已准入的 snapshot DTO，不读取执行对象

**决策：** `action_capability_audit.v1` 只接收 bounded、可序列化的 snapshot：稳定 action identity、source、namespace/tool、version/digest、metadata、evidence refs、Run/Stream mode 和 optional expected outcome。审计不调用 Tool 方法、MCP server、Provider、registry、网络或文件系统。

**理由：** snapshot DTO 使 replay 可移植、确定性和隐私边界清晰，避免把动态副作用、执行对象和 provider/MCP 生命周期带入离线路径。它也复用归档 146 的 admitted snapshot 经验。

**替代方案：** 直接把 runtime Tool/Action interface 传入审计器：不可序列化、难以跨进程回放，且可能触发动态行为，拒绝。

### 3. 采用“声明—证据—结论”三层投影

**决策：** 归一化输出分为 declared facts、bounded evidence references、verdict/drift。Evidence 只保存引用、阶段、状态、作用域、版本、attempt/correlation 和有限摘要；不保存正文。`intent`、`issued`、`confirmed` 独立建模。

`confirmed` 只有在每条 issued evidence 均能匹配 action identity、version/digest、declared scope、attempt ID 与 correlation ID 时才作为确认事实。高风险 Preview/Approve/Commit/Verify reference 同样必须绑定 action identity、version 和 scope；approval/commit scope 不同单独分类为 scope drift。`observed.started` 与 issued evidence 必须一致。v1 输入上限固定为 1 MiB，最多 64 case/evidence/list item、256 字节字符串/摘要、24 小时 timeout 和 10 次 retry；超限拒绝，不截断。

**理由：** “模型想做什么”“请求是否发出”“外部系统是否确认”是不同事实。分层可以支持未知结果下的恢复决策，也避免将观察材料直接当作业务 Outcome。

**替代方案：** 只输出一个 `success/failed`：无法表达未确认副作用、部分证据和证据不足，拒绝。

### 4. 风险和幂等字段采用显式 unknown，不提供安全默认

**决策：** effect、side_effect、risk、reversible、idempotent、retryable、preconditions、timeout 和 verify evidence 缺失时输出稳定 gap/insufficient-evidence，不推断 safe、reversible 或 retryable。

**理由：** 默认“安全”会把元数据缺口转化为重复副作用风险；显式 unknown 与现有 fail-fast、Policy precedence 和 HITL 语义一致。

**替代方案：** 对只读 namespace 使用隐式默认安全：容易被错误注册或错误命名绕过，且无法作为跨来源合同，拒绝。

### 5. Verdict 与 drift taxonomy 固定且可回放

**决策：** 每个 action 只产生一个 `compliant|gap|insufficient_evidence|not_applicable` verdict；schema、privacy、metadata、evidence、approval-scope、declared-observed、Run/Stream parity、duplicate-conflict 和 library-first boundary 使用稳定分类码。

**理由：** 固定词表便于 shell/PowerShell gate、历史 fixture 和后续增量 change 复用；不把“无结论”伪装成通过。

**替代方案：** 允许任意自由文本原因：不可稳定比较，也无法阻断 taxonomy drift，拒绝。

### 6. Run/Stream 只比较规范化语义，不要求事件字节相同

**决策：** 若 fixture 同时包含 Run/Stream projection，则比较 action identity、metadata、issued/confirmed state、verdict 和 drift classification；事件时间和字节级顺序可不同，但语义差异必须分类。

**理由：** 与仓库既有 Run/Stream parity 口径一致，避免 transport timing 差异造成误报，也防止一条路径静默丢失确认事实。

### 7. 不新增 runtime 配置和诊断写入

**决策：** audit 使用 fixture 文件或测试输入，不添加 `runtime/config` 键，不通过 `RuntimeRecorder` 写入 RunRecord，不增加高基数 OTel 字段。若未来需要运行时字段，必须另起增量 change 并遵守 additive + nullable + default。

**理由：** 首阶段目标是证明 gap，不是建立新的事实存储或配置生命周期；回滚可以只删除审计包、fixture、gate 和文档。

## Risks / Trade-offs

- **[Risk]** snapshot 与真实执行事实脱节，审计结论过于乐观。→ **Mitigation:** 要求 source/observed 双投影、correlation/version/scope 校验；缺失或冲突统一返回 gap/insufficient_evidence，不输出 compliant。
- **[Risk]** 过严的边界和隐私限制使证据不足案例增多。→ **Mitigation:** 提供 `insufficient_evidence` 独立 verdict 和 bounded reference 字段，禁止通过截断 raw payload 来“制造完整证据”。
- **[Risk]** 新 drift code 与既有 tool lifecycle/security/timeline taxonomy 重复或冲突。→ **Mitigation:** 在设计阶段建立映射表；gate 同时校验 spec、代码常量、fixture 和文档词表，对既有分类只引用不重命名。
- **[Risk]** 审计逻辑演变成第二套 Action lifecycle。→ **Mitigation:** 只消费 source-owned lifecycle/timeline projection；禁止新增执行、重试、补偿、终态或持久队列状态机。
- **[Risk]** Run/Stream fixture 只覆盖单一 happy path。→ **Mitigation:** 强制覆盖 allow/deny/timeout、pre-execution rejection、started interruption、issued-unconfirmed、confirmed, duplicate/conflict 和 historical compatibility cases。
- **[Trade-off]** 只做离线审计无法立即阻止真实危险 Action。→ **Mitigation:** 明确本 change 的输出是证据与后续 change 触发信号；现有 Action Gate、Policy、Sandbox 和 HITL 继续作为实际强制边界。

## Migration Plan

1. 建立 `action_capability_audit.v1` schema、bounded limits、normalizer 和稳定分类码。
2. 增加 canonical fixture：完整合规、metadata 缺失、幂等冲突、Preview/Verify 缺失、issued-unconfirmed、approval scope drift、Run/Stream parity drift、隐私/越界和历史兼容。
3. 接入 diagnostics replay 的离线解析与 deterministic digest；新增 shell/PowerShell 等价 gate，禁止 provider/tool/network/RuntimeRecorder side effect。
4. 更新 mainline contract index、module boundaries、runtime diagnostics 文档和 roadmap；确认 Example Impact Assessment 为 `无需示例变更（附理由）`。
5. 验证通过后，若 fixture 仍证明某类 gap，另起最小 runtime change；若无稳定 gap，则只归档本审计，不引入运行时字段。

回滚：删除本 change 新增的 spec、审计 DTO/normalizer、fixture、replay 和 gate；保留既有 Action Gate、Tool lifecycle、Security、Timeline、Policy、Sandbox、Run/Stream 和 historical fixture 语义，无配置或持久化迁移。

## Open Questions

无。Action metadata 的字段边界、证据分层、verdict 和禁止的运行时扩展均已在本设计和 spec 中确定；后续只能在不改变这些合同的前提下选择具体 fixture 数量和 gate 接线位置。
