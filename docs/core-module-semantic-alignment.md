# Core Module Semantic Alignment

更新时间：2026-03-20

## 目标

本文件用于给出核心模块文档与实现的语义映射，确保以下三者可追踪一致：

1. 模块 README 的职责描述
2. 关键实现入口（代码文件）
3. 代表性回归测试（单测 + 集成）

状态口径说明：
- 活跃变更：以 `openspec list --json` 为准。
- 当前在研变更：A33（A32 已完成待归档）。
- 本文件不宣称 in-progress 提案已归档完成，仅做“代码现状可核对性”映射。

## 模块映射表

| 模块 | README | 关键实现入口（代码） | 代表性测试（单测 + 集成） | 对齐结论 |
| --- | --- | --- | --- | --- |
| A2A | `a2a/README.md` | `a2a/interop.go`、`a2a/async_reporting.go` | `a2a/interop_test.go`、`a2a/async_reporting_test.go`、`integration/a2a_mcp_contract_test.go` | README 的 submit/status/result 与 async reporting 语义与代码一致 |
| Runner Core | `core/runner/README.md` | `core/runner/runner.go`、`core/runner/security.go`、`core/runner/security_delivery.go` | `core/runner/runner_test.go`、`integration/stream_golden_test.go` | Run/Stream 主循环、安全与终态收敛描述与实现一致 |
| Core Types | `core/types/README.md` | `core/types/types.go` | `core/types/types_test.go` | 纯契约层定位与代码一致（无业务执行逻辑） |
| Local Tool | `tool/local/README.md` | `tool/local/registry.go`、`tool/local/schema.go` | `tool/local/registry_test.go`、`core/runner/runner_test.go`（集成覆盖） | 注册/校验/调度叙述与实现一致 |
| MCP Runtime | `mcp/README.md` | `mcp/http/client.go`、`mcp/stdio/client.go`、`mcp/retry/retry.go`、`mcp/profile/profile.go`、`mcp/diag/diag.go` | `mcp/http/client_test.go`、`mcp/stdio/client_test.go`、`mcp/internal/reliability/retry_test.go`、`integration/mcp_transport_contract_test.go` | 传输 + 语义子域 + internal 边界描述一致 |
| Model Adapters | `model/README.md` | `model/openai/client.go`、`model/anthropic/client.go`、`model/gemini/client.go`、`model/providererror/classified.go` | `model/openai/client_test.go`、`model/anthropic/client_test.go`、`model/gemini/client_test.go`、`integration/model_multi_provider_contract_test.go` | 多 provider 统一契约与错误归类描述一致 |
| Context Assembler | `context/README.md` | `context/assembler/assembler.go`、`context/assembler/ca3.go`、`context/provider/provider.go`、`context/guard/guard.go`、`context/journal/storage.go` | `context/assembler/assembler_test.go`、`context/provider/provider_test.go`、`integration/context_assembler_external_retriever_integration_test.go` | CA1-CA4 与 provider 抽象边界描述一致 |
| Orchestration | `orchestration/README.md` | `orchestration/composer/composer.go`、`orchestration/scheduler/scheduler.go`、`orchestration/scheduler/async_reconcile.go`、`orchestration/mailbox/mailbox.go`、`orchestration/invoke/mailbox_bridge.go`、`orchestration/collab/primitives.go` | `orchestration/*/*_test.go`、`integration/composer_contract_test.go`、`integration/mailbox_contract_test.go`、`integration/async_await_reconcile_contract_test.go` | mailbox 主线、awaiting_report/reconcile、collab 语义与实现一致 |
| Adapter Contracts | `adapter/README.md` | `adapter/manifest/manifest.go`、`adapter/capability/negotiation.go`、`adapter/scaffold/scaffold.go`、`adapter/profile/profile.go` | `adapter/*/*_test.go`、`integration/adapterconformance/harness_test.go`、`integration/adaptercontractreplay/replay_test.go` | manifest/negotiation/scaffold/replay 叙述与实现一致 |
| Runtime Config | `runtime/config/README.md` | `runtime/config/config.go`、`runtime/config/manager.go` | `runtime/config/config_test.go`、`runtime/config/manager_test.go` | 配置优先级、fail-fast、热更新回滚与实现一致 |
| Runtime Diagnostics | `runtime/diagnostics/README.md` | `runtime/diagnostics/store.go` | `runtime/diagnostics/store_test.go`、`integration/unified_query_contract_test.go` | QueryRuns/QueryMailbox、趋势与幂等收敛语义一致 |
| Runtime Security | `runtime/security/README.md` | `runtime/security/redaction/redactor.go` | `runtime/security/redaction/redactor_test.go`、`integration/security_redaction_integration_test.go` | redaction 组件定位与调用侧复用语义一致 |
| Observability | `observability/README.md` | `observability/event/dispatcher.go`、`observability/event/runtime_recorder.go`、`observability/trace/trace.go` | `observability/event/runtime_recorder_test.go`、`observability/event/timeline_test.go` | RuntimeRecorder 单写入口与事件分发定位一致 |
| Skill Loader | `skill/loader/README.md` | `skill/loader/loader.go` | `skill/loader/loader_test.go` | Discover/Compile + scoring 策略描述与实现一致 |

## 本轮发现与修正

1. 状态口径漂移已修复：README 与 roadmap 已更新为“A33 在研、A32 完成待归档”，并与 `openspec list --json` 对齐。
2. `runtime/config` README 已补齐 async-await/reconcile 关键默认值，避免“代码有默认值、文档缺省”的信息缺口。
3. `runtime/diagnostics` README 已补齐 `RecentReloads`、`QueryMailbox` 默认分页和 A32 相关 additive 字段族说明。

## 维护建议

1. 模块 README 若新增“关键入口”文件，需确保文件路径真实存在且可被测试覆盖。
2. 每次 active/archived 变更切换后，先更新 `README.md` 与 `docs/development-roadmap.md` 状态，再跑 `scripts/check-docs-consistency.*`。
3. 若新增主链路能力，需同步更新：
   - `docs/mainline-contract-test-index.md`
   - 对应模块 README 的“配置与默认值”或“可观测性与验证”章节。
