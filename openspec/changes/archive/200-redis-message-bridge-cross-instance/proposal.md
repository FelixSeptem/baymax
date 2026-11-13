## Why

当前运行态在多实例部署下只能做到「同一台机器的多进程」：`core/sourcecoordination` 的 `File` 协调器用本地 `flock` + 原子 rename 提供同机多写者串行化，`orchestration/mailbox` 只有 `MemoryStore` / `FileStore`，实时游标 `realtimeCursors` 是纯进程内存，连接的 pending request 直接挂在单个 `host.Connection` 上。`cross_machine` 在 `core/history/testdata/owner_store_deployment.v1.json` 里被显式定为 `deployment_unsupported`，`README.md` 也写明跨机器协调不声明支持。

结果是：仓库没有任何消息总线、leader 选举或共享状态存储，`go.mod` 也没有 Redis/Kafka/NATS 依赖；一个 pod 上产生的 mailbox 消息、source 事实与 resume 游标，另一个 pod 完全看不到。水平扩展只能靠外部 sticky routing 硬扛，且一旦连接落到别的 pod，中断恢复与 mailbox 消费都会失效。

要真正支持跨 pod 多实例，需要把三类状态从进程内存/本地文件搬到集中式存储，并给出可验证的分布式 fencing 与 CAS 证据：

1. source-owner 协调（fencing + revision CAS）——决定谁能写；
2. mailbox 消息（publish/consume/ack/requeue/inflight）——决定消息怎么跨实例流动；
3. 运行态游标与实例归属（realtime cursor、session affinity）——决定流式会话落到哪个实例、能否续上。

Redis 具备这三件事所需的原子原语（`SET NX PX`、Lua 脚本、sorted set、pub/sub），且部署成本低，是本轮选定的集中式后端。

## What Changes

- 新增 `redis-cross-instance-coordination` 能力：定义 Redis 作为集中式协调/消息桥接后端的接入契约。
- 扩展部署范围契约：新增 `cross_machine` 部署画像的合法表达（`deployment_profile=cross_machine`、`fencing_scope=distributed`、`transaction_scope=distributed`），只有在显式配置分布式协调器并具备 fencing/CAS 证据时才 `Supported=true`；未配置时继续保持 fail-closed 的 `deployment_unsupported`。
- 实现 `core/sourcecoordination.Redis`：在既有 `Coordinator` 接口上提供 `Acquire`（`SET NX PX` 租约）、`CompareAndSwap`（Lua 原子校验 fence + revision）、`ValidateLease`、`DeploymentScope`（cross-machine 画像）。
- 实现 `orchestration/mailbox.RedisStore`：在既有 `Store` 接口上提供 publish/consume/heartbeat/ack/nack/requeue/stats/snapshot/restore，inflight 归属与超时回收用 Redis 原子脚本保证，语义与 `FileStore` 对齐。
- 新增运行态游标存储抽象（`CursorStore`）与 Redis 实现，允许把 `realtimeCursors` 从进程内存外置；默认仍是内存实现，行为不变。
- 新增实例注册表与 session affinity 记录：实例心跳（带 TTL）、`session → instance` 归属写入与查询，供宿主路由层做粘性转发；本仓不实现 HTTP/LB 本体，只提供契约与存储。
- `runtime/config` 新增协调后端配置域（`runtime.coordination.*`），默认 `memory`/`file`（维持现状），Redis 必须显式开启；桥接不可用时 fail-fast（启动）与 fail-closed（运行时），不做静默降级。
- 新增 gate `scripts/check-redis-coordination-contract.*`（shell/PS parity），离线用内存 Redis 替身跑契约，有 docker Redis 时跑真实集成。

## Capabilities

### New Capabilities
- `redis-cross-instance-coordination`: Redis 作为集中式协调与消息桥接后端的接入契约，含 fencing/CAS、mailbox 语义、实例归属与 cross_machine 部署证据。

### Modified Capabilities
- `durable-source-store-coordination`: 从「cross_machine 明确 unsupported」扩展为「配置分布式协调器且有证据时可声明 cross_machine」，未配置时仍 fail-closed。
- `mailbox-store-and-delivery-contract`: 新增 Redis backend，语义与 File/Memory 对齐，不新增平行投递语义。
- `realtime-event-protocol`: 实时游标从纯进程内存扩展为可注入 `CursorStore`，默认内存实现保证行为不变。

## Impact

- 代码：
  - `core/sourcecoordination`（新增 `redis.go`，不改既有 `Memory`/`File` 语义）
  - `core/types`（`deployment_scope.go` 新增 cross-machine scope 与校验分支）
  - `orchestration/mailbox`（新增 `store_redis.go`）
  - `core/runner`（`realtime.go` 引入可注入 `CursorStore`）
  - 新增 `orchestration/instanceregistry`（实例心跳与 session affinity）
  - `runtime/config`（`runtime.coordination.*` 配置、校验、诊断）
  - `scripts/check-redis-coordination-contract.{sh,ps1}`
- 依赖：新增 `github.com/redis/go-redis/v9`（唯一新增外部依赖）。
- 兼容性：默认配置下行为与当前完全一致（memory/file、cross_machine 仍 unsupported）；Redis 必须显式开启，不可用即 fail-fast，不静默降级。
- 部署：需要可访问的 Redis（单机或集群模式），本轮只声明 standalone/sentinel 语义，Redis Cluster 分片不作为本轮验收范围。

## Example Impact Assessment

新增示例

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`, `docs/mainline-contract-test-index.md`, `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`, `docs/external-adapter-template-index.md`, `docs/adapter-migration-mapping.md`
- 维护 owner：runtime/coordination maintainers
- 验证证据：`bash scripts/check-redis-coordination-contract.sh`, `bash scripts/check-docs-consistency.sh`, `bash scripts/check-openspec-documentation-impact.sh`, `bash scripts/check-openspec-example-impact-declaration.sh`, `go test ./core/sourcecoordination/... ./orchestration/mailbox/... ./core/types/... -count=1`

## Documentation Impact Assessment

| 文档域 | 变更 | 路径 | Owner | 证据 |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runtime maintainers | 新增 coordination/instance registry 依赖方向 |
| components | 修改文档 | `docs/redis-coordination-guide.md`（新增）、`orchestration/mailbox/README.md` | coordination maintainers | Redis backend 组件契约 |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | `runtime.coordination.*` 键与默认值 |
| contract/API | 修改文档 | `docs/mainline-contract-test-index.md` | coordination maintainers | fencing/CAS/mailbox 契约与 gate 索引 |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | 协调后端诊断字段与 fail-closed 语义 |
| examples | 修改文档 | `docs/adapter-migration-mapping.md` | coordination maintainers | Redis backend 迁移映射 |
| CLI/integration | 修改文档 | `docs/mainline-contract-test-index.md`, `docs/external-adapter-template-index.md` | coordination maintainers | gate 入口与 onboarding |
| best practices | 修改文档 | `README.md` | runtime maintainers | 能力边界更新 |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap owner | L 状态与范围 |
