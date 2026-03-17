# Development Roadmap

更新时间：2026-03-17

## 目标

在当前 v1 基线能力上，进入可发布、可运营、可扩展的工程化阶段，重点提升：
- 稳定性（错误恢复、兼容性、回归防护）
- 可运维性（配置、观测、调试工具）
- 可扩展性（模型/工具/MCP/技能生态）

## Open Source P0（仅保留最重要项）

目标：以最小成本满足“可被外部团队安全使用与协作”的开源基线。

### P0-1 发布与兼容承诺
- 统一版本策略（SemVer）与升级兼容承诺（含 breaking change 说明规则）。
- 明确 Go 版本支持窗口与 provider 支持级别。
- 在 README + docs 保持单一口径，不出现冲突描述。

验收标准：
- 可按固定模板发布一个规范版本（含 changelog/release notes）。
- 外部用户可明确判断“能否升级、升级风险是什么”。

### P0-2 安全响应入口
- 增加 `SECURITY.md`，定义漏洞报告渠道、响应时限、修复与披露流程。
- 保持 `govulncheck` 质量门禁为默认强制路径。

验收标准：
- 安全问题具备可执行、可追踪的响应流程。
- 主干 CI 对安全基线失败可阻断合并。

### P0-3 贡献与评审最小闭环
- 增加 `CONTRIBUTING.md` 与 Issue/PR 模板。
- 补齐最小评审清单（测试、文档同步、兼容性影响）。

验收标准：
- 外部贡献者可独立完成一次标准提交流程。
- 维护者可按清单执行一致化评审。

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

### 进展（2026-03-17）
- [x] `harden-security-baseline-s1-govulncheck-and-redaction`：完成 `govulncheck` strict 质量门禁接入（Linux/PowerShell/CI 一致语义）。
- [x] 完成统一脱敏管线落地（diagnostics/event/context assembler），并补齐关键词扩展口与回归测试。
- [x] `harden-security-s2-tool-permission-rate-limit-and-io-filter-e8`：完成 `namespace+tool` 权限策略、进程级限流、模型输入/输出过滤扩展口与 deny 默认语义收敛。
- [x] `introduce-security-s3-event-taxonomy-and-callback-alerting-e9`：完成安全事件 taxonomy（`policy_kind|decision|reason_code|severity`）与 deny-only callback 告警契约。
- [x] `harden-security-s4-callback-delivery-reliability`：完成 callback 投递可靠性治理（默认 `async`、有界队列 `drop_old`、最多 3 次重试、Hystrix 风格熔断）与 Run/Stream 语义等价验证。
- [x] 补齐安全契约门禁脚本：`check-security-policy-contract.*`、`check-security-event-contract.*`、`check-security-delivery-contract.*`。

### 当前状态
- 已形成 S1-S4 安全闭环：扫描与脱敏基线、策略执行与过滤、事件归一化、告警可靠投递全部落地。
- 安全策略与投递行为已纳入 runtime config 热更新与 fail-fast 校验路径，保持 `env > file > default` 一致语义。

### 下一阶段规划（2026-03-17）
- S4 运营化：基于 `alert_*` 诊断字段建设可观测看板与告警阈值建议，形成 callback sink SLO 基线。
- 策略模板化：沉淀 `namespace+tool` 权限/限流与 I/O 过滤最小模板，降低外部接入初始配置成本。
- 安全回归强化：扩展安全契约测试覆盖混合并发场景（tool/mcp/skill + 安全告警）并保持 Run/Stream 等价语义。

### 验收标准
- CI 包含安全扫描与安全契约门禁，且可稳定运行。
- 诊断与日志中的敏感字段可控脱敏，无明文泄漏回归。
- 高风险工具调用具备可审计的权限、限流与 I/O 过滤策略。
- deny 告警投递可观测（重试、队列丢弃、熔断状态）且具备可演练基线。

## Phase R3（6-8 周）生态扩展与开发者体验

