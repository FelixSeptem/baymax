# 运行生命周期

一次执行由 source owner 推进，宿主负责准入前的授权和业务 effect。`Run` 与 `Stream` 共用状态转移和终态决策；Stream 只改变交付形态。

```text
admit -> snapshot policy/budget -> assemble context
     -> model step -> tool/MCP attempt -> decide next
     -> pause/retry/continue OR finalize/abort
     -> source terminal evidence -> host-owned commit (if any)
```

1. **Admission**：解析宿主输入、capability、配置、authority 和预算；非法或 unsupported 输入在副作用前拒绝。
2. **Context projection**：按预算和策略组装只读上下文，保留 canonical facts；resolver、私有 memory 和业务授权仍由宿主拥有。
3. **Model projection**：`model/<provider>` 将 canonical request 投影为 provider-native request，并把协议错误归一到共享错误分类。
4. **Tool loop**：执行 local tool 或 MCP tool，记录 attempt/completion；action gate、timeout、retry 和 lease 由各自 owner 决定。
5. **Pause/recovery boundary**：`input_required`、approval、budget、checkpoint、history replay 和 continuation restore 是不同能力；history replay 只读，不能被文档描述成恢复执行。
6. **Event recording**：source facts 通过标准事件投影到 `RuntimeRecorder`；recorder 不重新执行、不拥有 terminal、不提交业务 effect。
7. **Terminal resolution**：source 产生完成、失败、取消、超时、DLQ 或 pause evidence；宿主再依据自己的 authorization/effect/commit contract 决定业务结果。

任何阶段失败都必须说明 owner、是否可重试、回滚点、typed error 和诊断字段。取消请求被接受不等于 source terminal cancellation；观察失败也不能改写 source outcome。

参考：[设计原则](design-principles.md)、[数据与事件流](data-event-flow.md)、[Runtime Harness 架构](../runtime-harness-architecture.md)、[主线契约测试索引](../mainline-contract-test-index.md)。

验证：`go test ./core/runner ./runtime/... ./integration -run 'Run|Stream|Terminal|Recovery' -count=1`。
