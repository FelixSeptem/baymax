## 1. 现状回归与文档基线

- [x] 1.1 固定原生 Event、MapEventToProtocol 与 Realtime binding 的现状回归测试，验证共享 TraceID、fallback identity、sequence=0、input_required pause 和既有 source 范围，不修改既有 mapper 语义。
- [x] 1.2 在集成指南、模块边界与新增 host/eventadapter README 中交付 source-owner/能力矩阵、observer/sink 合同与回滚说明，验证每个声明可追溯到既有 owner 或本 spec。
- [x] 1.3 核对 runner、workflow、teams、scheduler、composer、A2A 与 timeline 的实际映射入口和 parent-child references，交付支持/不支持接线矩阵及 source mismatch 负向用例；缺失能力明确报告，不按事件名称猜 source。

## 2. 版本化对象与纯投影

- [x] 2.1 新增 core/types 的 native_execution_event.v1 对象与固定 source capability profiles，复用 EventEnvelope/native metadata；单测验证版本、可选字段缺省、expected Run/Session 冲突和 unsupported durable 请求执行前拒绝。
- [x] 2.2 实现固定 allowlist 的独立 HostReferences 与构造参数校验，单测覆盖未知字段/类型、非法 UTF-8、有限正值限制、精确边界和超限，验证默认 1024-byte scalar、16 citations、8-KiB references、64-KiB event。
- [x] 2.3 实现有界 JSON-compatible 深快照与直接事件纯投影，单测验证嵌套 map/list 和 citation 修改不污染输出、序列化失败无部分交付、源输入不变、共享 mapped ID 与重复回调均保留。
- [x] 2.4 实现独立的 Realtime source-normalized binding 纯投影入口，单测覆盖 overlap、gap、expired cursor、subscription correlation 与非 Realtime 拒绝，验证不新增订阅 worker、历史存储或跨批去重缓存。

## 3. Observer 与宿主 sink 接缝

- [x] 3.1 新增 opt-in host/eventadapter EventHandler 和 context-aware error-returning sink，单测验证串行回调顺序、默认 1 秒 deadline、有限正值 override，以及无队列、自动 retry 或每事件 detached goroutine。
- [x] 3.2 实现固定大小 Status 与首错停止交付，单测覆盖 projection error、sink error、合作式 timeout、写入后 error 的 unconfirmed 分类及后续回调计数，验证不保存正文或无界错误列表。
- [x] 3.3 实现同一 observer 并发/重入 fail-fast、幂等 Close 与独立 observer context，单测和 race 验证状态一致、Close 后无 sink 调用、Run 取消仍可交付 canceled；不合作 sink 的限制在 API 文档中明示。
- [x] 3.4 提供与 RuntimeRecorder 独立 fan-out 的接线示例/测试，验证 observer failure 不反馈事件循环、不调用 Cancel/Resume/Retry，任何诊断仅通过既有 RuntimeRecorder 单写入口且无新 QueryRuns 字段。

## 4. 真实执行与生命周期集成

- [x] 4.1 用真实 runner 和显式 capability adapter fake model/tool 验证 Run/Stream 模型、工具、completed、model failure、工具失败后继续与 cancel/timeout 路径；断言 sink 保留真实 correlation 且不从 RunResult 合成 parent/terminal。
- [x] 4.2 集成覆盖动态 action prepare/pause、confirm、deny、timeout 与相同 idempotency duplicate resume；断言 input_required 非 terminal、confirm 一次 resolution 后 source completion、deny/timeout 一次 resolution 和 canceled，duplicate 无补发。
- [x] 4.3 通过既有组合执行 owner 覆盖 parent-child 交错事件和独立 observer，断言 child Run 与已有 causation 未被覆盖，没有跨 source 全局 sequence 或第二终态状态机。
- [x] 4.4 覆盖 terminal 投影/交付失败与观察断开，验证 source completed 不被改写、Status 暴露 incomplete delivery、缺失终态不合成 completed，业务 commit gate 仍归宿主。
- [x] 4.5 建立 Run/Stream parity 比较，验证只归一化生成 IDs、时间及 model delta 分片，并用 terminal、resolution cardinality、source correlation、failure class 和 delivery outcome drift 反例证明比较会失败。

