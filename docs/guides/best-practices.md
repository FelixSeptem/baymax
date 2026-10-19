# 最佳实践

## Host capability 与 mailbox plan bridge

调用 `orchestration/planbridge` 前先读取 versioned capability manifest；不要
用 package/DTO 存在性猜测 `mailbox_async`、history、checkpoint 或 pause 支持。
只有 manifest、mailbox contract version、当前 source availability 与完整
`HostPlanBinding` 都通过时，planbridge 才会调用既有 mailbox owner。

Mailbox message/delivery/lease/attempt/ack/nack/retry/DLQ 和 source terminal
由 mailbox owner 保持；plan digest、ack 或 DLQ 不是业务 authorization、effect
commit 或 durable resume。重复 dispatch 必须携带同一 authority、occurrence、
parent/causation、write scope 与 effect namespace；冲突 identity 应在 publish 前
fail closed。lease loss 不得通过重新 Run parent 恢复。

- 明确声明 Provider、model、tool、MCP endpoint 和 policy，不依赖自动发现。
- 把宿主 credential、超时、重试、限流和持久化策略留在宿主层。
- 对 Run 和 Stream 编写同一组业务终态断言。
- 使用结构化 tool schema 和 bounded 输入，避免把任意文本当作控制面。
- 诊断先脱敏再导出；fixture 不访问网络、不携带 secret。
- 配置变更先在文件或 fixture 中验证，再启用热更新；失败必须原子回滚。
- 新能力先写 OpenSpec、测试和文档影响评估，再改实现。
- 组件页面引用事实源和验证命令，避免 README 复制完整 contract。

不建议：隐式全局 registry、跨层导入 Provider SDK、旁路写诊断、为了通过门禁而放宽断言。

执行记录应消费真实 `types.EventHandler` 回调，用 [host/eventadapter](../../host/eventadapter/README.md) 显式绑定 source 和已授权 sink。不要把 RunResult 包装成额外 parent/terminal 事件；`input_required` 只表示暂停，未收到终态应记录 incomplete observation。业务 authorization、revision、effect key、receipt identity 和 commit gate 由宿主拥有。

直接事件的 EventID/TraceID 是 mapping reference，不是去重键；两条同 ID 回调都必须保留。并行 source 用独立 observer，不按时间戳生成全局 sequence。不合作 sink 会阻塞 source 回调；应响应独立 observer context 的 deadline，避免无界背压。sink error/timeout 可能发生在写入之后，必须作为 unconfirmed 处理，不能自动 retry。

快照接受有界 JSON-compatible 数据和标准 `time.Time`，拒绝自定义 JSON/Text marshaler。新增公开语义（包括 additive/opt-in）须同步其维护文档、spec、正负测试和门禁；运行 [专用 contract gate](../../scripts/check-native-execution-event-contract.ps1) 和离线示例后再交付。
# Host plan bridge boundaries

Treat `host_execution_plan.v1` as an opt-in compile-time consistency contract: always re-evaluate authorization, lease and remaining budget in `Binding.Admit` at activation. Do not treat a digest as authorization, a native terminal result as a business commit, or offline replay as Run resume. Keep callbacks/services in trusted bindings; their implementation cannot be authenticated by descriptor hashes. Roll back by ceasing bridge calls and using the native owner APIs; preserve task/checkpoint/outbox facts and reconcile effects through the host commit gate.

## Application outbox recovery（归档 166）

`establish-application-outbox-mailbox-recovery-contract` 已交付 bridge。接入时，应把 application outbox、业务 authorization、effect key/idempotency/CAS、cron/DST/misfire/backfill 和最终 outbox confirmation 保留给宿主；Baymax 仅复用既有 mailbox 或 scheduler owner。务必把 delivery acceptance、target lifecycle、host effect committed、outbox confirmed 分开记录和判断：mailbox ack、scheduler terminal、Run/Stream terminal、digest 或 replay 都不代表 business commit。

将完整 normalized identity（profile/version、target、outbox/occurrence、revision/effect reference、target binding、source correlation、idempotency key）用于 duplicate 判断；exact duplicate 才能收敛，改变 target/revision/effect binding 的 key reuse 必须拒绝。恢复采用 at-least-once，且不得把 mailbox delivery retry、scheduler execution retry 和 host business-commit retry 叠加。回滚停止新 bridge dispatch、保留事实并由宿主以 effect CAS 对账；不要删除 pending record 或宣称撤销已提交 effect。
