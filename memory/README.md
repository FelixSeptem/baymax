# memory 组件说明

## 功能域

`memory` 提供运行时 memory 统一抽象，覆盖以下能力：

- canonical SPI：`Query` / `Upsert` / `Delete`
- 统一错误模型：`memory.Error`（`operation/code/layer/message`）
- 运行模式治理：`external_spi` 与 `builtin_filesystem`
- fallback 策略：`fail_fast`、`degrade_to_builtin`、`degrade_without_memory`
- 主流 profile pack：`mem0`、`zep`、`openviking`、`generic`
- 内置文件系统引擎：WAL + snapshot + compaction + crash-safe recovery

## 架构设计

核心入口是 `Facade`：

- 对外暴露统一 `Engine` 接口，屏蔽 provider 差异。
- 在 `external_spi` 下通过 `ExternalEngineFactory` 注入外部引擎。
- 在 `degrade_to_builtin` 下按需初始化 builtin 引擎作为兜底目标。
- 在每次响应中补齐 canonical 元数据：`mode/provider/profile/contract_version/fallback_*`。

内置 `FilesystemEngine` 采用确定性持久化路径：

- 追加写 WAL（`memory.wal.jsonl`）
- 周期 compaction（`memory.snapshot.json`）
- 原子替换（`snapshot.next -> snapshot`，带 `snapshot.bak` 恢复路径）
- 进程重启后通过 snapshot + WAL tail 重放恢复状态

## 关键入口

- `memory/spi.go`：SPI、请求/响应结构、reason code 与错误层定义
- `memory/facade.go`：模式选择、配置校验、fallback 协调、响应装饰
- `memory/filesystem_engine.go`：内置引擎实现（WAL/snapshot/compaction/recovery）
- `memory/profile_pack.go`：profile 解析与 capability 基线

## 边界与依赖

- 本包只定义并实现 memory contract，不直接依赖 `runtime/diagnostics` 存储路径。
- 本包不感知具体业务域（如 context stage2、scheduler）；调用方负责命名空间和生命周期编排。
- 外部 provider 适配必须在进入 Facade 前后做 canonical 归一，不得将 provider-specific 错误形态直接透传给上层。
- contract version 当前固定 `memory.v1`；非该版本应 fail-fast。

## 配置与默认值

`Facade` 的配置模型与 `runtime.memory.*` 对齐：

- `mode`：默认 `builtin_filesystem`
- `external.profile`：默认 `generic`
- `external.contract_version`：默认 `memory.v1`
- `fallback.policy`：默认 `fail_fast`
- `builtin.compaction.min_ops`：默认 `32`
- `builtin.compaction.max_wal_bytes`：默认 `4 << 20`（4MiB）
- `scope.default`：默认 `session`（允许 `session|project|global`）
- `scope.allowed`：默认 `[session, project, global]`
- `write_mode.mode`：默认 `automatic`（`automatic|agentic`）
- `write_mode.automatic_window/agentic_window/idempotency_window`：默认 `30m/2h/24h`
- `injection_budget.max_records/max_bytes/truncate_policy`：默认 `8/16384/score_then_recency`
- `lifecycle.retention_days/ttl_enabled/ttl/forget_scope_allow`：默认 `30/false/168h/[session,project,global]`
- `search.hybrid.enabled/keyword_weight/vector_weight`：默认 `true/0.6/0.4`
- `search.rerank.enabled/max_candidates`：默认 `false/32`
- `search.temporal_decay.enabled/half_life/max_boost_rate`：默认 `false/168h/0.2`
- `search.index_update_policy/drift_recovery_policy`：默认 `incremental/incremental_then_full`

约束：

- `mode=external_spi` 时，`external.provider` 必填。
- `mode=builtin_filesystem` 或 `fallback.policy=degrade_to_builtin` 时，`builtin.root_dir` 必填。
- `fallback.policy` 仅允许：`fail_fast|degrade_to_builtin|degrade_without_memory`。

最小示例：

```go
facade, err := memory.NewFacade(memory.Config{
	Mode: memory.ModeBuiltinFilesystem,
	Builtin: memory.BuiltinConfig{
		RootDir: ".baymax/memory-store",
		Compaction: memory.FilesystemCompactionConfig{
			Enabled: true,
		},
	},
	Fallback: memory.FallbackConfig{
		Policy: memory.FallbackPolicyFailFast,
	},
}, nil)
if err != nil {
	// fail-fast
}
defer func() { _ = facade.Close() }()
```

## 可观测性与验证

SPI 响应统一携带可观测字段：

- `reason_code`
- `mode`
- `provider`
- `profile`
- `contract_version`
- `fallback_used`
- `fallback_reason_code`
- `memory_scope_selected`
- `memory_budget_used`
- `memory_hits`
- `memory_rerank_stats`
- `memory_lifecycle_action`

建议最小验证命令：

```bash
go test ./memory -count=1
go test ./memory -run 'Test(Facade|FilesystemEngine)' -count=1
pwsh -File scripts/check-memory-contract-conformance.ps1
pwsh -File scripts/check-memory-scope-and-search-contract.ps1
```

## 扩展点与常见误用

扩展点：

- 新增外部 provider：实现 `Engine` 并通过 `ExternalEngineFactory` 注入。
- 新增 profile：在 `profile_pack.go` 声明 `required/optional ops` 与 contract version。
- 新增 builtin 存储后端：保持 SPI 语义与 reason taxonomy 不变。

常见误用：

- 直接透传外部 SDK 错误，未归一为 `memory.Error`。
- 仅实现 `Query`，缺失 `Upsert/Delete` 导致 profile contract 漂移。
- 配置 `degrade_to_builtin` 却未提供可用 `root_dir`。
- 在调用方自行拼接 fallback 逻辑，绕过 Facade 导致行为不一致。
