# Context Assembler Phased Plan

更新时间：2026-03-13

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

### 范围

- 引入 Goldilocks Zone（默认 `40%-70%`）与压力监控。
- 支持 batch squash/prune（不做单块回收，避免碎片化）。
- 支持 spill/swap（文件落盘与按需回填），保留 provenance path。
- 完善中断恢复流程（cancel/retry/replay）。

### 验收

- 压测下 usage 稳定在目标区间。
- 中断后恢复不出现状态撕裂。
- 被压缩内容可通过 `origin_ref` 回溯完整路径。

## CA4：Production Hardening

### 范围

- 参数化策略收敛（budget/top_k/timeout/cache_ttl/squash 阈值）。
- 补齐契约测试矩阵（Run/Stream 语义一致、fail-fast/best-effort、一致性与恢复）。
- 完成诊断字段与文档统一，纳入 CI 质量门禁。

### 验收

- `go test ./...`、`go test -race ./...`、`golangci-lint` 全通过。
- 契约测试覆盖关键降级与恢复场景。
- 文档与实现行为一致，无语义漂移。

## 风险与边界

- 不在 CA1-CA2 引入向量数据库生命周期管理；provider 已支持外部接入但不绑定具体厂商 SDK。
- CA2 external retriever 的性能治理依赖观测基线，指标与阈值口径以 `docs/ca2-external-retriever-evolution.md` 为准。
- 不在该分期内引入 HITL pause/resume 主状态机改造。
- 不改变现有 tool-call complete-only 对外语义与 streaming 事件契约。