## 5. 离线 fixture、replay 与治理 gate

- [x] 5.1 新增版本化 fake-fact fixtures，覆盖成功投影、共享 identity 不丢事件、unsupported cursor、source conflict、引用/事件超限、pause-as-terminal、observer failure 和 Realtime binding 边界；验证 fixtures 不包含真实业务正文或 credential。
- [x] 5.2 将纯离线投影/conformance replay 接入 tool/diagnosticsreplay，正负测试验证稳定分类与 parity drift，证明不调用 model/tool/network/application sink 或业务写入。
- [x] 5.3 新增语义一致的 shell/PowerShell 专用 contract gate 并接入 quality gate，验证正向通过、负向 fixture 失败及禁止新增 execution owner、global dedup/order queue、terminal FSM、直接诊断 writer 的边界。
- [x] 5.4 更新 docs/mainline-contract-test-index.md 的正向/负向/integration/replay/gate 映射，通过索引与脚本检查确认每项 spec requirement 都有可定位验证证据。

## 6. 示例与完整文档交付

- [x] 6.1 先完成 examples/native-execution-events README 的 source path、semantic anchors、能力矩阵、expected markers 和 rollback，再提供真实 runner + fake model/tool + 内存 sink 示例；离线 smoke 验证 Run/Stream、pause 和 Status markers。
- [x] 6.2 同步 architecture、components、configuration、diagnostics、integration 和 best-practices 文档，逐项对照下方 Documentation Impact Assessment 验证 owner、limits、constructor-only 配置与无 direct TraceID dedup 约定。
- [x] 6.3 更新 README 和 roadmap 的实际交付状态及示例入口，运行 roadmap/docs consistency 确认与 openspec list 一致；如范围涉及 agent-modes，先完成 MATRIX 与对应 README 文档基线后再改代码。
- [x] 6.4 逐项审计新增/变更的对外 API、运行时行为、事件顺序/identity/终态、错误反馈、配置和诊断语义（含 additive/opt-in），建立“语义变化—维护文档—测试/gate”对应证据，同一变更与 PR 同步最终语义、默认值/限制、兼容性、迁移和回滚；通过 docs consistency、相关 contract/replay gate 与示例核对后才勾选对应任务及进入交付/归档。

## 7. 验证、归档与分支集成

- [x] 7.1 执行受影响包正向/负向/边界与 integration/replay/example smoke、双平台专用 gate，记录命令、输出和平台未执行项，验证源码与 spec delta 对齐。
- [x] 7.2 执行 go test ./...、go test -race ./...、golangci-lint run --config .golangci.yml、pwsh -File scripts/check-quality-gate.ps1 和 pwsh -File scripts/check-docs-consistency.ps1，记录全部结果后才标记实施完成。
- [x] 7.3 执行 OpenSpec strict validation 与 git diff --check，审阅 additive API、source ownership、风险和回滚点；确认所有实施任务具备代码/测试/文档/门禁证据。
- [x] 7.4 实施完成并满足归档条件后仅使用 scripts/openspec-archive-seq.ps1 -ChangeName establish-native-execution-event-adapter-contract 归档，同步 roadmap、README 与归档索引并复验状态一致。
- [x] 7.5 将已验证并归档的功能分支提交、合并到最新 master、推送并验证合并结果，再删除已合并本地分支及关联 worktree；用 git 状态与远端 commit 确认，不删除未跟踪缓存/运行产物。

## Example Impact Assessment

新增示例

新增宿主中立的 examples/native-execution-events，使用 fake model/tool，无 live provider 或外部副作用；包含真实事件、Run/Stream parity、input_required 非终态与失败反馈。当前不修改 examples/agent-modes；范围变更时先满足 MATRIX/README 文档先行要求。

## Documentation Impact Assessment

涉及对外语义变化，必须同步修改并维护受影响文档，不能仅补提案中的影响声明。影响范围变化时同步更新 proposal/design/spec/tasks 的评估与下表；维护文档未同步或一致性验证未通过，对应实施任务不得勾选，变更不得视为可交付或可归档。

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

实施与对外语义审计证据见 [verification.md](verification.md)。任务仅按已经完成的代码、测试、维护文档与门禁证据勾选；完整门禁、归档与主线集成分别记录，不提前宣称完成。
