# 设计原则

- **Library-first**：宿主控制生命周期、凭据、网络和持久化，Baymax 提供可嵌入能力。
- **Contract-first**：行为、配置、诊断和 adapter contract 先进入 OpenSpec，再进入代码与测试。
- **Single source of truth**：每类事实只有一个 owner；概览文档通过链接引用。
- **Additive evolution**：诊断和 QueryRuns 采用 additive + nullable + default，避免破坏旧消费者。
- **Fail-fast rollback**：非法配置和热更新先校验，失败不改变当前有效快照。
- **Deterministic evidence**：replay、gate、fixture 离线、bounded、可重复，不依赖网络探测。
- **Run/Stream parity**：传输形态不改变决策、工具语义和终态。
- **Explicit boundaries**：Provider SDK、MCP transport、runtime 和 orchestration 保持分层。
