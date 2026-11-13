# redis-cross-instance

Redis 跨实例协调的 onboarding 示例：完全 offline 可跑（内置 miniredis 替身），
也可把 `BAYMAX_EXAMPLE_REDIS_ADDR` 指向真实 Redis，同一序列原样执行。

## 演示内容

1. **Coordinator 租约与 CAS**（`core/sourcecoordination.Redis`）
   - 两个 writer 竞争同一个 source-owner 租约：后者被 `ErrOwnerUnavailable` 阻塞。
   - CAS revision 冲突返回 `ErrDeploymentConflict`。
   - `DeploymentScope` 在 Redis 可达且租约键可读时上报
     `profile=cross_machine fencing=distributed transaction=distributed`，否则 fail-closed。
2. **Mailbox 跨实例投递**（`orchestration/mailbox.RedisStore`）
   - 实例 A publish，实例 B consume + ack；A 从共享 keyspace 读到 B 的 ack 统计。
   - 语义与 memory/file 后端逐字节一致（同一 `mailboxState` 状态机，仅换持久化与互斥层）。
3. **实例注册与 session affinity**（`orchestration/instanceregistry.RedisRegistry`）
   - 心跳注册 + TTL；`BindSession`/`ResolveSession`。
   - 优雅下线（`Unregister`）释放 binding → `ErrSessionUnbound`；
     崩溃路径（TTL 过期）→ `ErrAffinityLost`，由宿主决定重放或拒绝。

## 运行

```bash
go run ./examples/redis-cross-instance
```

## 稳定 marker

`coordinator-cross-machine-scope`、`coordinator-lease-contended`、
`cas-revision-conflict`、`mailbox-cross-instance-delivery`、
`mailbox-cross-instance-ack-visible`、`affinity-bound`、
`affinity-released-on-graceful-leave`。

## 配置入口

`runtime.coordination.*`（见 `docs/runtime-config-diagnostics.md`）：
`backend`（memory|file|redis）、`redis.*`、`mailbox_backend`、`instance_registry`。
默认姿态不变：全部 memory/file，未配置 Redis 时 `cross_machine` 保持 unsupported。
