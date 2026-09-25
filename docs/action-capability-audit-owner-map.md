# Action Capability Audit Owner Map

本文件固定 `action_capability_audit.v1` 的 source-of-truth 边界。审计层只消费这些 owner 已经准入、归一化的 bounded snapshot；不复制运行时状态，不执行 Action，也不写入诊断事实。

| 事实域 | 唯一 owner | 审计层可消费的 bounded projection | 审计层禁止承担的职责 |
| --- | --- | --- | --- |
| Action 授权与人工审批 | `action-gate-hitl`、`action-gate-parameter-rules` | approval decision、scope、identity、version、parameter-rule reference | 重新审批、修改授权、执行 commit |
| Tool/MCP 生命周期 | `tool-lifecycle-and-failure-isolation` | admission、attempt、started/finished、failure class、idempotent finalize reference | 启动 Tool、重试、补偿、维护第二套生命周期 |
| 安全与边界 | `tool-security-governance-s2`、Policy/Sandbox owner | policy/sandbox decision、scope、egress reference、bounded violation code | 发现能力、绕过 policy/sandbox、执行网络或文件系统操作 |
| Action 时间线 | `action-timeline-events` | action identity、phase、status、correlation、sequence reference | 追加事件、改变顺序、拥有终态 |
| 诊断单写入口 | `observability/event.RuntimeRecorder` | 仅允许 reference-only 输入；本审计不写入 | 直接写 RuntimeRecorder、RunRecord 或 OTel 高基数字段 |
| 离线回放 | `tool/diagnosticsreplay` | canonical normalization、fixture replay、deterministic digest、verdict | 调用 provider/tool/MCP/registry/网络/凭证/时钟 |
| 贡献与门禁 | `tool/contributioncheck`、shell/PowerShell gates | taxonomy、fixture、privacy、library-first boundary 校验 | 成为 runtime policy、registry 或业务 Outcome owner |
| Run/Stream 语义 | 既有 Run/Stream source owners | normalized action/evidence parity projection | 创建平行终止、决策或事件状态机 |

## Boundary rules

- 输入版本固定为 `action_capability_audit.v1`，只接受宿主或既有准入 owner 提供的 bounded admitted snapshot。
- v1 JSON 最多 1 MiB；case/evidence/list 各最多 64 项；string/reference/summary 最多 256 字节；timeout 不超过 24 小时，retry 不超过 10；严格拒绝重复键、未知字段、payload-like 字段与敏感内容，不截断后继续。
- 缺失风险、幂等、可逆性、前置条件或 Verify 证据只能产生 `gap` 或 `insufficient_evidence`，不得推断安全、成功或可重试。
- `intent`、`issued`、`confirmed` 是不同事实；任何未匹配 action/version/scope/attempt/correlation 的 issued→confirmed 关系均不得算确认，read-only issued-unconfirmed 也不得判为 compliant。审计输出只保留 reference、状态、scope、version、attempt/correlation 和 bounded summary。
- 禁止 raw payload、凭证、reasoning、完整命令输出、无界响应、动态 discovery、credential store、global queue、hosted session store 和自动补偿执行器。
- 若未来需要运行时字段或执行策略，必须另起 OpenSpec change，并重新确认 additive + nullable + default 与 Run/Stream 对等约束。

## Example Impact Assessment

无需示例变更（附理由）：本文件只记录 owner 边界，不改变 `examples/agent-modes` 的 runtime path、配置、expected markers 或 rollback notes。
