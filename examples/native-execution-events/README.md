# 原生执行事件接入

## Semantic anchors 与 runtime path

本示例保留两条独立路径：

- 实时路径：真实 `runner.Engine.Run/Stream` → `host/eventadapter.Observer` → 宿主内存 sink。fake model 使用 `adapter/modelcapability` 的显式适配，无网络、真实 tool effect 或业务数据库。`native_execution_event.v1` 保留 native type、source、Run、iteration、payload；callback `mapping_reference` 不能用于去重或 replay。
- history 路径：既有 `orchestration/mailbox.FileStore` → `Mailbox.HistorySource/Capabilities/ReadAfter`。示例先由 source 写入 admission/lease/ack 事实，再关闭写入阶段；新建 FileStore/reader 后只读已确认 cursor 之后的 retained 事实。runner/child/team/workflow 只有显式注入 owner journal 才启用相同 reader contract；journal 与 source-local sequence 归 owner 所有，eventadapter 不保存 history。

history anchor 是 `NativeExecutionEvent.History` 中的 source/event identity 与 source-local sequence；它独立于实时 callback EventID/TraceID 和协议 Sequence。cursor 是 opaque `mailbox.history.v1` source-instance position，不能合并成并行 child/team 的全局顺序。

source projection recovery 也必须由 source owner 显式注入：`history.Journal` 与
`history.ProjectionLedger` 同时提供时，owner 才能暴露 source commit、projection
acknowledgement、terminal evidence 和 `recovery-required` 状态。示例不会把 callback
EventID、host observation sequence 或 mailbox lifecycle history 当作投影事务；ledger
写入不确定时只读 recovery 必须停在 source cursor，不能重新 Run、retry、resume 或提交
业务 effect。

## Expected markers

运行 `go run ./examples/native-execution-events`，保留原有 `native-events run=completed`、`native-events stream=completed`、`observer=healthy` 和 `pause-is-terminal=false`，并输出：

- `mailbox-history durable=true`、`mailbox-history restart-events=2`、`mailbox-history terminal=acked`：新 reader 从已确认 admission cursor 读到 lease/ack，而非重新执行。
- `mailbox-history malformed=true`、`mailbox-history foreign=true`、`mailbox-history expired=true`：typed cursor 错误不静默回退 latest。fixture/gate 另外覆盖 retained gap 与 source unavailable。
- `native-history unsupported=unconfigured-runner,unconfigured-child,unconfigured-team,unconfigured-workflow`：其 callback API 存在不表示默认 owner 已启用 retained journal。
- `source-projection owner-injected-only`：source commit/projection recovery 只有在显式 owner ledger 绑定时成立；未绑定或 ledger 不可用保持 `unsupported`/`recovery-required`，不静默降级为 latest replay。

`input_required` 是暂停；confirm resolution 不表示 canceled。observer.Status 独立于业务 terminal，sink error 是 unconfirmed，必须由业务 commit gate 决定提交。mailbox `acked` 只证明 source lifecycle，不能证明业务 effect committed。

## 能力与回滚

direct callback 无 history/cursor/dedup/global sequence；Realtime 仅通过既有 binding 纯投影保留 source 能力。Mailbox FileStore 原子保存的 journal 支持重开读取，MemoryStore 只声明 process-local history。保留最后 1024 个 lifecycle events，无时间保证；floor 是最后已移除的 after-position，`nil` cursor 从当前 retained floor 开始，旧 cursor 返回 `ErrCursorExpired`。每页 limit 为 1–200；`Complete` 只表示当前 retained tail 已读完，不代表 source 永久终止。

history replay 只读，不调用 Publish/Consume/Ack/Nack/retry/resume/restore、模型、工具或 handler。写入阶段和 replay 阶段的代码显式分离；临时 mailbox 文件仅由示例创建并在退出时删除。无 Provider、网络或业务数据库。

默认交付 deadline 1 秒，sink 必须响应 context；默认引用 1024 bytes/16 citations/8 KiB，event 64 KiB。Rollback：停用 observer 恢复原 handler，或停止新的 history read；保留真实宿主 cursor/记录和 mailbox snapshot，不删除 source 状态，不以重新 Run 代替 replay。本示例不修改 agent-modes。
