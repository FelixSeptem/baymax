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

## Security Track（R2-R3，交叉推进）

### 进展（2026-03-12）
- [x] `harden-security-baseline-s1-govulncheck-and-redaction`：完成 `govulncheck` strict 质量门禁接入（Linux/PowerShell/CI 一致语义）。
- [x] 完成统一脱敏管线落地（diagnostics/event/context assembler），并补齐关键词扩展口与回归测试。

### 目标
- 建立可落地的安全基线，不破坏现有 library-first 主路径。

### 交付项
- S1（R2）：将依赖/静态安全扫描纳入 CI（优先 `govulncheck`，再评估 `gosec` 规则集）。
- S1（R2）：补齐敏感信息脱敏策略（日志/诊断/错误消息）并与现有 runtime diagnostics 对齐。
- S2（R3）：工具调用安全收敛（参数校验强化、权限策略、频率限制）。
- S2（R3）：模型输入输出安全过滤接口（PII/注入防护）先提供扩展点，再逐步默认启用。
- S3（R3+）：安全事件分类与告警字段规范，接入统一观测。

### 验收标准
- CI 包含安全扫描门禁且可稳定运行。
- 诊断与日志中的敏感字段可控脱敏，无明文泄漏回归。
- 高风险工具调用具备可审计的权限与限流策略。

## Phase R3（6-8 周）生态扩展与开发者体验

### 进展（2026-03-12）
- [x] `bootstrap-multi-llm-providers-m1`：完成 `model/anthropic`、`model/gemini` 官方 SDK 最小非流式适配。
- [x] 新增跨 provider 契约测试（OpenAI/Anthropic/Gemini）最小成功路径与基础错误分类一致性。
- [x] `align-multi-provider-streaming-and-error-taxonomy-m2`：完成 Anthropic/Gemini streaming 接入、跨 provider 事件语义对齐与错误分类细化。
- [x] `add-provider-capability-detection-and-fallback-m3`：完成基于官方 SDK 的动态能力探测、model-step preflight、provider 级有序降级与 fail-fast 终止。
- [x] `build-context-assembler-ca1-prefix-append-only-baseline`：完成 pre-model hook、immutable prefix hash、一致性 fail-fast、append-only JSONL journal 与 CA1 最小诊断字段。
- [x] `implement-context-assembler-ca2-lazy-stage-routing-and-tail-recap`：完成 CA2 双阶段路由、file provider、tail recap 与 CA2 诊断字段。
- [x] `add-r3-advanced-concurrency-pattern-examples-05-07`：完成 R3 高阶示例扩容（05/06/07/08），并为异步与多代理示例补齐结构化事件输出与 runtime manager 接入。

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
- Context Assembler（RAG + Memory）分期实施：
  - CA1（基础骨架，已完成）：新增 `context/assembler` pre-model hook，建立 immutable prefix 与 append-only journal 基线。
  - CA2（按需加载，已完成基础版）：接入 Stage1/Stage2 路由、可配置 stage 策略、tail recap；rag/db provider 保持接口占位。
  - CA3（压力控制）：落地 Goldilocks Zone（40%-70%）与 batch squash/prune，支持 spill/swap 回填。
  - CA4（生产收敛）：补齐规则防护、可中断恢复、观测面板与契约测试闭环。
  - 详细分期见 `docs/context-assembler-phased-plan.md`。
- Skill 语义触发升级（可插拔检索/打分器）。
- Agent Action 输出体验（规划）：
  - 基于现有事件流构建用户侧 Action Timeline（run/model/tool/mcp 阶段）。
  - 增加统一动作状态语义（pending/running/succeeded/failed/skipped）。
  - 增加 step/phase 关联字段规范，支持前端稳定渲染执行路径。
- 提供最小 CLI 示例（本地调试和回放）。
- 交付 R3 高阶示例：`05-parallel-tools-fanout`、`06-async-job-progress`、`07-multi-agent-async-channel`、`08-multi-agent-network-bridge`。
  - TODO：结合 CA2 增加 staged context 路由示例（本提案不新增 example 代码）。
- Knowledge 基础能力（R3，先接口后实现）：
  - 向量检索 provider 抽象（优先定义 `rag/db` 接口与错误语义，不在 R3 绑定具体供应商）
  - 文档解析与分片策略接口（parser/chunker contract），与 Context Assembler 对齐
  - 检索结果与 CA2 Stage2 集成约定（保持 fail-fast/best-effort 语义一致）
- Guardrails 基础能力（R3，扩展点优先）：
  - 输入验证扩展口（schema/regex/custom）
  - 输出过滤扩展口（PII/格式约束）
  - 规则优先级与冲突处理策略（最小可配置）
  - 违规记录进入现有 diagnostics/event，避免旁路链路

