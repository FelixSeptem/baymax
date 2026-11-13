## Design

### Owner and scope

`core/sourcecoordination` 拥有 source-owner fencing/CAS 后端；`orchestration/mailbox` 拥有消息投递后端；新增的 `orchestration/instanceregistry` 拥有实例心跳与 session affinity 记录；`core/runner` 只消费 `CursorStore` 接口，不感知 Redis。`runtime/config` 是唯一开关来源，负责 fail-fast 校验与诊断投影。

既有语义不重写：`Coordinator`、`mailbox.Store`、`deployment_scope.v1` 的字段与校验保持向后兼容，Redis 只是新增实现与新增合法取值。

### Deployment scope: 让 cross_machine 有合法表达

现状 `DeploymentProfileCrossMachine` 只有常量、没有构造函数，`Validate()` 的协调分支也只接受 `multi_instance`，所以跨机器只能表达成 unsupported。

新增：

```go
func CoordinatedCrossMachineScope(owner, evidence string) DeploymentScope
// Profile=cross_machine, WriterScope=multi_writer,
// FencingScope=distributed, TransactionScope=distributed
```

`Validate()` 的 coordinated 分支扩展为 `multi_instance` 或 `cross_machine`，且 `cross_machine` 必须 `FencingScope=distributed` + `TransactionScope=distributed`（`source_owner` fencing 不足以证明跨机器）。`multi_instance` 的既有组合保持不变。

`Supported=true` 只能由「显式配置了分布式协调器且后端自证」产生：`Redis.DeploymentScope()` 在 ping 成功且租约键可读时返回 cross-machine scope，否则返回 `DefaultDeploymentScope`（unsupported，fail-closed）。

### Redis Coordinator: 租约 + CAS

Key 布局（统一前缀 `baymax:coord:<namespace>`）：

| Key | 类型 | 用途 |
| --- | --- | --- |
| `<prefix>:lease` | string | 当前 writer 的 JSON（writer_id/fence/expires_at） |
| `<prefix>:state` | hash | `revision` / `payload` |

- `Acquire(writerID, ttl)`：`SET lease <json> NX PX ttl`。成功即写 lease 并返回带随机 fence 的 `Lease`；失败说明有活跃 writer，返回 `types.ErrOwnerUnavailable`。同一 writer 重复 Acquire 允许刷新（`GET` 校验 writer_id 后 `PEXPIRE`），避免自身 TTL 抖动。
- `Read()`：`HMGET state revision payload`，无 state 时返回 revision=0 空 payload。
- `CompareAndSwap(lease, expectedRevision, payload)`：单个 Lua 脚本内完成「读 lease → 校验 namespace/writer/fence/expires → 读 revision → 比对 → 写 revision+1 与 payload」。任一校验失败返回对应错误码，脚本内不做部分写入。
- `ValidateLease(lease)`：只读 Lua 脚本，不产生副作用。
- `DeploymentScope()`：ping + 读 lease 键，成功走 cross-machine scope，失败走 unsupported。

错误映射与 `Memory`/`File` 完全一致：`ErrForeignWriter` / `ErrStaleWriter` / `ErrDeploymentConflict` / `ErrOwnerUnavailable`，不新增错误类型。

### Redis Mailbox Store: 与 FileStore 语义对齐

实现形态（实施期确定，替代早期 per-key 结构草稿）：**状态文档 + 分布式锁**。
mailbox 的全部投递语义（幂等 publish、inflight 归属、超时回收、retry backoff、
DLQ、lifecycle 事件、host-plan binding、continuation）由同一个 `mailboxState`
状态机承载，memory/file/redis 三个后端只是不同的持久化与互斥层——语义逐字节
一致是 AGENTS.md 的硬约束，用 Lua 重写状态机会引入平行终止/决策语义，故不采用。

Key 布局（前缀 `baymax:mailbox:<namespace>`）：

| Key | 类型 | 用途 |
| --- | --- | --- |
| `state` | string | 序列化的 `mailbox.Snapshot` JSON（状态文档） |
| `lock` | string | 变更互斥锁（随机 token，`SET NX PX lock_ttl`） |

- 所有变更（Publish/Consume/Heartbeat/Ack/Nack/Requeue/Restore/continuation mutate）
  在单个 Lua 获取的分布式锁内执行：`SET lock <token> NX PX lock_ttl`；
  释放走 compare-token-delete Lua，防止误删他方锁。锁竞争在 `op_timeout` 内
  重试（10ms 步进），超时返回 `ErrRedisUnavailable`（fail-closed，不降级）。
- 读路径（Stats/Snapshot/continuation snapshot）不取锁：单 key `GET` 原子，
  返回的总是某个一致 revision 的投影。
- inflight 归属与超时回收复用 `mailboxState.consume/reclaimStaleInFlight`，
  由调用方传入的 `now` 驱动，三后端行为一致（契约测试断言同一序列）。
