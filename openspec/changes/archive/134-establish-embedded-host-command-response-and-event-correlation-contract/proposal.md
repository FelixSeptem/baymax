## Why

Baymax 已能把 Run、Realtime、durable event stream 和终态投影给嵌入式宿主，但宿主仍缺少一个可在 Run 进行中关联命令响应、异步事件和反向 HITL 请求的标准接缝。现有协议虽声明 `cancel|resume|retry` action，却主要验证可用性而不连接 source Runtime；Realtime 事件也主要在 Run/Stream 入口摄入，导致 IDE、桌面宿主或 headless UI 必须自行拼接生命周期、取消函数和 resolver，容易产生重复终态、悬挂请求和输出通道污染。

这是 Roadmap 当前首选 P2，也是 Pi 对照研究后最明确的有界增量：在不建设远程网关或第二套状态机的前提下，把既有 owner 连接成 transport-neutral 的宿主合同，并提供首个严格 JSONL/stdio binding。

## What Changes

- 新增 transport-neutral 的宿主命令、命令受理响应、异步 Runtime 事件和反向 HITL 请求/响应 envelope，并以稳定 ID、`run_id`、causation 和 source correlation 关联。
- 增加 source-owned active Run control handle/registry 接缝，使 `cancel` 和运行中 `interrupt/resume` 能到达权威 Runner；协议适配层只路由和校验，不拥有 Run 状态。
- 明确命令“已受理/被拒绝”与命令最终导致的业务终态分离；late/duplicate 命令、重复 HITL 响应和终态冲突必须幂等且不得覆盖首个权威终态。
- 将现有 `ClarificationResolver` 与 `ActionGateResolver` 适配为可选的 host-mediated 反向请求，保留既有 RequestID、timeout、deny/cancel 和 Run/Stream 等价语义。
- 提供首个 strict JSONL over stdin/stdout binding：LF-only framing、版本协商、最大帧限制、malformed frame fail-fast、stdout 协议纯净、stderr 日志隔离和显式输出 backpressure。
- 复用 durable stream 的 cursor/catch-up/live-tail、terminal recovery、readiness admission、policy/sandbox、failure taxonomy 和 `RuntimeRecorder`，不创建新的 cursor、event store、终态仲裁或诊断写入口。
- 增加正向、负向、边界、disconnect/timeout、Run/Stream parity、subprocess、replay 和 shell/PowerShell parity gate 覆盖。
- 明确不包含 steering/follow-up 队列、REST/SSE/WebSocket/gRPC gateway、remote Session/Artifact store、托管连接、RBAC、多租户、CBOR、attachment/lease 或平台化 UI。

## Example Impact Assessment

修改示例

先更新 `examples/agent-modes/MATRIX.md` 与 `examples/agent-modes/realtime-interrupt-resume/README.md` 的文档基线，再为既有 realtime 模式增加宿主命令关联、取消、HITL reverse request、断连恢复和 stdout 纯净的可回归场景；不新增平行 numbered example。

## Capabilities

### New Capabilities

- `embedded-host-command-response-and-event-correlation-contract`: 定义 transport-neutral 宿主 envelope、request correlation、命令受理与异步结果分离、HITL reverse request、pending 收口，以及 strict JSONL/stdio binding 的 framing、输出完整性和 backpressure 合同。

### Modified Capabilities

- `agent-runtime-protocol-contract`: 将 host-visible `cancel|resume|retry` 从纯可用性声明扩展为可连接 source-owned Run control 的执行接缝，同时保持动作可用性与授权分离、单一终态和 source Runtime 所有权。
- `realtime-event-protocol-and-interrupt-resume-contract`: 增加 active Run 期间的 source-owned realtime command ingress，使 interrupt/resume 继续复用现有 envelope、sequence、dedupe、cursor 和 Run/Stream 等价语义。

## Impact

- `core/types`: additive host envelope、control handle、command result、reverse request 和 binding DTO/interfaces；兼容字段保持 nullable/defaultable。
- `core/runner`: active Run 的有界注册、幂等控制与 Run/Stream 对等接入；不把诊断或传输状态放入 Engine。
- `orchestration/composer`: 组合 Runner control、readiness、policy/sandbox、durable binding、terminal recovery 和宿主 adapter，不成为新事实源。
- `observability/event`、`runtime/diagnostics` 与 tracing：只增加关联和收口事实，诊断写入继续经过 `RuntimeRecorder`。
- 新增本地宿主 adapter 与 strict JSONL/stdio binding；`runtime/*` 不依赖 `mcp/http`、`mcp/stdio` 或任何网络 listener。
- `examples/agent-modes/realtime-interrupt-resume`：按文档先行规则修改现有示例和 marker。
- Contract/replay/gate：新增宿主协议 fixture、subprocess conformance、framing drift、pending cleanup、终态冲突、Run/Stream parity 和 shell/PowerShell 对等检查。
- 不新增外部服务或运行时依赖，不修改配置优先级，不引入 Provider SDK、远程持久化或第二套 Session/Run 状态机。
