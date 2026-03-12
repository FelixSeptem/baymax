# Development Roadmap

更新时间：2026-03-12

## 目标

在当前 v1 基线能力上，进入可发布、可运营、可扩展的工程化阶段，重点提升：
- 稳定性（错误恢复、兼容性、回归防护）
- 可运维性（配置、观测、调试工具）
- 可扩展性（模型/工具/MCP/技能生态）

## 阶段规划

## Phase R1（2-4 周）稳定化与发布准备

### 进展（2026-03-11）
- [x] `upgrade-openai-native-stream-mapping`：完成 OpenAI 原生流式映射、fail-fast 终止语义、complete-tool-call-only 事件发射。
- [x] 增加 streaming golden tests 与回归用例（顺序、错误分类、Run/Stream 语义一致性）。
- [x] 引入 `golangci-lint` 配置与 CI 工作流（`go test` + `golangci-lint`）。
- [x] MCP HTTP/stdio 统一配置对象与默认值文档化（由 `harden-mcp-runtime-reliability-profiles` 完成）。

### 目标
- 冻结 v1 API 草案并补齐回归测试矩阵。
- 清理实现中的 compatibility-only 路径，降低行为歧义。

### 交付项
- 为 `model/openai` 补全原生流式映射（替换当前兼容实现）。
- 为 MCP HTTP/stdio 增加统一配置对象与文档化默认值。
- 增加 golden tests（事件序列、错误分类、tool feedback 合并）。
- 引入 lint + test + benchmark 的 CI。

### 验收标准
- `go test ./...` 稳定通过。
- 关键路径（run/tool/mcp/stream）覆盖率达到团队约定阈值。
- 事件/日志/trace 在一条运行内可 100% 关联。

## Phase R2（4-6 周）生产可运维能力

### 进展（2026-03-11）
- [x] `add-runtime-config-and-diagnostics-api-with-hot-reload`：完成 Viper 配置加载（YAML + Env + Default）、原子热更新、回滚语义与库级诊断 API。
- [x] `refactor-runtime-responsibility-boundaries-and-enrich-docs`：完成（配置/诊断 API 从 MCP 单体 runtime 包拆分到全局 runtime 模块，补齐迁移文档）。
- [x] `unify-diagnostics-contract-and-concurrency-baseline`：完成诊断 single-writer + idempotency、run/skill 契约加固、并发安全质量门禁收敛。

### 目标
- 支持线上部署场景下的调优与排障。
- 建立并发与异步执行机制的可调优与可观测闭环。

### 交付项
- 配置层（环境变量 + 文件）和热更新策略。
- 观测增强：采样率、日志级别、慢调用阈值告警字段。
- MCP 健康检查与自愈策略（指数退避、熔断窗口、重连上限）。
- 运行诊断 API（导出最近 N 次 run/MCP 调用摘要，库接口）。
- 并发调度策略（队列/背压/取消传播）与异步通讯机制收敛。
- 交付 R2 批次示例：`01-chat-minimal`、`02-tool-loop-basic`、`03-mcp-mixed-call`、`04-streaming-interrupt`（附 TODO 演进位）。

### 验收标准
- 在故障注入测试中，MCP 间歇性错误可自动恢复。
- 关键指标可通过单一 dashboard 观测（延迟、错误率、重试率）。

## Phase R3（6-8 周）生态扩展与开发者体验

### 进展（2026-03-12）
- [x] `bootstrap-multi-llm-providers-m1`：完成 `model/anthropic`、`model/gemini` 官方 SDK 最小非流式适配。
- [x] 新增跨 provider 契约测试（OpenAI/Anthropic/Gemini）最小成功路径与基础错误分类一致性。
- [x] `align-multi-provider-streaming-and-error-taxonomy-m2`：完成 Anthropic/Gemini streaming 接入、跨 provider 事件语义对齐与错误分类细化。
- [ ] M3 待办：能力探测与 provider 级降级策略（特性缺失时自动回退）。

### 目标
- 降低新接入成本，增强外部集成能力。
- 将模型层从“OpenAI 单 provider”演进为“多 provider 可插拔”能力。

### 交付项
- 模型适配接口文档与示例（多 provider）。
- 新增官方协议适配：
  - `model/anthropic`（Anthropic Messages API 语义映射）
  - `model/gemini`（Google Gemini API 语义映射）
- 对齐跨 provider 的统一语义：
  - 非流式 `Generate` 输出结构一致
  - 流式 `Stream` 事件语义一致（delta/tool_call/error/completed）
  - 错误分类对齐到 `types.ErrorClass`
