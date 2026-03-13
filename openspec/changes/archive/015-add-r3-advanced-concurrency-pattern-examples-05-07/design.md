## Context

当前示例仅覆盖 R2 基线（01-04），而 roadmap 与 examples 规划已进入 R3 高阶模式（Parallel / Async / Multi-Agent）。项目定位是 library-first，因此示例扩容应优先承担“模式演示与行为观察”职责，而不是引入新的核心运行时能力。用户已确认本提案收敛范围：最小演示、结构化 event 输出、接入 runtime manager、不改核心行为。

## Goals / Non-Goals

**Goals:**
- 提供 4 个可运行高阶示例（05-08），覆盖并发工具扇出、异步进度、进程内多 agent、网络通信桥接。
- 所有示例统一接入 runtime manager，并输出结构化 event 便于观察执行路径。
- 在 README 与 docs 中建立一致的 Pattern 导航与示例映射，消除文档漂移。
- 保持现有 runner/runtime/model 契约不变，避免额外回归风险。

**Non-Goals:**
- 不实现 Teams/Workflow/Control Plane 的生产能力。
- 不新增示例级集成测试矩阵（本期仅可编译可运行）。
- 不引入新的外部基础设施依赖（如数据库、消息队列、向量库）。

## Decisions

### Decision 1: 07 拆分为 channel 与 network 两个独立示例
- Choice: 保留 `07-multi-agent-async-channel`（单进程 channel）并新增 `08-multi-agent-network-bridge`（最小网络通信）。
- Rationale: 两种通信模型关注点不同，拆分后更利于理解、演进与排障。
- Alternative: 将两者放入同一示例。Rejected：复杂度过高，不利于“最小演示”目标。

### Decision 1.1: 08 网络示例采用 JSON-RPC 2.0 协议语义
- Choice: `08-multi-agent-network-bridge` 使用 JSON-RPC 2.0 进行 agent 间请求/响应通信（参考 MCP 协议的 JSON-RPC 语义）。
- Rationale: 与项目现有 MCP 生态一致，便于用户理解“工具协议与 agent 协作协议”之间的映射关系。
- Alternative: 自定义 JSON 消息协议。Rejected：重复造轮子且不利于后续与 MCP/A2A 的认知衔接。

### Decision 1.2: 08 网络示例固定使用 HTTP 作为传输层
- Choice: `08-multi-agent-network-bridge` 使用 HTTP 承载 JSON-RPC 2.0 消息。
- Rationale: 最小落地成本、调试友好，且与 MCP HTTP 适配路径认知一致。
- Alternative: TCP 原生 socket。Rejected：超出本期“最小演示”目标并增加额外协议处理复杂度。

### Decision 2: 所有高阶示例统一接入 runtime manager
- Choice: 在每个示例使用 `runtime/config.Manager` 作为标准入口，读取默认或最小配置并记录 diagnostics。
- Rationale: 与项目主路径一致，示例即最佳实践。
- Alternative: 示例直接硬编码配置。Rejected：会放大与生产路径的差异。

### Decision 3: 结构化 event 作为异步示例的基础输出
- Choice: 06/07/08 统一输出结构化事件（JSON logger）并包含 run/iteration/call 等关联字段。
- Rationale: 并发与异步行为需要可观测证据，纯文本日志不足以复用。
- Alternative: 仅终端文本输出。Rejected：不利于后续前端 Timeline 与诊断复用。

### Decision 4: 提案范围限制在 examples + docs
- Choice: 不改 core/runtime/model 逻辑，仅在示例层组合现有能力。
- Rationale: 降低回归风险，快速交付可见价值。
- Alternative: 顺带调整 runner 状态机或事件契约。Rejected：超出本提案边界。

## Risks / Trade-offs

- [Risk] 示例最小化可能覆盖不足。 -> Mitigation: 每个示例配套 `TODO.md` 明确后续扩展点。
- [Risk] 网络示例在不同环境可用性波动。 -> Mitigation: 提供本地 loopback 默认配置与失败提示。
- [Risk] 文档更新滞后导致再次漂移。 -> Mitigation: 本提案将 README + docs 作为同批次强制交付项。

## Migration Plan

1. 新增 examples 05-08 的目录与最小实现。
2. 接入 runtime manager 与结构化 event 输出。
3. 更新 README 的 Pattern 导航索引表与示例说明。
4. 更新 examples 扩容计划与 roadmap 进展。
5. 更新 v1 acceptance 的示例验收项。

Rollback strategy:
- 若某个新示例在特定环境不稳定，可单独回退该示例目录，不影响核心包与现有示例。
- 文档可同步回滚到上一个已归档状态，保持实现与文档一致。

## Open Questions

- 是否在下一阶段为 05-08 补充轻量 smoke tests（当前非本提案范围）。
