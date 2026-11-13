## Implementation Tasks

### Contract and deployment scope

- [x] `core/types` 新增 `CoordinatedCrossMachineScope`（`cross_machine` + `distributed` fencing + `distributed` transaction），扩展 `DeploymentScope.Validate()` 的 coordinated 分支，并补正向/负向测试（`multi_instance` 既有组合不受影响）。
- [x] 更新 `core/history/testdata/owner_store_deployment.v1.json`：保留「未配置分布式协调器 → `deployment_unsupported`」，新增「配置 Redis 协调器 → `admitted`」case，并同步 `deployment_test.go` 的覆盖断言。

### Backends

- [x] 新增 `core/sourcecoordination/redis.go`：基于 `redis.UniversalClient` 实现 `Acquire`（`SET NX PX`）、`Read`、`CompareAndSwap`（Lua 原子校验 fence + revision）、`ValidateLease`、`DeploymentScope`；错误映射复用 `ErrForeignWriter`/`ErrStaleWriter`/`ErrDeploymentConflict`/`ErrOwnerUnavailable`。
- [x] 新增 `orchestration/mailbox/store_redis.go`：实现 `Store` 全部方法（publish/consume/heartbeat/ack/nack/requeue/stats/snapshot/restore），inflight 归属与超时回收走 Lua，语义与 `FileStore` 对齐。
- [x] `core/runner` 引入 `CursorStore` 接口（Load/Save/Delete），默认 `memoryCursorStore` 保持现有 map+TTL 行为；新增 `redisCursorStore` 与 `WithCursorStore` 注入选项。
- [x] 新增 `orchestration/instanceregistry`：实例心跳注册（TTL）、活跃实例枚举、`session → instance` 归属写入/查询，实例消失返回 `ErrAffinityLost`。
- [x] 三个后端实现（Memory/File/Redis）跑同一套 table-driven 契约断言，确保 fencing、CAS、mailbox 幂等与 inflight 回收语义一致。

### Configuration, evidence, and gates

- [x] `runtime/config` 新增 `runtime.coordination.*`（backend/file_dir/redis/mailbox_backend/instance_registry），默认 `memory`，含 fail-fast 校验、热更新原子回滚与诊断字段投影。
- [x] 新增 `scripts/check-redis-coordination-contract.ps1` 与 `.sh`（shell/PowerShell parity），纳入 gate 阻断；无 `BAYMAX_TEST_REDIS_ADDR` 时真实 Redis 集成标记 `skipped-unconfigured`。
- [x] 新增 `examples/redis-cross-instance`（offline 可跑，用 miniredis 演示 token 竞争、CAS 冲突、mailbox 跨实例消费与 affinity 查询），打印稳定 marker。
- [x] 更新 `README.md`、`docs/development-roadmap.md`、`docs/mainline-contract-test-index.md`、`docs/runtime-config-diagnostics.md`、`docs/runtime-module-boundaries.md`、`docs/external-adapter-template-index.md`、`docs/adapter-migration-mapping.md`、`orchestration/mailbox/README.md`，并新增 `docs/redis-coordination-guide.md`。
- [x] 运行受影响包测试、`go test -race`、`golangci-lint`、`check-quality-gate`、`check-docs-consistency`、`openspec validate --all`、`git diff --check`，记录结果后归档。

### Gate evidence (2026-11-13)

- 受影响包 `go test`：`core/sourcecoordination`、`orchestration/mailbox`、`orchestration/instanceregistry`、`core/types`、`core/history`、`runtime/config`、`core/runner`、`orchestration/composer` 全部 ok。
- `go test -race`（上述前五个包）：全部 ok。
- `golangci-lint run --config .golangci.yml --timeout 25m`（受影响包 + 示例）：0 issues（修复 2 处 gofmt）。
- `scripts/check-redis-coordination-contract.sh`：passed（已接入 `check-quality-gate.{sh,ps1}` 阻断）。
- `scripts/check-source-history-store-contract.{sh,ps1}`：passed（fixture case 数 7→8 同步更新，并新增 cross_machine redis coordinator 断言）。
- `check-docs-consistency.sh`：Docs consistency check passed。
- 治理脚本：roadmap-status-consistency / example-impact-declaration / documentation-impact 全部 passed。
- `openspec validate redis-message-bridge-cross-instance`：valid。
- `check-quality-gate.ps1`：全部契约步骤 ok（含新增 redis coordination contract 步骤）；最后一步 `go test ./...` 撞 261s 单步预算超时（环境问题，非断言失败）。单独全量复跑：72 包 ok，2 个失败包均为既有环境问题——`cmd/host-jsonl`（干净 master 同样失败）、`tool/contributioncheck`（exec helper 在本机沙箱下 601s 包超时，该包无本地改动），与 K 归档时基线一致。

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
