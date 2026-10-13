## Why

Danny 集成方报告 Task Harness 同时承担业务计划选择与 parent/child 执行、预算、checkpoint、replan 和恢复；其 direct/subagent/team/workflow/mailbox/scheduled 模式与 Baymax 既有 owner 之间缺少可验证的接入合同。原生事件接入已由归档 160 收口，现在需要先固定宿主计划身份、运行前校验与 owner 映射，供后续恢复准入和 outbox/mailbox 接缝复用。

## What Changes

- 新增显式 opt-in 的 `host_execution_plan.v1` 接入合同和 `orchestration/planbridge` 编译/委托层，将版本化宿主描述与独立的原生执行 binding 校验后交给既有 runner/composer、workflow、teams 或 scheduler owner。
- 独立表达 execution shape（direct/subagent/team/workflow）、delivery（inline/mailbox）和 trigger（manual/scheduled/resumed/callback）。只支持已声明的 owner 路径，不能把 scheduled 当执行形态、把 mailbox accepted 当 terminal、把 resumed 标签当恢复授权。
- 固定不可变 plan identity/version/canonical digest、authority/tool capability digest、source parent correlation、输入/handler reference、预算事实与可选 checkpoint reference；replan 生成新版本，digest 只证明描述一致性。
- 提供 bounded、无副作用的全计划预校验，复用原生 DAG/teams/scheduler 验证器与 timeout resolver；校验版本/字段/引用、图环/悬空依赖、完整预算、child 限额、team aggregation、continuation capability 和 binding mismatch 后才允许执行。运行时准入仍由 source owner 在实际 activation boundary 再评估。
- 首阶段交付 direct、inline teams parallel/vote、flat workflow 和有明确 scheduler binding 的 local child 路径；subagent fanout 通过现有 workflow/teams 的组合路径表达。mailbox delivery、native resume、跨 owner durable continuation 先给出明确不支持矩阵，不自动补执行循环；合法拒绝本身属于验收证据。
- 保留 source-native 结果、事件、failure/cancel/timeout、retry 和 terminal 语义；复用归档 160 的独立 host references 和 observer。不建立新全局顺序、事件游标、终态、预算账本、重试或恢复状态机。
- 提供真实 owner + fake model/tool 的正负边界与 Run/Stream conformance、版本化离线 fixture/replay、shell/PowerShell gate、宿主中立示例及同步维护文档。

## Capabilities

### New Capabilities

- `host-execution-plan-bridge`：宿主执行计划的版本化身份、bounded 编译与 fail-fast 校验、能力支持矩阵、既有 owner 委托、source 结果/关联保真及离线 conformance。

### Modified Capabilities

无。既有 `multi-agent-lib-first-composer`、`multi-agent-composed-orchestration`、`workflow-deterministic-dsl`、`workflow-graph-composability`、`teams-collaboration-runtime`、`distributed-subagent-scheduler`、budget admission、snapshot/recovery 与 `native-execution-event-adapter` requirements 作为基线复用。只有 fixture 证明最小 source API 接缝缺失时，才暴露保持原语义的验证入口；如需改变既有 requirement，先补本 change 的对应 spec delta 与维护文档。

## Impact

预期实施范围：`orchestration/planbridge`、必要的 teams/composer 验证接缝、`integration`、`tool/diagnosticsreplay`、`tool/contributioncheck`、双平台专用 gate 与 `examples/host-execution-plan`。不依赖 Danny、Stele、AG-UI、Provider SDK 或业务数据库；不增加 runtime 配置、QueryRuns 字段、Host command/wire 扩展或后台 worker。

需求来自用户报告，未独立审计 Danny 仓库。已审计 Baymax：composer.Run/Stream 当前只委托 runner，Teams()/Workflow() 返回各自入口，DispatchChild 使用 scheduler claim/commit；workflow.ValidateDefinition 已校验 DAG，teams.normalizePlan 为包内函数，scheduler.Guardrails/timeout resolver 已承担子预算约束。提案阶段确认接入结构，实施任务 1 再用 fixture 固定必要的新增接缝，不把描述性需求当成已复现的 runtime defect。

