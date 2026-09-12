## Why

Baymax 现在已经具备 source-owned active Run control、嵌入式宿主命令关联、Realtime interrupt/resume 和 Run/Stream 对等执行路径，但宿主仍不能在 Run 进行中提交有界、可关联且可回放的追加输入。现有 `RunRequest.Input`/`Messages` 只描述启动输入，Clarification response 也只是既有 HITL 恢复路径，无法表达影响当前 Run 下一步的 steering 与空闲后执行的 follow-up。

这是基础 Embedded Host 接缝完成后的下一项独立语义收口：在不向正在执行的 Provider 调用直接注入 prompt、不建立第二套 Session/Run 状态机或全局队列的前提下，定义运行中输入的 owner、admission、时序、安全边界、恢复和 Run/Stream parity。Why now：当前 host/runner 审计已经确认缺口位置和可复用 owner，且该方向已被明确选为下一项宿主能力；若继续由宿主自行拼接，将产生输入顺序、终态竞态和回放漂移。

## What Changes

- 新增 versioned、transport-neutral 的 steering/follow-up input envelope、admission response、source correlation、causation 和 bounded payload 规则。
- 区分 `steering` 与 `follow_up`：steering 只在 source-owned safe point 影响 active Run 的下一次决策；follow-up 只在当前 Run 进入约定的 idle/terminal 边界后排队执行。
- 在 `core/runner` 增加 source-owned、每 Run 有界的输入 ingress/queue 接缝；队列满、Run 未知、Session 不匹配、过期、重复、终态竞态和取消竞态必须 fail-fast 且返回确定性 admission outcome。
- 将 host command correlation 接入输入 admission，但不让 host adapter 拥有输入队列、Run 状态、Session history 或终态仲裁。
- 明确模型调用、tool execution、HITL 等待、Realtime interrupt/resume、cancel 和 retry 与 steering/follow-up 的优先级及生效边界；禁止任意 prompt 注入当前 Provider 调用。
- 定义 disconnect、reconnect、duplicate replay 和 pending input 的处理；默认不把未消费输入写入第二套持久化存储，恢复只复用现有 source-owned checkpoint/history/snapshot 能力。
- 为 Run/Stream 增加等价的 admission、safe-point、queue pressure、terminal race、cancel/retry/resume 和 replay contract 覆盖。
- 增加 versioned transcript/replay fixture、并发/race 测试、backpressure gate、host subprocess 场景以及 shell/PowerShell parity 检查。
- 不新增远程 gateway、hosted Session/Artifact store、RBAC、多租户、provider-specific steering API、全局输入队列或新的 compaction/history owner。

## Example Impact Assessment

修改示例

先更新 `examples/agent-modes/MATRIX.md` 与对应 `realtime-interrupt-resume` README 的 semantic anchor、runtime path、expected markers 和 rollback notes，再在既有 minimal/production-ish 变体中增加 steering admission、safe-point 生效、follow-up idle 边界、重复/过期输入、断连和 Run/Stream parity 场景；不新增平行 numbered example。

## Capabilities

### New Capabilities

- `runtime-steering-and-follow-up-input-contract`: 定义运行中 steering/follow-up 输入 envelope、source-owned admission、每 Run 有界输入队列、安全生效边界、终态/取消竞态、断连/replay 和 Run/Stream parity。

### Modified Capabilities

- `embedded-host-command-response-and-event-correlation-contract`: 扩展首 profile command vocabulary 和 correlated command response，使宿主可以提交 steering/follow-up，同时保持 command admission 与业务终态分离、pending 收口和 source ownership。
- `agent-runtime-protocol-contract`: 增加运行中输入的稳定 correlation、source admission 和 lifecycle 投影约束；不改变既有 Run 状态 taxonomy、终态 owner、cancel/retry 因果语义或 Session context 的 reference-only 边界。

## Impact

- `core/types`：新增 additive 输入 envelope、输入类型、admission status/reason、safe-point/queue outcome 和 replay DTO；所有可选字段保持 nullable/defaultable 并绑定 profile version。
- `core/runner`：实现 source-owned active Run input ingress、有限队列、safe-point 应用、Run/Stream 对等路径和终态竞态保护；不把输入队列放入 host 或 diagnostics。
- `host` / `host/jsonl`：增加输入命令校验、correlation、响应和断连收口；继续保持 stdout protocol-only、bounded delivery 和无业务状态 ownership。
- `observability/event` / diagnostics：通过 `RuntimeRecorder` 增加 bounded input admission、queue pressure、apply/reject、duplicate/stale 和 terminal-race facts；不记录原始 prompt 内容为高基数字段。
- `tool/diagnosticsreplay`、contract fixtures、subprocess harness、shell/PowerShell gates：新增 steering/follow-up replay、parity、race、backpressure 和 forbidden-boundary 覆盖。
- `examples/agent-modes`、README、runtime module boundaries、runtime config/diagnostics、mainline contract index 和 roadmap：同步能力边界、示例 marker、验证命令、回滚点与不吸收项。
- 不新增配置优先级、Provider SDK、远程服务或持久化依赖；非法输入和非法热更新（如后续引入配置）必须 fail-fast 并保持原子回滚。
