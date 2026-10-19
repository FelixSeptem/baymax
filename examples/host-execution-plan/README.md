# Host Execution Plan Bridge

这是 `host_execution_plan.v1` 的离线 opt-in 示例。运行 `go run ./examples/host-execution-plan`；它使用 fake model/tool 和真实 runner，不访问 Provider、业务数据库或网络。示例证明 compile 无副作用、source owner 保留、Run/Stream 等价及预检拒绝；Spec 1 还规定 manifest-approved `mailbox_async` 只能委托既有 mailbox owner，不能构成业务 commit、durable resume 或 mailbox worker。

## Semantic anchors

- `host_execution_plan.v1`：版本化身份、bounded descriptor/native binding、一致性 digest 和全计划预检。
- execution shape、delivery、trigger 三个独立维度；触发标签不提供业务授权或 scheduler worker。
- source-native execution/admission/budget/retry/aggregation/terminal/checkpoint；RuntimeRecorder 单写。
- 真实事件沿用 `host/eventadapter`；plan version/digest 属 host references，不改变 source identity/顺序。
- digest 不等于授权，native terminal 不等于业务 commit，event replay 不等于 execution resume。

## Runtime path

宿主授权 input/handler/tool 与业务写范围 → descriptor + 独立 native binding → `orchestration/planbridge` Compile（无副作用）→ actual source activation/admission → native result + actual event observer → 宿主 execution record/commit gate。

direct 使用真实 runner/composer（composer.Run/Stream 目前只委托 runner）；team 使用原生 teams parallel/vote；flat workflow 使用原生 validator/dispatch adapter；local child 使用 composer/scheduler 独占隔离 binding。fake model 通过显式 model capability adapter 满足 Stream preflight，fake tool/handler 与内存 source store 使示例无网络、无真实 Provider、无业务数据库依赖。

## 支持矩阵（首版）

| 请求 | 预期 | 限制 |
| --- | --- | --- |
| direct + inline + manual/scheduled/callback | 原生 Run/Stream | trigger 是来源标签；input_required 保留 pause |
| team parallel/vote + inline | 原生执行与 aggregation | terminal-only task binding；无组合动态 continuation |
| flat workflow + inline | 原生 DAG 执行与 retry/timeout | 不桥接 subgraph/native checkpoint resume |
| local child + inline | 隔离独占 scheduler 原生 dispatch | 完整 child/parent budget；同步 handler 不自动提供 token streaming |
| subagent fanout | 经 native teams/workflow 组合 | bridge 不建立 fanout loop |
| shared-queue child dispatch | 编译拒绝 | 不 enqueue/claim，无关 task 不变 |
| mailbox_async + compatible manifest/binding | 委托既有 mailbox owner | source-owned message/delivery/lease/retry/DLQ；不创建 worker 或 parent Run |
| mailbox/resumed/native resume/durable continuation without declared capability | 编译/准入拒绝 | accepted 不等于 completed；相同 run ID 不证明恢复 |

limits 默认是 descriptor 64 KiB、合计 structural items 128、dependencies 1024、单 reference 1024 UTF-8 bytes、depth 8；copied binding 独立 64 KiB。零 override 选默认，正值只能收紧，负值/放宽拒绝。

## Expected markers

markers 为示例本地验证输出，不是新 runtime 事件或 terminal state：

- `plan-compiled`：version/digest 存在，模型/工具/store 尚未执行或改变。
- `source-owner-preserved`：真实 owner 返回 source-native outcome/correlation/budget facts。
- `run-stream-parity`：等价场景 normalized 结果一致，不要求分片数相同。
- `invalid-plan-rejected`：非法后续节点或 binding mismatch 阻止全部 dispatch。
- `unsupported-capability-rejected`：mailbox/resume/shared-queue/组合 pause 请求在 activation 前拒绝。
- `no-side-effects-before-rejection`：model/tool/handler/enqueue/publish/claim/commit 计数与真实 store 状态证明拒绝边界。
- `host-commit-required`：source result/observer 状态没有自动提交业务结果。
- `mailbox-logical-admission`：同一 full logical binding 的 Run/Stream dispatch 只保留一个 mailbox message admission。
- `mailbox-lifecycle-source-owned`：delivery/lease/retry/DLQ 与 terminal evidence 来自 mailbox owner，不是 bridge synthetic state。

全部 marker 都来自前置断言后的真实离线验证；`scripts/check-host-execution-plan-contract.sh/.ps1` 同时校验它们与故意 drift fixture。mailbox 的 owner lifecycle、Run/Stream parity 与 reference-only fixture 由 `scripts/check-mailbox-plan-bridge-contract.sh/.ps1` 单独覆盖；两者都不会访问 Provider、业务数据库或网络。

## Rollback notes

停用 opt-in bridge，回到原 owner 显式 API。保留 task/attempt/checkpoint/outbox 与业务记录，不删除待处理事实；已经发生的写入由宿主 effect/commit 规则协调。无需存储迁移，也不变更原 owner 默认配置。

## 文档与验证责任

owner：orchestration 与 host integration/examples maintainers。对外 API/限制/错误/支持矩阵变化必须同一 OpenSpec 同步本 README、package README、集成指南、模块边界和 roadmap；未同步或 gate 未通过不能完成任务或归档。

验证入口：`go test ./orchestration/planbridge -count=1`、真实 owner integration、reference-only fixture/replay、`scripts/check-host-execution-plan-contract.sh/.ps1`、离线 example smoke，以及 AGENTS.md 完整代码门禁。实现与文档在同一 OpenSpec change 维护；归档只在这些要求具有实际 exit-code 证据后进行。
