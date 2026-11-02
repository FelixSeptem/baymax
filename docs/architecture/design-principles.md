# 设计原则

这些原则共同决定了 Baymax 的 API、失败分类、文档结构和测试门禁。遇到新能力时，先判断它属于哪个 owner，再判断是否能提供可复现 evidence；不要先从实现便利性推断公开契约。

- **Library-first**：宿主控制生命周期、凭据、网络、持久化、业务授权和最终 effect；Baymax 提供可嵌入的执行 primitive，不建设隐式控制面。
- **Contract-first**：行为、配置、诊断和 adapter contract 先进入 OpenSpec 与 spec delta，再进入代码、测试、replay 和 gate。
- **Source ownership**：每类 lifecycle fact、terminal outcome、lease、checkpoint 和 history 都由唯一 source owner 产生；adapter 只做受限投影或关联。
- **Additive evolution**：诊断和 QueryRuns 使用 `additive + nullable + default`，并保持旧消费者可解析；破坏性迁移必须单独记录。
- **Fail-fast and rollback**：非法配置、非法热更新、foreign/stale/expired/corrupt restore 和 unsupported capability 在副作用前拒绝；失败时保留上一个有效快照。
- **Deterministic evidence**：fixture、replay、contract test 和 gate 离线、bounded、可重复，不依赖网络探测或模型自述。
- **Run/Stream parity**：传输形态可以改变事件交付方式，但不改变工具决策、取消/超时、错误分类或 terminal semantics。
- **Explicit boundaries**：Provider SDK 只在 `model/<provider>`，MCP transport 只在 `mcp/*`，共享契约在 `core/types`，诊断写入只经过 `RuntimeRecorder`。

延伸阅读：[模块 ownership](module-ownership.md)、[运行生命周期](runtime-lifecycle.md)、[数据与事件流](data-event-flow.md)。
