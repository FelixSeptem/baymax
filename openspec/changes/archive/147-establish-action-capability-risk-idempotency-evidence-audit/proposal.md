## Example Impact Assessment

无需示例变更（附理由）：本 change 只新增离线 action-capability descriptor、fixture、replay、contract 与 gate，不改变 `examples/agent-modes` 的 runtime path、expected markers、配置语义或用户可见行为。

## Why

Handbook 第 6、13、16、29、30 章反复指出：工具或 Action 被模型描述出来，不等于它已经被授权、已经发出或已经产生了预期结果；缺少副作用、风险、可逆性、幂等性、前置条件和验收证据声明时，重试、审批、补偿与完成判定只能依赖人工约定。Baymax 已有 Action Gate、工具生命周期、Policy/Sandbox、RuntimeRecorder 和 diagnostics replay owner，但尚无一个离线、确定性、可回放的合同，审计这些声明与有限执行证据是否一致。

现在适合先做审计而不是直接改变执行器：可以复用已准入 Tool/MCP/Action 的 bounded snapshot 和既有 lifecycle/timeline 事实，在不调用 provider、tool、网络或凭证的前提下，先证明缺口是否真实、哪些动作无法安全重试、哪些验收证据不足。只有审计 fixture 证明存在稳定 drift，后续 change 才能讨论最小运行时字段或策略行为。

## What Changes

- 新增版本化 `action_capability_audit.v1` 离线合同，描述已准入 Action 的稳定身份、版本/digest、owner、作用域、effect、副作用、风险等级、可逆性、幂等性、前置条件、超时/重试边界以及 Preview/Approve/Commit/Verify 能力；v1 输入上限为 1 MiB，引用字符串最多 256 字节，最多 64 个 case/evidence/list item，timeout 最长 24 小时，retry 最多 10 次。
- 新增受限的 evidence reference 投影，用于关联 Action intent、已发出事实、已确认结果、Policy/Sandbox/Tool lifecycle/timeline 证据；只保存 bounded reference、状态和摘要，不保存 raw payload、凭证、reasoning、完整命令输出或无界响应。
- 新增确定性的审计结论与 drift taxonomy，区分 `compliant`、`gap`、`insufficient_evidence` 和 `not_applicable`；缺失的幂等、风险、前置条件或 Verify 证据不得被默认为安全或成功。
- 新增离线 canonical fixture、replay normalizer、正向/负向/边界用例，覆盖副作用声明缺失、幂等冲突、Preview/Commit/Verify 缺口、已发出但未确认、重复执行、Run/Stream 证据不一致、隐私越界和历史字段缺失。
- 新增 shell/PowerShell 对等 contract gate，并把 replay 结果接入既有 diagnostics/contribution check 入口；审计输出不写 RuntimeRecorder、不修改 RuntimeRecord、不调用执行路径。
- 更新 action/tool/replay 相关文档与 roadmap 状态，明确该 change 的首阶段是 evidence audit，不引入凭证存储、Action registry、动态下载、全局路由、第二套终态状态机或自动补偿执行器。

## Capabilities

### New Capabilities

- `action-capability-risk-idempotency-evidence-audit`: 对已准入 Tool/MCP/Action 的风险、幂等、可逆性、前置条件、Preview/Approve/Commit/Verify 和 reference-only evidence 进行离线、确定性、可回放的一致性审计。

### Modified Capabilities

无。现有 `action-gate-hitl`、`action-gate-parameter-rules`、`tool-lifecycle-and-failure-isolation`、`tool-security-governance-s2`、`action-timeline-events` 和 `diagnostics-replay-tooling` 的运行时要求不在本 change 中改变；新能力只消费其 bounded、source-owned projection。

## Impact

- 代码/测试：预计新增 `tool/diagnosticsreplay` 或相邻离线审计包的 canonical DTO、normalizer、fixture/replay 测试，以及 `tool/contributioncheck` 的 contract gate；不修改 `core/runner`、`tool/local`、MCP dispatcher、Policy、Sandbox 或 Provider adapter 的执行语义。
- 规范/文档：新增 capability spec，更新 `docs/mainline-contract-test-index.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`（仅说明无配置/无诊断写入边界）和 `docs/development-roadmap.md`。
- 配置/API：不新增 runtime 配置键、热更新分支、公共执行 API 或凭证接口；所有新字段仅存在于版本化离线 fixture/审计输出，未知字段必须安全忽略，缺失字段按明确的 nullable/default 规则处理。
- 依赖/隐私：不新增 provider、网络、数据库、registry、marketplace 或外部服务依赖；审计输入只接受宿主已经准入的 bounded snapshot，禁止 raw payload、credentials、reasoning、完整命令输出和无界系统调用日志。
- 兼容/回滚：历史工具生命周期、Action Gate、Policy/Sandbox、Run/Stream 和 diagnostics fixture 保持可解析；回滚只移除新 capability、fixture、replay、gate 和文档，不需要迁移持久化数据或改变业务副作用。
- 验证：至少运行 `openspec validate --all`、受影响包正负/边界测试、diagnostics replay fixture、shell/PowerShell gate、`scripts/check-quality-gate.ps1`、`scripts/check-docs-consistency.ps1`，并在实现完成前执行仓库要求的 Go 全量、race 和 lint 门禁。
