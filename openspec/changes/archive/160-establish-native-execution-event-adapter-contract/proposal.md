## Why

Danny 集成方报告当前通过 `engine.Stream(ctx, request, nil)` 执行，再自行构造 parent/terminal ExecutionEvent；这样无法向应用 execution record 或后续 workbench 提供真实执行过程。Baymax 已有原生 EventHandler、Protocol event mapping 与 Host delivery 基线，需要现在提供可直接嵌入的事件接入 contract，防止宿主重复定义顺序、去重和终态语义。

## What Changes

- 新增 opt-in、transport-neutral 的 native execution event adapter，接收实际 Run/Stream 或 source timeline 回调，复用现有 EventEnvelope 映射并保留 native type/version、iteration、trace/span 及可用 source correlation。
- 新增 bounded、reference-only 的宿主关联信息；task、occurrence、plan、projection、knowledge generation、citation 等标识置于独立 host references，不能覆盖 source Run/Session/Step 或授权事实。
- 显式声明 direct callback 的能力限制：没有 durable history/cursor、没有可据以去重的唯一事件 identity、没有跨 source 全局顺序；Realtime 能力只能来自现有 source-owned projection。
- 提供 context-aware、有限等待的 sink 接入与可查询的 bounded 失败结果。EventHandler 无 error 返回，观察失败必须可见，不能隐式 cancel/retry/resume 或伪造 terminal。
- 用真实 runner + fake model/tool 覆盖 Run/Stream、parent-child、工具错误、dynamic action pause/resolution/resume、cancel/timeout、重复与断连边界；增加离线 fixture/replay、双平台 gate 和宿主中立示例。
- 完成集成与责任归属文档，移除 roadmap 中本候选并标记为进行中；其余四项保持未启动。

## Capabilities

### New Capabilities

- `native-execution-event-adapter`: 将真实执行事件投影为带 native metadata、bounded host references 和 source capability 声明的版本化接入对象，提供有界交付失败反馈及离线 conformance。

### Modified Capabilities

无。现有 `agent-runtime-protocol-contract`、`embedded-host-command-response-and-event-correlation-contract`、`durable-runtime-event-stream-binding`、动态 action 与终态 contract 均作为基线复用；不修改原 mapper event ID 规则、Host wire protocol、Realtime 游标或 runner 终态语义。

## Impact

预期代码范围：新增 `core/types/native_execution_event.go` 与 `host/eventadapter`，以及测试、`tool/diagnosticsreplay` 的离线支持、`integration` fixture/gate、`examples/native-execution-events`。不依赖 Danny/AG-UI，不增加 runtime 配置、QueryRuns 字段、Provider SDK 或外部服务。

该 API 是显式 opt-in 的 additive surface；旧 EventHandler、EventEnvelope、Host JSONL、Run/Stream 与 RuntimeRecorder 接入保持有效。Danny 的 TaskProfile、业务授权、write scope、revision、effect key、持久化与 commit gate 仍由宿主维护。引用和事件接入能力不构成执行权限或业务提交证据。

需求线索来自用户报告，未独立审计 Danny 仓库。Baymax 代码审计确认：`types.Event` 没有独立 event ID/sequence，`MapEventToProtocol` 优先使用 TraceID，且只浅拷贝 payload；`host.Connection.emitEvent` 已复用 mapper；durable binding 当前仅验证 Realtime source。首项任务把这些边界钉成可重复测试，不将 direct callback 宣称为 durable log。

## Risks and Rollback

主要风险是用共享 TraceID 去重丢事件、把 input_required 当 terminal、sink 无限阻塞、host references 越权覆盖 source facts，以及异步消费可变 payload。设计以保守能力声明、有界快照、合作式 context sink、失败结果和反例测试约束。

回滚停用新增 adapter，恢复原有显式 EventHandler/Host 路径；不删除业务 execution record、不迁移存储、不改变已发生副作用，不自动降级到 fabricated terminal。

## Example Impact Assessment

新增示例

新增 `examples/native-execution-events`，仅用 fake model/tool 与内存 sink，展示真实事件、Run/Stream parity、input_required 非 terminal 和失败反馈。本 change 不修改 `examples/agent-modes`；若实施确需修改，先补 MATRIX 与对应 README 文档基线后再改代码。

## Documentation Impact Assessment

对外语义变化必须同步修改并维护对应文档。新增或改变公开 API/contract、运行时行为、事件顺序/identity/终态、错误与失败反馈、配置默认值/校验或诊断字段时，必须在同一变更与 PR 中同步 OpenSpec、代码、测试及受影响的维护文档；additive 和 opt-in 入口同样适用。文档应说明实际语义、默认值与限制、兼容性、迁移和回滚方式，并更新对应 README、集成指南、组件/架构、配置诊断、contract index 与示例说明。影响范围变更时重新评估下表及 Example Impact Assessment；文档未同步或一致性验证未通过，不得勾选对应任务或宣称可交付、可归档。

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/architecture/data-event-flow.md`, `docs/runtime-module-boundaries.md` | runtime/host owners | source-owner 边界评审与 docs consistency |
| components | 修改文档 | `core/types/README.md`, `observability/README.md`, 新增 `host/eventadapter/README.md` | core/types 与 host owners | API/example 对照和 package tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime config owner | 明确仅构造参数、无 env/file/hot-update 新键；docs gate |
| contract/API | 修改文档 | 新 capability spec, `docs/mainline-contract-test-index.md` | contract owner | OpenSpec strict validation 与 contract/replay gate |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md`, `observability/README.md` | observability owner | 失败结果和 recorder 单写接线测试；无 QueryRuns 新字段 |
| examples | 新增文档 | `examples/native-execution-events/README.md`, `README.md` 示例入口 | examples owner | 离线 example smoke 与 expected markers |
| CLI/integration | 修改文档 | `docs/guides/integration.md`, `host/eventadapter/README.md` | host/integration owner | handler/sink/failure/capability 接入测试 |
| best practices | 修改文档 | `docs/guides/best-practices.md` | documentation owner | 禁止 fabricated terminal、TraceID 去重与无限背压；docs gate |
| roadmap | 修改文档 | `docs/development-roadmap.md`, `README.md` | roadmap owner | openspec list 与 roadmap status consistency |

## Validation Strategy

提案阶段执行 OpenSpec strict validation、example/documentation impact gate、roadmap/doc consistency 与 git diff 检查。实施阶段执行受影响包正负边界测试、integration/replay、专用 shell/PowerShell contract gate，及 AGENTS.md 的全量 Go test、race、lint、quality/docs gate。提案完成不表示这些实现测试已经执行。
