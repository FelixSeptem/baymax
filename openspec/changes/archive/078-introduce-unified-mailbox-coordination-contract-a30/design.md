## Context

当前多代理主链路已经具备：
- A11：同步 `submit + wait`；
- A12：异步 `SubmitAsync + ReportSink`；
- A13：延后调度 `not_before`。

这些能力功能可用，但协作消息模型分散，导致：
- API 面重复（sync/async/delayed 各自维护）；
- 幂等与重试语义难以统一回归；
- 质量门禁需要在多个入口重复维护。

在 A29（Task Board 查询）实施阶段后，下一步最小闭环是统一消息协调契约。项目尚未对外使用，因此可以采用“新主路径 + 旧 API deprecate，不承诺兼容”的收敛策略。

## Goals / Non-Goals

**Goals:**
- 提供统一 mailbox 协调契约：`command/event/result` envelope + lifecycle。
- 提供统一 mailbox 查询 API（过滤、排序、分页、游标）。
- 将 sync/async/delayed 三种执行语义映射到 mailbox 契约。
- 新增 `mailbox.*` 配置域并纳入 runtime fail-fast 校验。
- 建立 mailbox contract suites 并接入 shared multi-agent gate。
- 对 A11/A12/A13 旧 API 进入 deprecate 路径，不再承诺兼容。

**Non-Goals:**
- 不引入平台化控制面（UI/RBAC/多租户/运维平台）。
- 不引入外部 MQ（Kafka/NATS/RabbitMQ）适配。
- 不承诺 exactly-once，采用 at-least-once + idempotency 收敛。
- 不改变 `runtime/*` 与 `mcp/*` 边界约束。

## Decisions

### 1) 新建 `orchestration/mailbox` 包作为统一协调面
- 方案：新增 mailbox 包，定义 envelope 模型、store 接口、发布消费与查询 API。
- 原因：消息协调语义属于 orchestration 域，避免把临时协调逻辑散落在 `a2a`/`invoke`/`scheduler`。
- 备选：在 `a2a` 内扩展。拒绝原因：会混淆传输互联语义与编排协调语义。

### 2) 交付语义采用 `at-least-once + idempotency-key`
- 方案：消息可重投，依赖 `idempotency_key` 收敛逻辑重复。
- 原因：兼顾可靠性与实现复杂度，符合当前 0.x 阶段。
- 备选：exactly-once。拒绝原因：实现成本和状态复杂度过高。

### 3) envelope 标准字段固定，支持延后与过期
- 方案：标准字段至少包括 `message_id`, `idempotency_key`, `correlation_id`, `kind`, `from_agent`, `to_agent`, `task_id`, `run_id`, `payload`, `not_before`, `expire_at`, `attempt`。
- 原因：覆盖同步、异步、延后三类语义并支持可观测关联。
- 备选：按场景定义多套 message model。拒绝原因：继续分裂契约。

### 4) mailbox backend 首期 `memory + file`，file 失败降级 memory
- 方案：默认 `memory`，支持 `file`；`file` 初始化失败时回退 `memory` 并发出诊断标记。
- 原因：沿用 scheduler/composer 既有治理模式，降低引入风险。
- 备选：仅 memory。拒绝原因：无法覆盖持久化恢复路径测试。

### 5) mailbox 查询契约与 A18/A29 同治理基线
- 方案：默认 `page_size=50`，最大 `200`，默认排序 `updated_at desc`，cursor 为 opaque 且绑定查询边界。
- 原因：统一查询 API 治理口径，降低使用复杂度。
- 备选：无分页上限。拒绝原因：内存和延迟风险不可控。

### 6) 旧 API 采用 deprecate + 主链路迁移，不保兼容
- 方案：A11/A12/A13 旧入口标注 deprecated，shared gate 以 mailbox 主路径为准。
- 原因：项目未对外使用，当前是最低成本收敛窗口。
- 备选：双栈长期兼容。拒绝原因：维护成本高且容易语义漂移。

### 7) 运行时配置新增 `mailbox.*` 并 fail-fast 校验
- 方案：新增 `mailbox.enabled/backend/path/retry/ttl/dlq/query` 等配置子域，非法值启动/热更新均 fail-fast。
- 原因：保持 `runtime/config` 一致治理原则。
- 备选：hard-code mailbox 行为。拒绝原因：无法满足不同部署需求和契约测试矩阵。

## Risks / Trade-offs

- [Risk] 一次性收敛三条 API 路径导致改动面较大  
  → Mitigation: 先保持内部桥接适配，再逐步切换调用方与 gate，最后清理旧路径。

- [Risk] mailbox 查询与 task board 查询出现职责重叠  
  → Mitigation: 文档明确“task board 看任务状态，mailbox 看消息状态”，通过关联键组合查询。

- [Risk] `file` backend 在并发下出现状态漂移  
  → Mitigation: 复用 snapshot/restore 一致性策略并增加 memory/file parity contract suites。

- [Risk] 不保兼容导致内部调用改造压力上升  
  → Mitigation: 在 A30 tasks 中显式列出迁移顺序和 deprecate 清单，保证单变更可回归。

## Migration Plan

1. 新增 `orchestration/mailbox` 核心模型、store 接口、memory/file backend 与 query API。
2. 增加 mailbox->scheduler/a2a 适配层，打通 sync/async/delayed 三种路径。
3. 在 `runtime/config` 增加 `mailbox.*` 并补齐 fail-fast 校验与热更新回滚语义。
4. 在 `runtime/diagnostics` 增加 mailbox 相关汇总字段与查询关联键。
5. 将 shared gate 的 A11/A12/A13 主路径切换至 mailbox contract suites。
6. 旧 API 标记 deprecated，文档迁移到 mailbox 主路径。

回滚策略：
- 若 mailbox 主路径不稳定，可回滚 gate 切换与适配层接入；
- 保留旧路径代码作为短期兜底，但不再增加契约承诺。

## Open Questions

无阻塞项，按推荐值冻结：
- `mailbox.backend=memory`（默认）
- `mailbox.retry.max_attempts=3`
- `mailbox.retry.backoff_initial=50ms`
- `mailbox.retry.backoff_max=500ms`
- `mailbox.retry.jitter_ratio=0.2`
- `mailbox.ttl=15m`
- `mailbox.dlq.enabled=false`
- `mailbox.query.page_size_default=50`
- `mailbox.query.page_size_max=200`