- Tool SDK 指南（schema、错误语义、幂等建议）。
- Context Assembler（RAG + Memory）能力规划：
  - 新增 `context/assembler` 作为模型调用前的上下文组装阶段。
  - Provider 化接入 `session-memory`、`long-term-memory`、`rag-retrieval`（分阶段启用）。
  - 与 `runtime/config` 对齐可配置策略（enable、top_k、timeout、budget、fail-fast/best-effort）。
  - 与 `runtime/diagnostics` 对齐可观测字段（provider latency/hit/miss/truncate reason）。
- Skill 语义触发升级（可插拔检索/打分器）。
- Agent Action 输出体验（规划）：
  - 基于现有事件流构建用户侧 Action Timeline（run/model/tool/mcp 阶段）。
  - 增加统一动作状态语义（pending/running/succeeded/failed/skipped）。
  - 增加 step/phase 关联字段规范，支持前端稳定渲染执行路径。
- 提供最小 CLI 示例（本地调试和回放）。
- 交付 R3 高阶示例：`05-parallel-tools-fanout`、`06-async-job-progress`、`07-multi-agent-async-channel`。

### 验收标准
- 新工具接入时间显著缩短（按团队 KPI 评估）。
- 外部团队可根据文档独立完成接入。
- 至少 3 个 provider（OpenAI/Anthropic/Gemini）通过同一 runner 契约测试集。
- Context Assembler 在 `best-effort` 与 `fail-fast` 两种模式下通过契约测试，且不会破坏现有 runner/tool/skill 语义。
- Action Timeline 在 streaming 与 non-streaming 路径均可输出一致阶段视图。

## Multi-Provider 里程碑（规划，当前不实现）

- M1（R3 前半）：完成 `model/anthropic`、`model/gemini` 最小非流式适配与契约测试。
- M2（R3 后半）：完成流式事件映射与工具调用语义对齐，补齐回归测试。
- M3（R4 可选）：能力探测与 provider 级降级策略（例如特性缺失时自动回退）。

说明：截至 2026-03-12，仓库已实现 OpenAI/Anthropic/Gemini 的非流式 + 流式基础语义对齐；M3 仍为规划项。

## Phase R4（长期）平台化能力（非 v1）

### 方向
- 持久化恢复与跨会话编排
- 多租户与权限治理
- 审计与合规流水线
- 分布式执行与弹性调度
- Memory/RAG 平台化（向量索引生命周期管理、分层缓存、跨会话长期记忆治理）
- Human-in-the-loop 原生编排：
  - 在 agent loop 中支持 `await_user` / `resume` / `cancel_by_user` 生命周期。
  - 增加可配置的 Action Gate（高风险动作执行前确认）。
  - 保持 fail-fast 语义与审计可追溯性（确认人、确认时间、确认结果）。
- A2A 协议支持（Agent-to-Agent 协作互联）：
  - 引入 `a2a` 适配层用于跨 Agent 对等协作（任务生命周期、状态查询、异步回调/推送）。
  - 明确与 MCP 的互补边界：A2A 负责 Agent 间协作，MCP 负责工具集成。
  - 支持 Agent Card 能力发现与路由策略，纳入统一观测与错误分类体系。

## HITL 与 Action Timeline 里程碑（规划，当前不实现）

- H1（R3 前半）：先交付 Action Timeline 标准化与字段规范，不改 runner 主状态机。
- H2（R3 后半或 R4 前半）：引入 Action Gate（执行前确认钩子），支持外部编排式 HITL。
- H3（R4）：引入原生 pause/resume 语义（`run.awaiting_user` / `run.resumed`），完善契约测试与诊断记录。

## A2A 里程碑（规划，当前不实现）

- A1（R3 预研）：完成 A2A 协议语义映射设计（任务状态机、SSE/推送、能力发现），输出设计文档与边界说明。
- A2（R4 前半）：实现最小 A2A Client/Server 互联能力，支持基础任务提交、状态查询与结果回传。
- A3（R4 后半）：与现有 observability/diagnostics 集成，补齐跨协议契约测试（A2A + MCP 组合场景）。

参考资料：
- https://a2aprotocol.ai/blog/2025-full-guide-a2a-protocol-zh

## 技术债清单（当前建议优先）

- 清理仓库中的临时/备份产物与目录规范化（持续项）。
- 收敛 `mcp/http` 与 `mcp/stdio` 中重复的重试/事件逻辑到共享组件。
- 为 `skill/loader` 的语义匹配引入可测试的评分接口。
- 为 runner 添加更多压力测试（高并发工具调用 + 取消风暴场景）。

## 性能与并发安全基线

- 性能回归采用相对提升百分比规则，详见 `docs/performance-policy.md`。
- 并发安全为强制门禁：`go test -race ./...` + goroutine 泄漏检查。

## 发布节奏建议

- 每周：1 次内部预发布（含 benchmark 回归对比）
- 每双周：1 次稳定 tag（附变更日志与风险说明）
- 每月：1 次架构评审（评估是否进入下一 phase）
