## Why

Baymax 已具备 Realtime event protocol、durable runtime event-stream binding，以及统一 terminal outcome contract，但仍缺少一份跨 Run/Stream 路径的可恢复性合同：观察者断连、停止消费、取消或 Provider 流异常后，调用方必须能够补回已产生事实并取得唯一权威终态。当前缺口会让事件流、终态查询与 diagnostics replay 在边界场景下出现丢失、重复或语义漂移；现在收口可以直接复用已归档的 cursor/dedupe、durable binding 和 terminal outcome 基线。

## What Changes

- 新增 transport-neutral 的事件流终态恢复合同，覆盖 disconnect、consumer stop、catch-up/live-tail handoff、retention expiry、backpressure、cancel、timeout 和 Provider stream failure。
- 固化 source-owned cursor/replay 恢复流程：已产生事件可补收，catch-up 与 live tail 去重，恢复后不得重写首个业务终态。
- 固化事件流、权威终态、diagnostics query/replay 之间的关联与一致性要求，并保留 partial output 和已完成 tool-call facts。
- 增加 Run/Stream 对等的正向、负向、边界和故障注入 contract/replay/integration 覆盖，以及 shell/PowerShell parity gate。
- 为 retention、backpressure、断连恢复和终态查询增加 additive、nullable、defaultable 观测字段；不改变既有状态机、interrupt/resume 语义或 source-of-truth ownership。
- 明确失败与回滚边界：绑定校验或恢复协商失败不得修改 Runtime 状态；不得引入传输网关、外部事件存储、托管连接控制面或第二套终态状态机。

## Capabilities

### New Capabilities

- `runtime-event-stream-terminal-recovery`: 定义事件流断连/补收/live-tail 切换与权威终态恢复的一致性、幂等性、保留事实和回放门禁合同。

### Modified Capabilities

无。既有 durable binding、Realtime、Agent Runtime Protocol 和 diagnostics replay requirements 保持原文不变；本 change 通过新增组合能力与门禁验证它们在断连恢复场景下的联合语义。

## Impact

- 影响 `core/runner`、`core/types`、Realtime/durable binding 映射、`runtime/diagnostics`、`tool/diagnosticsreplay`、`observability/event` 和相关 integration/gate 脚本。
- 新增或扩展 `runtime-event-stream-terminal-recovery.v1` replay fixture、断连/补收/背压/终态一致性测试与 agent-mode 示例断言。
- 继续遵守 `env > file > default`、fail-fast + 原子回滚、`RuntimeRecorder` 单写入口、library-first 模块边界和 additive schema 兼容窗口。
- Example Impact Assessment：`修改示例`。扩展 `realtime-interrupt-resume`（或等价 agent-runtime 示例）的文档基线与运行断言，覆盖断连、cursor catch-up、live-tail 去重及终态查询；不新增传输网关示例。

## Example Impact Assessment

修改示例