### 进展（2026-03-12）
- [x] `bootstrap-multi-llm-providers-m1`：完成 `model/anthropic`、`model/gemini` 官方 SDK 最小非流式适配。
- [x] 新增跨 provider 契约测试（OpenAI/Anthropic/Gemini）最小成功路径与基础错误分类一致性。
- [x] `align-multi-provider-streaming-and-error-taxonomy-m2`：完成 Anthropic/Gemini streaming 接入、跨 provider 事件语义对齐与错误分类细化。
- [x] `add-provider-capability-detection-and-fallback-m3`：完成基于官方 SDK 的动态能力探测、model-step preflight、provider 级有序降级与 fail-fast 终止。
- [x] `build-context-assembler-ca1-prefix-append-only-baseline`：完成 pre-model hook、immutable prefix hash、一致性 fail-fast、append-only JSONL journal 与 CA1 最小诊断字段。
- [x] `implement-context-assembler-ca2-lazy-stage-routing-and-tail-recap`：完成 CA2 双阶段路由、file provider、tail recap 与 CA2 诊断字段。
- [x] `activate-ca2-external-retriever-spi-and-http-adapter`：完成 Stage2 External Retriever SPI、HTTP adapter、`http/rag/db/elasticsearch` 可运行路径与新增诊断字段。
- [x] `harden-ca2-external-retriever-observability-and-thresholds-e2`：完成 CA2 external provider 维度趋势聚合与静态阈值信号（`p95_latency_ms/error_rate/hit_rate`，默认窗口 `15m`）。
- [x] `extend-ca2-external-retriever-capability-hints-and-template-pack-e3`：完成 CA2 external capability hints 扩展口与 template pack 收敛（`graphrag_like|ragflow_like|elasticsearch_like` + `explicit_only`），新增 `stage2_template_*` / `stage2_hint_*` 诊断字段、Run/Stream 等价契约与 hint/template 解析 benchmark 基线。
- [x] `implement-ca2-agentic-routing-baseline`：完成 CA2 `agentic` callback 路由基线、`best_effort_rules -> rules` 失败回退、最小路由诊断字段（`stage2_router_*`）与 Run/Stream 语义等价契约。
- [x] `add-r3-advanced-concurrency-pattern-examples-05-07`：完成 R3 高阶示例扩容（05/06/07/08），并为异步与多代理示例补齐结构化事件输出与 runtime manager 接入。
- [x] `standardize-action-timeline-events-h1`：完成 Action Timeline 结构化事件契约（Run/Stream 语义一致、默认启用、`context_assembler` 独立 phase、新增 `canceled` 状态）。
- [x] `converge-action-timeline-observability-h15`：完成 Action Timeline phase 级聚合可观测收敛（含 `latency_p95_ms`、重放幂等、Run/Stream 分布等价）。
- [x] `add-cross-run-timeline-trend-aggregation-h16`：完成跨 run 窗口趋势聚合（`last_n_runs` + `time_window`），支持 `phase+status` 双维度与 `latency_p95_ms` 指标。
- [x] `implement-context-assembler-ca3-memory-pressure-and-recovery`：完成 CA3 内存压力控制与恢复（五级分区、双阈值触发、squash/prune、spill/swap、Run/Stream 语义一致、CA3 诊断字段）。
- [x] `implement-context-assembler-ca4-production-convergence`：完成 CA4 生产收敛（阈值解析顺序、token 计数固定回退、Run/Stream 契约增强、CA4 benchmark 门禁）。
- [x] `introduce-ca3-semantic-compaction-spi-f1`：完成 CA3 compaction SPI（`truncate|semantic`）、semantic 走当前 model client、`best_effort` 回退/`fail_fast` 终止语义、evidence retention 规则与新增诊断字段。
- [x] `harden-ca3-semantic-compaction-quality-and-template-controls-f2`：完成 CA3 semantic compaction 质量门控（规则评分+阈值）、runtime 模板白名单控制、embedding SPI hook 占位、新增质量/回退诊断字段与 benchmark 基线。
- [x] `implement-ca3-semantic-embedding-adapter-e3`：完成 CA3 semantic embedding adapter（OpenAI/Gemini/Anthropic 选择）、cosine 混合评分、独立 embedding 凭证配置与 `ca3_compaction_embedding_*` 诊断字段。
- [x] `harden-ca3-semantic-reranker-and-threshold-tuning-e4`：完成 CA3 reranker 加固（provider-specific 扩展接口、mandatory provider/model 阈值 profile、Anthropic 可用路径）、新增 `ca3_compaction_reranker_*` 诊断字段与离线阈值调优工具（markdown 输出）。
- [x] `govern-ca3-threshold-rollout-and-observability-e5`：完成 CA3 阈值治理与灰度收敛（`enforce|dry_run`、`provider:model` rollout、profile version），补齐治理诊断字段与契约/benchmark 门禁。
- [x] `harden-runner-cancel-storm-and-backpressure-baseline-r5`：完成 runner 取消风暴与背压基线收敛（默认 `block`、`cancel.propagated`/`backpressure.block` reason、并发诊断字段、Run/Stream 契约对齐、cancel-storm benchmark 输出 `p95` + `goroutine peak`）。
- [x] `introduce-skill-trigger-scoring-and-contract-tests-d1`：完成 skill trigger scoring 收敛（默认 lexical weighted-keyword、`highest_priority` tie-break、低置信度抑制默认开启、runtime YAML 配置与合同测试）。
- [x] `introduce-drop-low-priority-backpressure-r6`：完成 `drop_low_priority` 背压策略基线（local dispatch）、全量 drop fail-fast、`backpressure.drop_low_priority` reason 与契约测试/benchmark 收敛。
- [x] `extend-drop-low-priority-backpressure-to-mcp-and-skill-r7`：将 `drop_low_priority` 语义扩展到 `local+mcp+skill`，补齐分桶诊断、跨路径 fail-fast 与契约/benchmark 收敛。

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
  - CA2（按需加载，已完成）：接入 Stage1/Stage2 路由、可配置 stage 策略、tail recap；支持 `file/http/rag/db/elasticsearch` 与 external retriever SPI。
  - CA3（压力控制）：落地分级压力响应策略（安全区/舒适区/警告区/危险区/紧急区），支持绝对阈值+百分比双模式配置，batch squash/prune（带"不可压缩"标记），spill/swap 回填，补充监控指标。
  - CA4（生产收敛）：补齐规则防护、可中断恢复、观测面板与契约测试闭环。
  - 详细分期见 `docs/context-assembler-phased-plan.md`。
