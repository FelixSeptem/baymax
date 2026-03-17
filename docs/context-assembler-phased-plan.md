# Context Assembler Phased Plan

更新时间：2026-03-16

## 目标

将 Context Assembler 作为 `runner` 的 pre-model 阶段能力逐步落地，遵循以下核心原则：

- P1 前缀一致性：连续调用间 prefix 字节级不变
- P2 Append-Only：信息只追加到末尾，不插入不重排
- P3 按需加载：不预装可推迟的信息
- P4 渐进降级：Stage 1 可解则不上 Stage 2
- P5 末尾复述：关键状态推送到上下文末尾
- P6 不信任 LLM：防护基于规则与 hash，不依赖模型自觉
- P7 可中断性：任意操作可中断且状态一致
- P8 不丢信息：可压缩降级，但完整路径可追溯
- P9 可观测性：不可测量即不可优化
- P10 文件即记忆：优先使用文件系统承载工作状态

## 分期总览

| 里程碑 | 阶段 | 重点原则 | 交付重点 |
| --- | --- | --- | --- |
| CA1 | R3 前半 | P1/P2/P6/P9/P10 | pre-model hook + immutable prefix + append-only journal |
| CA2 | R3 后半 | P3/P4/P5 | Stage1/Stage2 lazy loading + tail recap |
| CA3 | R4 前半 | P7/P8 | memory pressure control + squash/prune + spill/swap |
| CA4 | R4 后半 | 全量收敛 | 策略参数化 + 契约测试 + 压测指标 |

## CA1：Prefix + Append-only Baseline

状态：已完成（2026-03-12）

### 范围

- 新增 `context/assembler` 最小骨架，接入 `core/runner` model step 前置钩子。
- 定义 immutable prefix（CA1 当前以 system messages + prefix version + capability snapshot 为基线块）并生成 `prefix_hash`。
- 建立 append-only `context/journal`（JSONL）与事件 schema（intent/commit）。
- 引入基础 guard：hash 校验、schema 校验、敏感字段脱敏。
- storage backend 支持 `file`，`db` 仅占位并返回 unsupported（fail-fast）。
- 诊断字段打通到 run 摘要：`prefix_hash`、`assemble_latency_ms`、`assemble_status`、`guard_violation`。

### 验收

- 同一 session 连续调用 `prefix_hash` 稳定（无显式升级时）。
- 不发生中间插入/重排。
- 诊断可看到 assemble 时延、prefix hash、基础命中率。

### CA1 边界（未包含）

- 未接入 Stage2 retrieval orchestration（RAG / long-term memory）。
- 未接入 memory pressure 控制（Goldilocks/squash/prune/spill-swap）。
- 未改动外部 complete-only tool-call 事件契约。

## CA2：Lazy Allocation + Stage Routing

状态：已完成（基础版，2026-03-12）

### 范围

- 引入两阶段装配：
  - Stage1：session-memory / hot state
  - Stage2：retrieval provider（支持 `file/http/rag/db/elasticsearch`）
- 支持 `best-effort/fail-fast` 两种模式与超时策略。
- 增加 tail recap（规则模板）将 `todo/decisions/status` 推送末尾。
- 路由模式默认 rules；agentic hook 仅预留 TODO 扩展点。
- 新增外部检索接入：统一 Retriever SPI + 通用 HTTP adapter（JSON 映射、Bearer/自定义鉴权头）。
- 新增 Stage2 诊断字段：`stage2_hit_count`、`stage2_source`、`stage2_reason`。

### 验收

- Stage1 可满足时不触发 Stage2。
- Stage2 触发链路可观测（触发原因、耗时、命中率）。
- tail recap 在输出中位置稳定且可追溯来源。

### CA2 边界（未包含）

- 未实现 agentic routing 决策执行，仅保留接口占位与显式 not-ready 错误。
- examples 目录本期不新增 CA2 示例，后续在 roadmap TODO 批次补齐。

### CA2 External Retriever 演进约束（与 roadmap 对齐）

- 详细计划、触发门槛与观测治理统一维护在 `docs/ca2-external-retriever-evolution.md`（单一事实源）。

## CA3：Memory Pressure + Recovery

状态：已完成（2026-03-13）

### 范围

- 引入以 Goldilocks Zone（默认 `35%-60%`）为目标的分级压力响应策略：
  - **安全区（< 30%）**：正常加载新内容。
  - **舒适区（30%-50%）**：限制新内容加载预算，优先进行现有内容优化。
  - **警告区（50%-70%）**：触发 squash（软压缩），优先处理低价值内容。
  - **危险区（70%-90%）**：触发 prune（硬删除），按优先级保留内容。
  - **紧急区（> 90%）**：触发 spill/swap（溢出到磁盘）并进入保护模式（默认拒绝低优先级加载请求）。
- 阈值设计支持双模式配置：
  - 绝对阈值（默认 safe/comfort/warning/danger/emergency=`24000/48000/72000/96000/115200`）+ 百分比阈值（默认 `20/40/60/75/90`），满足任一条件即触发对应策略。
  - 不同阶段可配置不同阈值（如 Stage2 检索前可更激进）。
- 支持 batch squash/prune（不做单块回收，避免碎片化）：
  - **Squash**：摘要合并、冗余消除、结构化压缩。
  - **Prune**：按重要性评分排序，优先删除低相关内容（优先规则评分，保留评分扩展口）。
  - 支持“不可压缩/不可删除”标记（critical/irreversible 类内容）。