### 验收标准
- 新工具接入时间显著缩短（按团队 KPI 评估）。
- 外部团队可根据文档独立完成接入。
- 至少 3 个 provider（OpenAI/Anthropic/Gemini）通过同一 runner 契约测试集。
- Context Assembler 在 `best-effort` 与 `fail-fast` 两种模式下通过契约测试，且不会破坏现有 runner/tool/skill 语义。
- Action Timeline 在 streaming 与 non-streaming 路径均可输出一致阶段视图。

## DX Track（R3-R4，交叉推进）

### 目标
- 提升开发与调试效率，同时保持“库接口优先、CLI 可选”的项目定位。

### 交付项
- D1（R3）：补齐 API 参考与 godoc 示例覆盖（优先核心包：`core/*`、`runtime/*`、`context/*`）。
- D1（R3）：增强诊断调试体验（事件视图/回放接口），优先库接口，不强依赖 CLI。
- D2（R3-R4）：可选本地调试 CLI（run 回放、配置验证、事件查看）作为辅助工具，不改变 runtime API 主路径。
- D2（R4）：评估可视化调试/playground 与代码生成工具价值，按真实接入痛点推进。

### 验收标准
- 文档可支持外部团队按 README/docs 独立接入核心能力。
- 关键调试链路可通过库接口完成定位，CLI/可视化仅作为增益项。

## Multi-Provider 里程碑（规划，当前不实现）

- M1（R3 前半）：完成 `model/anthropic`、`model/gemini` 最小非流式适配与契约测试。
- M2（R3 后半）：完成流式事件映射与工具调用语义对齐，补齐回归测试。
- M3（R3 已完成）：能力探测与 provider 级降级策略（特性缺失时自动回退）。

说明：截至 2026-03-12，仓库已实现 OpenAI/Anthropic/Gemini 的非流式 + 流式语义对齐，以及官方 SDK 动态能力探测与 provider 级降级策略。

## Phase R4（长期）平台化能力（非 v1）

### 目标
- 保持 library-first 核心，同时提供可选的服务化部署路径
- 在不破坏当前 runtime 契约的前提下，逐步引入多 Agent 协作与平台化能力

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
- Teams（多 Agent 协作）：
  - 角色模型与协作策略（Leader/Worker/Coordinator；串行/并行/投票）
  - 任务分发、结果聚合、冲突处理与状态同步
- Workflows（工作流编排）：
  - 工作流 DSL（YAML/JSON）与确定性执行语义
  - 条件分支、循环控制、依赖调度与恢复点
  - 与 agentic 步骤混合编排
- Control Plane（可选服务化层）：
  - Web 控制台（任务监控、配置管理、性能分析）
  - Session 生命周期管理（创建、恢复、终止）
  - 权限、审计与运营指标可视化
- Memory 增强：
  - 短期/长期记忆分层管理
  - 检索排序与遗忘策略（LRU/时间/重要性）
- 生态集成扩展：
  - 数据库、消息队列、云服务与企业应用适配按需求增量推进
  - 不设固定数量目标，以维护成本与实际接入价值为准
- 生态与插件化能力（R4）：
  - 企业系统适配器（数据库、消息队列、API 网关）与云服务 SDK 集成收敛。
  - 插件系统架构（生命周期、版本管理、审核流程）先设计后实现。
  - 标准化工具定义与发布规范，支持社区贡献治理。
- 智能运营能力（R4）：
  - 自动性能调优与智能降级建议（先建议模式，后自动执行）。
  - 运营仪表板与异常检测/自愈策略。
  - 成本优化策略（模型选择、缓存命中、资源调度）。

## Context Assembler 里程碑（规划，按 P1-P10 原则分期）

- CA1（R3 前半，已完成）：Prefix + Append-only 基线
  - 对齐 P1/P2/P6/P9/P10（前缀一致性、只追加、不信任 LLM、可观测、文件即记忆）。
- CA2（R3 后半，已完成基础版）：Lazy + Stage 化加载
  - 对齐 P3/P4/P5（按需加载、渐进降级、末尾复述）。
- CA3（R4 前半）：内存压力与可恢复性
  - 对齐 P7/P8 + Arena 机制（可中断、不丢信息、batch reset、spill/swap）。
- CA4（R4 后半）：生产级策略收敛
  - 完成策略参数化、契约测试、压测阈值与文档统一。

说明：Context Assembler 不建议单次大改完成，需按分期逐步启用，保证 runner 语义稳定与并发安全基线不回退。

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
- 统一错误分类与错误处理策略（细化 error taxonomy 与跨模块映射）。
- API 版本控制与兼容策略文档化（在对外接口扩张前完成）。
- 多环境配置管理（开发/测试/生产）差异项收敛与模板化。
- 完善 godoc 注释与代码示例覆盖率（与 DX Track 对齐）。

## 性能与并发安全基线

- 性能回归采用相对提升百分比规则，详见 `docs/performance-policy.md`。
- 并发安全为强制门禁：`go test -race ./...` + goroutine 泄漏检查。