- Skill 语义触发升级（可插拔检索/打分器）。
- Agent Action 输出体验（规划）：
  - 基于现有事件流构建用户侧 Action Timeline（`run/context_assembler/model/tool/mcp/skill` 阶段）。
  - 增加统一动作状态语义（`pending/running/succeeded/failed/skipped/canceled`）。
  - 增加 step/phase 关联字段规范，支持前端稳定渲染执行路径。
  - 已完成 H1.5：phase 级聚合字段收敛到 diagnostics 契约。
  - 已完成 H16：跨 run 趋势聚合与窗口化指标（库接口，默认启用，兼容增量字段）。
- 提供最小 CLI 示例（本地调试和回放）。
- 交付 R3 高阶示例：`05-parallel-tools-fanout`、`06-async-job-progress`、`07-multi-agent-async-channel`、`08-multi-agent-network-bridge`。
  - TODO：结合 CA2 增加 staged context 路由示例（本提案不新增 example 代码）。
- Knowledge 基础能力（R3，先接口后实现）：
  - 向量检索 provider 抽象（已完成 SPI + HTTP adapter；保持不绑定具体供应商 SDK）
  - 文档解析与分片策略接口（parser/chunker contract），与 Context Assembler 对齐
  - 检索结果与 CA2 Stage2 集成约定（保持 fail-fast/best-effort 语义一致）
  - 通过配置映射请求/响应字段与鉴权信息接入外部服务（已完成最小闭环）
  - 已完成首批 provider 风格映射模板（GraphRAG/RAGFlow/ES）；后续按接入反馈继续扩展模板集合与示例。

- CA2 External Retriever 后续演进（R3-R4）：
  - 详细计划、触发门槛、观测治理统一维护在 `docs/ca2-external-retriever-evolution.md`（单一事实源）。
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

## 近期执行计划（2026-03-17 起）

- P0 文档一致性收敛：以 `docs/development-roadmap.md` 与 `openspec/changes/archive/INDEX.md` 为主口径，持续消除 README 与验收文档状态漂移。
- P1 安全运营收敛：聚焦 S4 callback 投递稳定性，建立常态化回放演练与失败原因分层分析。
- P1 并发可靠性压测：扩展 runner 压测到混合路径（高并发 tool/mcp/skill + cancel storm + security delivery 干扰）。
- P2 DX D2 精简模式推进：保持 library-first，不引入重 CLI 依赖，优先补齐可复现的 replay/配置校验/事件查看入口。
- P2 CA2/CA3 调优连续性：按 profile version 治理阈值迭代，沉淀可复现调优语料与回归基线。

### Naming Migration（规划，当前不实现）