宿主继续拥有 TaskProfile、模式选择理由、业务授权、write scope、revision、effect key、input/handler bindings 和业务 commit gate。该 additive API 必须显式启用；旧直接调用路径继续有效。桥接结果不等于业务提交，也不构成动态 action resume 或跨进程恢复保证。

## Risks and Rollback

主要风险是 plan digest 与实际 binding 脱节、校验后输入被修改、编译时默认值与 activation 时配置漂移、用预估预算替代实时 parent budget，以及组合路径把 pause/accepted 错判为 completed。采用 detached 编译快照、明确 digest 范围、source 再准入、原生结果透传和 unsupported-before-dispatch 反例约束。

停用新增 bridge 并回到既有显式 owner 调用即可回滚；保留已有 task/attempt/checkpoint/outbox 与业务记录，不删除待处理事实，不回滚已发生副作用。没有存储迁移；旧权限是否仍有效由宿主决定。

## Example Impact Assessment

新增示例

新增 `examples/host-execution-plan`：先写 owner/支持矩阵、runtime path、expected markers 和 rollback notes，再实施离线 fake model/tool 示例。本 change 不修改 `examples/agent-modes`；如范围改变，先更新 MATRIX 与对应模式 README 的文档基线。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/components/orchestration.md`、`orchestration/README.md`、`orchestration/planbridge/README.md`、`docs/runtime-module-boundaries.md`、`docs/guides/integration.md`、`docs/guides/best-practices.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`examples/host-execution-plan/README.md`
- 维护 owner：orchestration 与 host integration maintainers
- 验证证据：OpenSpec strict validation、documentation/example impact gate、roadmap/docs consistency；实施后补齐 plan bridge contract/replay、Run/Stream parity 与离线 example smoke

当前“已同步”指本 change 已在同一功能分支交付并同步的对外 API、limits/defaults、错误分类、支持矩阵、owner 边界、兼容性和回滚说明。新增维护路径已纳入三份工件的一致承诺；缺少实际测试或门禁 exit-code 证据时，仍不得勾选任务、宣称验证完成或归档。

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | orchestration maintainers | owner 边界审计与 docs consistency；禁止第二执行/预算/恢复状态机 |
| components | 修改文档 | `docs/components/orchestration.md`、`orchestration/README.md`；实施新增 `orchestration/planbridge/README.md` | orchestration maintainers | 支持矩阵与实际 owner API 对照、package tests |
| configuration | 修改文档 | 实施同步 `docs/runtime-config-diagnostics.md` | runtime config maintainers | 明确 constructor-only limits、既有配置优先级和无新增 env/file/hot-update 键 |
| contract/API | 修改文档 | 本 change spec；实施同步 `docs/mainline-contract-test-index.md` | contract maintainers | strict validation、逐 requirement 的单测/integration/replay/gate 证据；gate 接入后才更新主线索引 |
| diagnostics | 修改文档 | 实施同步 `docs/runtime-config-diagnostics.md` | observability maintainers | reference-only 错误/结果与 RuntimeRecorder 单写；不新增 QueryRuns 字段 |
| examples | 新增文档 | `examples/host-execution-plan/README.md`、`README.md` | examples maintainers | 文档基线先行，离线 smoke 与 expected markers；不依赖 live provider |
| CLI/integration | 修改文档 | `docs/guides/integration.md`；实施新增 `orchestration/planbridge/README.md` | host integration maintainers | plan/binding 编译、执行前拒绝、源结果透传与回滚示例 |
| best practices | 修改文档 | 实施同步 `docs/guides/best-practices.md` | documentation maintainers | digest 不等于授权、投递不等于完成、replay 不等于 resume 的反例核对 |
| roadmap | 修改文档 | `docs/development-roadmap.md`、`README.md` | roadmap maintainers | openspec list 与 roadmap status consistency；候选移入进行中 |

## Validation Strategy

提案阶段：`openspec validate establish-host-execution-plan-bridge-contract --strict`、documentation/example impact gate、roadmap/docs consistency 与 `git diff --check`。

实施阶段：`go test ./orchestration/... ./integration/... ./tool/diagnosticsreplay ./tool/contributioncheck -count=1`，专用 `scripts/check-host-execution-plan-contract.sh/.ps1`，以及 `go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`、quality/docs gate。完整命令与实际结果写入 verification.md；本提案不提前宣称实现测试通过。