- 运行期 Redis 不可用：所有变更返回 `ErrRedisUnavailable`，调用方 fail-closed；
  Snapshot/Restore 的原子性由「锁内 GET→mutate→SET」保证。
- 不使用 pub/sub 做投递（会丢消息），pub/sub 只用于实例注册表的变更通知（可丢，读取侧始终回源）。

### CursorStore: 把实时游标从进程内存抽出来

`core/runner` 现状直接用 `map[string]realtimeCursorRecord`。改为接口：

```go
type CursorStore interface {
    Load(ctx context.Context, sessionID string, now time.Time) (string, bool, error)
    Save(ctx context.Context, sessionID, cursor string, expiresAt time.Time) error
    Delete(ctx context.Context, sessionID string) error
}
```

默认 `memoryCursorStore` 包装原有 map 行为（含 TTL 过期清理），保证默认路径零语义变化。`redisCursorStore` 用 `SET key cursor PXAT expiresAt` / `GET`，key 前缀 `baymax:cursor`。Engine 通过 `WithCursorStore` 选项注入；注入 nil 或失败时回落到内存实现并记录诊断，不阻断 Run/Stream。

### Instance registry and session affinity

新增 `orchestration/instanceregistry`：

- `Register(instanceID, endpoint, ttl)`：`SET inst:<id> <json> PX ttl`，心跳周期 `ttl/3` 刷新。
- `List()`：`SCAN inst:*` 过滤未过期，返回活跃实例。
- `BindSession(sessionID, instanceID, ttl)`：`SET sess:<id> <instanceID> PX ttl`。
- `ResolveSession(sessionID)`：读归属，实例已消失返回 `ErrAffinityLost`，由宿主决定重放或拒绝。

本仓不含 HTTP/LB，所以这层只提供**存储与契约**：宿主的路由层据此做粘性转发。没有路由层时，`Register` 仍可运行（用于观测与诊断），affinity 查询返回 `ErrAffinityUnavailable`。

### Configuration

```yaml
runtime:
  coordination:
    backend: memory            # memory|file|redis，默认 memory
    file_dir: ""               # backend=file 时必填
    redis:
      addr: ""                 # 例如 127.0.0.1:6379
      username: ""
      password: ""
      db: 0
      dial_timeout: 2s
      op_timeout: 1s
      key_prefix: "baymax"
      lease_ttl: 30s
      inflight_timeout: 60s
    mailbox_backend: memory    # memory|file|redis，默认 memory
    instance_registry: disabled # disabled|redis
```

校验规则：

- `backend=redis` 时 `redis.addr` 必填，`lease_ttl>0`、`inflight_timeout>0`、`op_timeout>0` 且 `op_timeout < lease_ttl`。
- `backend=file` 时 `file_dir` 必填。
- `mailbox_backend=redis` 要求 `backend=redis`（避免只配消息后端却没有协调器）。
- `instance_registry=redis` 要求 `backend=redis`。
- 非法枚举、缺失必填项、超时顺序非法：启动与热更新 fail-fast + 原子回滚。
- 运行期 Redis 不可用：`Acquire`/`Publish` 返回 `ErrOwnerUnavailable` 类错误，调用方 fail-closed（不降级到 host/memory），并写诊断。

诊断字段（additive + nullable + default）：`coordination_backend`、`coordination_scope`、`coordination_redis_reachable`、`coordination_lease_ttl_ms`、`mailbox_backend`、`instance_registry_backend`、`instance_id`、`session_affinity_state`。

### Testing strategy

- 单元/契约：用 `redis.UniversalClient` 接口注入 `miniredis` 替身，覆盖租约竞争、fence 过期、CAS revision 冲突、mailbox 幂等 publish、inflight 回收、归属不匹配、snapshot/restore 原子性。
- 集成（可选）：`docker run redis` 后跑同一套契约；未配置 `BAYMAX_TEST_REDIS_ADDR` 时标记 `skipped-unconfigured`，不算失败。
- 语义等价：同一组场景在 `Memory` / `File` / `Redis` 三实现上跑同一断言（table-driven），确保不产生平行语义。
- fail-closed：Redis 断开后所有写操作返回错误，`DeploymentScope()` 变 unsupported，cross_machine 不再被声明。

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

## Risks and rollback

- 风险：Redis 成为单点。缓解：`lease_ttl`/`op_timeout` 明确、不可用即 fail-closed 并暴露诊断；不静默降级到本地状态。
- 风险：mailbox 语义漂移。缓解：三实现同套断言 + gate。
- 风险：cross_machine 被过早声明。缓解：只有 `Redis.DeploymentScope()` 在连通性验证通过时返回 supported，fixture 保留 unsupported case。
- 回滚：`runtime.coordination.backend` 改回 `memory`/`file` 即回到当前行为；Redis key 全部带统一前缀，可整批清理；不涉及数据迁移。
