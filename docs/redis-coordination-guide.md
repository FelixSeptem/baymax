# Redis Coordination Guide

跨 pod 多实例部署的协调栈：分布式租约协调器、跨实例 mailbox、实时游标共享与
实例亲和记录。默认姿态不变——没有任何 Redis 配置时，系统仍是单实例语义，
`cross_machine` 保持 unsupported（fail-closed）。

## 组件与所有权

| 组件 | 包 | 契约 | Redis 用途 |
| --- | --- | --- | --- |
| Source-owner coordinator | `core/sourcecoordination`（`Redis`） | `Coordinator` / `LeaseValidator` | 租约键 `SET NX PX` + Lua 原子 CAS |
| Mailbox backend | `orchestration/mailbox`（`RedisStore`） | `mailbox.Store` | 状态文档 + 分布式互斥锁 |
| Cursor store | `core/runner`（`redisCursorStore`） | `CursorStore` | `SET` + 服务端 TTL |
| Instance registry | `orchestration/instanceregistry`（`RedisRegistry`） | `Registry` | 心跳 TTL + affinity 键 |

分层约束：`core/runner` 只消费 `CursorStore` 接口，不感知 Redis 细节；
`runtime/config` 是唯一开关来源；宿主路由层（不在本仓内）据此做粘性转发。

## Key 布局（前缀默认 `baymax`，可配）

| Key | 组件 | 说明 |
| --- | --- | --- |
| `<prefix>:coord:<namespace>:lease` | coordinator | writer 租约 JSON（TTL 权威） |
| `<prefix>:coord:<namespace>:state` | coordinator | revision / payload hash |
| `<prefix>:mailbox:<namespace>:state` | mailbox | 序列化 `mailbox.Snapshot` |
| `<prefix>:mailbox:<namespace>:lock` | mailbox | 变更互斥锁（token + TTL） |
| `<prefix>:cursor:<session>` | cursor store | resume cursor（TTL） |
| `<prefix>:registry:inst:<id>` | registry | 实例心跳 JSON（TTL） |
| `<prefix>:registry:sess:<session>` | registry | session → instance 归属（TTL） |

## Deployment scope 语义

- `Redis.DeploymentScope()` 仅在 ping 成功且租约键可读时返回
  `cross_machine`（distributed fencing + distributed transaction，evidence
  `redis-coordination.v1`）；任何失败退化为 unsupported，缺失从不隐含更宽保证。
- Memory/File 协调器语义不变：`same_host` / `multi_instance`。
- `deployment_scope.v1` 校验扩展：`cross_machine` 必须携带 distributed
  fencing + transaction；`source_owner` fencing 不足以证明跨机器。

## 错误与 fail-closed

- Coordinator：复用 `ErrForeignWriter` / `ErrStaleWriter` /
  `ErrDeploymentConflict` / `ErrOwnerUnavailable`，不新增错误类型。
- Mailbox Redis 后端：运行期不可用统一返回 `ErrRedisUnavailable`，不降级到
  memory（跨实例部署静默变单实例比失败更危险）。
- Registry：binding 指向已死实例 → `ErrAffinityLost`（宿主决定重放或拒绝）；
  无 binding → `ErrSessionUnbound`；后端不可达 → `ErrAffinityUnavailable`。
- Cursor store：注入的分布式 store 出错时回落进程内存实现（resume cursor 是
  可丢的便利数据，不承载准入决策），Run/Stream 不中断。

## 配置

`runtime.coordination.*`（详见 `docs/runtime-config-diagnostics.md`）：

- `backend`：memory（默认）| file | redis
- `file_dir`：backend=file 时必填
- `redis.addr/username/password/db/dial_timeout_ms/op_timeout_ms/key_prefix/
  lease_ttl_ms/inflight_timeout_ms`
- `mailbox_backend`：memory（默认）| file | redis（要求 backend=redis）
- `instance_registry`：disabled（默认）| redis（要求 backend=redis）

非法配置启动与热更新均 fail-fast；manager 原子回滚到上一个合法快照。

## 测试与 gate

- 三后端共享契约：`core/sourcecoordination`（fencing/CAS）、
  `orchestration/mailbox`（幂等 publish / inflight 回收 / 归属校验 / 快照），
  Memory/File/Redis 跑同一 table-driven 断言。
- 真实 Redis 集成：设 `BAYMAX_TEST_REDIS_ADDR` 后自动启用，未配置时标记
  `skipped-unconfigured`（不算失败）。
- Gate：`scripts/check-redis-coordination-contract.{sh,ps1}`（已接入
  quality gate 阻断）。
- Offline 示例：`examples/redis-cross-instance`。

## 已知边界

- Mailbox Redis 后端吞吐低于 memory/file（每变更多一次锁往返）；跨实例正确性
  优先于吞吐，量大时按 namespace 分片。
- 实例注册表只提供存储与契约，不含 HTTP/LB/路由；affinity 消费方是宿主路由层。
- 锁与 TTL 依赖 Redis 单调时间；主从切换的极小概率双写窗口由 fence/CAS 校验兜底
  （与 coordinator 同一防线）。
