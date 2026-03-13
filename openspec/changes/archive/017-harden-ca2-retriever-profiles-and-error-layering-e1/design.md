## Context

当前 CA2 Stage2 external retriever 已通过统一 SPI + HTTP adapter 支持 `file/http/rag/db/elasticsearch`，并将核心结果写入 diagnostics。但在工程实践中仍有三个问题：

1. 接入成本：每次接入外部检索服务都要重复配置 request/response mapping 与鉴权字段，容易出错。
2. 排障成本：`stage2_reason` 语义较粗，无法稳定区分 transport/protocol/semantic 失败域。
3. 治理成本：缺少统一可扩展 profile 与预检查入口，不利于上线前质量把关与后续演进决策。

本提案定位 E1 收敛，不引入供应商绑定 SDK，不触及 agentic routing 与 runner 主状态机。

## Goals / Non-Goals

**Goals:**
- 引入 CA2 Stage2 external retriever profile 模板机制，降低接入配置复杂度。
- 在 Stage2 retrieval 路径提供稳定的错误分层与 reason code 输出。
- 扩展 diagnostics 字段（`stage2_reason_code`、`stage2_error_layer`、`stage2_profile`）并保持向后兼容。
- 提供 external retriever 配置预检查库接口，定义 warning/error 的执行策略边界。
- 保持现有 `env > file > default` 与 `fail_fast/best_effort` 语义不变。

**Non-Goals:**
- 不引入 GraphRAG/RAGFlow/Elasticsearch 等供应商专用 SDK 适配器。
- 不实现 CA2 agentic routing。
- 不修改 runner 主状态机与 tool-call 事件契约。
- 不新增 CLI 诊断命令。

## Decisions

### Decision 1: profile 模板采用“默认值 + 显式覆盖”模型
- Choice: `external.profile` 决定一组默认 mapping/auth/header 建议值，用户显式配置字段可覆盖模板默认值。
- Rationale: 既降低接入门槛，又保留灵活性与向后兼容。
- Alternative: profile 完全锁定配置。Rejected：会牺牲非标准服务接入能力。

### Decision 2: 错误分层先落最小 taxonomy
- Choice: 统一三层错误域：`transport`、`protocol`、`semantic`，并输出 reason code。
- Rationale: 可显著提升排障效率，且不会过早绑定厂商语义。
- Alternative: 继续仅输出自由文本 `stage2_reason`。Rejected：可观测性与自动化治理不足。

### Decision 3: 预检查接口保持库优先，不引入 CLI
- Choice: 新增 runtime 级预检查 API，返回 warning/error 列表；warning 可继续，error fail-fast。
- Rationale: 与项目 library-first 定位一致，便于集成到现有启动/热更新路径。
- Alternative: 仅依赖运行时报错。Rejected：上线风险高，反馈滞后。

### Decision 4: 不改变现有 stage policy 行为
- Choice: 维持 `fail_fast` 立即终止、`best_effort` 降级继续；错误分层仅增强诊断语义。
- Rationale: 降低行为回归风险，保持已验收契约稳定。
- Alternative: 按错误层级引入新重试/降级策略。Rejected：扩大语义面与测试面。

## Risks / Trade-offs

- [Risk] profile 模板可能被误解为“供应商绑定协议”。 -> Mitigation: 明确 profile 为可覆盖默认集，保留 mapping 自定义路径。
- [Risk] 错误分层边界不清导致 reason code 漂移。 -> Mitigation: 定义固定映射规则与契约测试，避免自由文本扩散。
- [Risk] 预检查 warning 过多降低信号质量。 -> Mitigation: 仅保留高价值 warning 类型，并提供稳定分类字段。

## Migration Plan

1. 扩展 runtime 配置模型并加入 profile 默认表与归一化逻辑。
2. 在 provider/assembler 路径接入错误分层与 reason code 输出。
3. 扩展 diagnostics 存储与 recorder 字段映射。
4. 新增预检查 API，并复用启动/热更新校验逻辑。
5. 补齐单测/回归/集成测试，验证 fail_fast/best_effort 不回归。
6. 同步 README/docs（含 v1-acceptance）并执行文档一致性检查。

Rollback strategy:
- 配置层可将 `external.profile` 退回 `http_generic` 或显式 mapping 配置。
- 如新增字段影响消费者，可继续仅消费旧字段（兼容保留）。
- 如错误分层引发异常，可临时回退到现有 `stage2_reason` 兜底行为。

## Open Questions

- 是否需要在本期同时输出 `stage2_reason_message`（原始可读信息）作为可选字段。
- profile 模板在 docs 中是否需要附最小样例片段（建议至少提供一组 ragflow_like 示例）。
