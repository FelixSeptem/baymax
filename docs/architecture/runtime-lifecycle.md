# 运行生命周期

1. **Admission**：解析宿主输入、配置和预算；非法值在执行前拒绝。
2. **Context projection**：按预算和策略组装只读上下文，保留 canonical facts。
3. **Model projection**：Provider adapter 将 canonical request 投影为 native request。
4. **Tool loop**：执行 local tool 或 MCP tool，记录 attempt/completion 和错误隔离。
5. **Event recording**：所有诊断写入 `RuntimeRecorder`，Run/Stream 共享事件语义。
6. **Terminal resolution**：统一完成、失败、中断、超时和恢复终态，必要时生成 checkpoint/replay 输入。

任何阶段失败都必须说明 owner、可重试性、回滚点和诊断字段。禁止在 Stream 路径新增与 Run 不同的终止或决策语义。

参考：[Runtime Harness 架构](../runtime-harness-architecture.md)、[终态与恢复 contract](../mainline-contract-test-index.md)。

验证：`go test ./core/runner ./runtime/... ./integration -run 'Run|Stream|Terminal|Recovery' -count=1`。