- 目标：在不破坏现有语义与兼容性的前提下，逐步将 `CA1/CA2/CA3/CA4` 迁移为更直观的 phase 命名。
- N1（别名期）：文档采用双命名（如 `CA4 / Production Convergence`），不改代码符号与对外字段。
- N2（兼容期）：配置/诊断/事件层支持新旧命名并行读取与输出，补齐兼容契约测试与迁移指南。
- N3（收敛期）：新命名成为默认口径，旧命名进入 deprecation 窗口并按版本计划移除。
- 验收：Run/Stream 语义不变；迁移窗口内向后兼容；README/docs/runtime-config-diagnostics/roadmap 命名一致。
- 非目标：本轨道不引入新功能，不触发 package path 重命名。

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
- CA3（R4 前半，已完成）：内存压力与可恢复性
  - 对齐 P7/P8 + Arena 机制（可中断、不丢信息、batch reset、spill/swap）。
- CA4（R4 后半，已完成）：生产级策略收敛
  - 已完成阈值解析顺序、token 计数 fallback、契约测试、压测阈值与文档统一。

说明：Context Assembler 不建议单次大改完成，需按分期逐步启用，保证 runner 语义稳定与并发安全基线不回退。

## HITL 与 Action Timeline 里程碑（规划）

- H1（R3 前半，已完成）：交付 Action Timeline 标准化与字段规范，不改 runner 主状态机。
- H1.5（R3-R4，已完成）：补齐 timeline 聚合可观测字段（phase 级计数/耗时/失败率）并与 diagnostics 契约对齐。
- H16（R4，已完成）：补齐跨 run 趋势聚合（`last_n_runs|time_window`）与 `phase+status` 双维度指标，保持 Run/Stream 语义一致与幂等统计。
- H2（R3 后半，已完成）：引入 Action Gate（执行前确认钩子），默认 `require_confirm`、timeout-deny、Run/Stream 语义一致，并收敛最小诊断字段（`gate_checks/gate_denied_count/gate_timeout_count`）。
- H3（R4，已完成）：引入原生 clarification 生命周期（`await_user` / `resumed` / `canceled_by_user`），默认超时策略 `cancel_by_user`，并完成 Run/Stream 契约测试与诊断字段（`await_count/resume_count/cancel_by_user_count`）。
- H4（R4，已完成）：引入 Action Gate 参数规则（`path/operator/expected` + `AND/OR` 复合条件），支持规则 action 继承与优先级收敛（参数规则优先），新增 `gate.rule_match` reason 与最小诊断字段（`gate_rule_hit_count/gate_rule_last_id`）。

## A2A 里程碑（规划，当前不实现）

- A1（R3 预研）：完成 A2A 协议语义映射设计（任务状态机、SSE/推送、能力发现），输出设计文档与边界说明。
- A2（R4 前半）：实现最小 A2A Client/Server 互联能力，支持基础任务提交、状态查询与结果回传。
- A3（R4 后半）：与现有 observability/diagnostics 集成，补齐跨协议契约测试（A2A + MCP 组合场景）。

参考资料：
- https://a2aprotocol.ai/blog/2025-full-guide-a2a-protocol-zh

## 技术债清单（当前建议优先）

- 清理仓库中的临时/备份产物与目录规范化（持续项）。
- 收敛 `mcp/http` 与 `mcp/stdio` 中重复的重试/事件逻辑到共享组件。
- TODO（skill scoring 演进）：当前仅实现 lexical weighted-keyword；后续在 scorer internal 接口上增量接入 embedding 检索/打分能力。
- 为 runner 添加更多混合压力测试（高并发 tool/mcp/skill + 取消风暴 + 安全告警投递干扰场景）。
- 统一错误分类与错误处理策略（细化 error taxonomy 与跨模块映射）。
- API 版本控制与兼容策略文档化（在对外接口扩张前完成）。
- 多环境配置管理（开发/测试/生产）差异项收敛与模板化。
- 完善 godoc 注释与代码示例覆盖率（与 DX Track 对齐）。
- CA2 external retriever 观测增强：按 `docs/ca2-external-retriever-evolution.md` 收敛指标口径与触发阈值配置。

## 性能与并发安全基线

- 性能回归采用相对提升百分比规则，详见 `docs/performance-policy.md`。
- 并发安全为强制门禁：`go test -race ./...` + goroutine 泄漏检查。