- Compaction 策略支持 `truncate|semantic`（默认 `truncate`）：
  - `semantic` 通过当前 model client 执行语义压缩。
  - `best_effort` 下语义失败回退 `truncate`；`fail_fast` 下语义失败立即终止。
- CA3 semantic F2 收敛（已完成）：
  - 质量门控：规则评分（coverage/compression/validity）+ 阈值判定。
  - 模板控制：runtime 模板与占位符白名单（启动/热更新 fail-fast）。
  - embedding SPI hook：仅通用接口占位，不绑定 adapter。
  - 诊断字段：`ca3_compaction_quality_score`、`ca3_compaction_quality_reason`、`ca3_compaction_fallback_reason`。
- CA3 semantic E3 收敛（已完成，2026-03-16）：
  - embedding adapter：支持 `openai|gemini|anthropic` 三 provider 路径选择。
  - 混合评分：规则分 + cosine 相似度分量，默认权重 `rule=0.7`、`embedding=0.3`。
  - 独立凭证：支持 `embedding.auth.*` 与 `embedding.provider_auth.<provider>.*`。
  - 诊断字段：新增 `ca3_compaction_embedding_*` 系列字段用于 provider/状态/贡献观测。
- CA3 semantic E4 加固（已完成，2026-03-17）：
  - 新增 reranker 阶段（默认关闭），位于 base hybrid score 之后、最终 gate 之前。
  - reranker 配置：`context_assembler.ca3.compaction.reranker.enabled|timeout|max_retries|threshold_profiles`。
  - 阈值 profile：开启 reranker 时要求 provider/model 对应阈值 profile 存在（fail-fast）。
  - 扩展接口：支持 provider-specific reranker 注册（`assembler.WithSemanticReranker`）。
  - 新增诊断字段：`ca3_compaction_reranker_*` 系列。
  - 提供离线阈值调优工具入口：`cmd/ca3-threshold-tuning`（最小 markdown 输出）。
- CA3 semantic E5 治理收敛（已完成，2026-03-17）：
  - 阈值治理配置：`reranker.governance.mode|profile_version|rollout_provider_models`。
  - 灰度粒度：仅 `provider:model`；支持 `enforce|dry_run` 两种模式。
  - 语义约束：`best_effort` 回退与 `fail_fast` 终止语义保持不变，Run/Stream 等价。
  - 新增诊断字段：`ca3_compaction_reranker_profile_version|rollout_hit|threshold_drift`。
- Prune 证据保留规则：`keywords + recent_window`，并输出保留计数诊断字段。
- 支持 spill/swap（文件落盘与按需回填），保留 provenance path：
  - 溢出内容保留 `origin_ref` 便于完整路径追溯。
  - 本期实现文件后端；DB/对象存储仅接口占位。
- Token 计数策略：
  - 默认 `sdk_preferred`，优先官方 SDK tokenizer（Anthropic/Gemini）。
  - 小增量更新先走预估（`small_delta_tokens`），超过阈值再触发 SDK 计数，降低调用频率。
- 完善中断恢复流程（cancel/retry/replay）。
- 补充监控指标：
  - 各区域停留时间分布。
  - squash/prune 触发频率与压缩率。
  - 内容价值分布（基于访问频率/相关性评分）。

### 验收

- 分级策略可配置且生效，不同阈值组合按预期触发对应操作。
- 压测下 usage 稳定在目标区间（默认以 `35%-60%` 为主）。
- squash/prune 支持“不可压缩/不可删除”标记，critical 内容不被删除。
- 中断后恢复不出现状态撕裂。
- 被压缩/溢出内容可通过 `origin_ref` 回溯完整路径。
- 监控指标覆盖各区域停留时间、触发频率、压缩率等。

## CA4：Production Hardening

状态：已完成（2026-03-16）

### 范围

- 固化阈值解析顺序：`stage override -> percent/absolute 双触发并行评估 -> 取更高压力分区`。
- 固化 token 计数回退链路：`provider counter -> local tiktoken estimate -> lightweight estimate`（counting-only fail-open）。
- 补齐契约测试矩阵（Run/Stream 的 zone/reason/trigger 语义一致、small-delta 与 refresh interval、fallback 分支）。
- 完成诊断字段与文档统一，纳入 CA4 benchmark 相对百分比门禁（含 `p95`）。

### 验收

- `go test ./...`、`go test -race ./...`、`golangci-lint` 全通过。
- 契约测试覆盖关键降级与恢复场景（含 provider unsupported 与 local tokenizer unavailable）。
- `BenchmarkCA4PressureEvaluation` 通过相对回归门禁（`ns/op` 与 `p95-ns/op`）。
- 文档与实现行为一致，无语义漂移。

## 风险与边界

- 不在 CA1-CA2 引入向量数据库生命周期管理；provider 已支持外部接入但不绑定具体厂商 SDK。
- CA2 external retriever 的性能治理依赖观测基线，指标与阈值口径以 `docs/ca2-external-retriever-evolution.md` 为准。
- 不在该分期内引入 HITL pause/resume 主状态机改造。
- 不改变现有 tool-call complete-only 对外语义与 streaming 事件契约。
