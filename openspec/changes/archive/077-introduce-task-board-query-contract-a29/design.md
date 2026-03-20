## Context

A18 提供了 run 维度统一查询（`QueryRuns`），但 scheduler 任务态仍以 `Get/Stats/Snapshot` 为主，缺少标准化批量检索能力。对于多代理编排场景，调用方需要同时观察队列态（`queued/running/dead_letter`）与 run 诊断摘要；当前缺乏稳定、可回归的任务看板查询契约。

项目定位仍是 `library-first`：本次只补齐库接口与契约测试，不引入平台化控制面或任务运维 UI。

## Goals / Non-Goals

**Goals:**
- 在 `orchestration/scheduler` 提供只读任务看板查询 API。
- 固化过滤、分页、排序、游标、错误分类的确定性语义。
- 保证 memory/file backend 下查询语义一致。
- 与既有观测链路对齐：通过 `run_id/task_id` 与 `QueryRuns` 可组合关联。
- 将契约测试纳入 shared multi-agent gate 与主干索引。

**Non-Goals:**
- 不引入任务写操作（cancel/retry/reassign/priority mutate）。
- 不引入 mailbox、任务控制台、RBAC、多租户运维面。
- 不引入外部数据库/全文检索引擎。
- 不替代既有 `QueryRuns`；二者并存并互补。

## Decisions

### 1) API 落在 scheduler 包内，采用 QueryRequest/QueryResult 模型
- 方案：新增 `QueryTasks(ctx, req)` 与 `TaskBoardQueryRequest/TaskBoardQueryResult`。
- 原因：任务态语义属于 scheduler 领域，避免跨模块倒灌依赖。
- 备选：放入 `runtime/config.Manager`。拒绝原因：manager 不直接持有 scheduler 运行态，耦合边界不清晰。

### 2) 查询数据源基于 scheduler snapshot，保证后端无关
- 方案：查询执行基于 `Snapshot` 读路径做过滤/排序/分页。
- 原因：memory/file 后端可复用同一查询逻辑并保持一致行为。
- 备选：为每种后端实现独立查询。拒绝原因：容易语义漂移，维护成本高。

### 3) 过滤语义固定为 AND
- 方案：请求中多个过滤条件按 `AND` 组合。
- 原因：结果可预期，便于构建稳定契约测试。
- 备选：支持 OR/表达式。拒绝原因：首版复杂度高，收益不匹配。

### 4) 分页与排序采用与 A18 一致的治理基线
- 方案：`page_size` 默认 `50`、上限 `200`；默认排序 `updated_at desc`；首版支持 `updated_at|created_at` 排序字段。
- 原因：与既有查询契约一致，降低调用方认知成本并约束性能风险。
- 备选：默认 100/1000。拒绝原因：单次查询成本和抖动风险更高。

### 5) 游标使用 opaque token + query-hash 绑定
- 方案：游标不暴露内部偏移含义；游标必须匹配当前查询边界，否则 fail-fast。
- 原因：避免游标被跨查询复用导致结果漂移。
- 备选：公开 offset。拒绝原因：调用方容易构造不稳定分页行为。

### 6) 错误语义区分“参数非法”与“无匹配”
- 方案：非法状态枚举、非法时间范围、非法 page_size、非法 cursor 均 fail-fast；合法无匹配返回空集。
- 原因：与 A18 一致，便于调用方重试与告警分层。
- 备选：统一返回空集。拒绝原因：会掩盖调用方参数错误。

### 7) Task Board 与 QueryRuns 采用“可组合关联”而非“强耦合合并”
- 方案：Task Board 响应保留 `run_id/task_id/workflow_id/team_id` 等关联键，由调用方按需再调用 `QueryRuns` 获取 run 级摘要。
- 原因：避免跨域查询逻辑过早耦合，同时满足 scheduler + diagnostics 的组合观测诉求。
- 备选：在 Task Board 查询内直接内联 run 诊断。拒绝原因：跨模块耦合升高，首版复杂度偏大。

## Risks / Trade-offs

- [Risk] 快照后内存过滤/排序在任务量大时开销上升  
  → Mitigation: 固定分页上限、默认倒序，后续按实测再评估索引化提案。

- [Risk] 查询期间状态变化造成调用方“读到旧页”感知  
  → Mitigation: 文档明确查询是快照语义；游标绑定查询边界并 fail-fast 防止混页。

- [Risk] 与 `QueryRuns` 职责边界被误解  
  → Mitigation: README 与 runtime 文档明确“任务态看板 vs run 摘要”分工。

## Migration Plan

1. 在 scheduler 增加 Task Board 请求/响应模型与参数校验。
2. 基于 snapshot 实现过滤、排序、分页和 opaque cursor。
3. 增加 unit/integration 合同测试（AND 语义、空集、fail-fast、游标稳定、memory/file 一致性）。
4. 接入 `check-multi-agent-shared-contract.*` 与主干契约索引。
5. 同步更新 roadmap/README/runtime 文档中的能力边界与使用方式。

回滚策略：
- 若新查询路径出现稳定性问题，可回滚 `QueryTasks` 入口与 gate 接入；
- 既有 scheduler 执行链路（enqueue/claim/commit）与 `QueryRuns` 不受影响。

## Open Questions

当前无阻塞问题；按推荐值冻结：
- 默认分页：`50`
- 最大分页：`200`
- 默认排序：`updated_at desc`
- 首版排序字段：`updated_at`、`created_at`
- 首版仅只读查询，不含写操作。
