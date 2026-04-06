## 1. Baseline and Governance Setup

- [ ] 1.1 固化 A64-S1~S10 模块映射与负责人，补充到 change 内索引说明。
- [ ] 1.2 采集当前基线（`ns/op`、`allocs/op`、`B/op`）并生成可复现记录。
- [ ] 1.3 定义 A64 回归阈值与基线更新流程（含触发条件与审批口径）。
- [ ] 1.4 明确各 S 子项默认开关、回滚开关与失败回滚策略。

## 2. S1 Context Assembler and Stage2 Hotpath

- [ ] 2.1 优化 `sanitizeRecap` 脱敏路径，去除多次 `json marshal/unmarshal` 往返。
- [ ] 2.2 为 `context/journal` 与 CA3 spill backend 增加句柄复用/批量 flush 可选路径。
- [ ] 2.3 优化 stage2 file/external provider 编解码与缓冲复用路径。
- [ ] 2.4 增加并接入 `BenchmarkContextAssemblerLoop*`、`BenchmarkCA3Stage2Pass*`、`BenchmarkStage2FileProvider*` 基准与受影响 contract/replay 回归。
- [ ] 2.5 为 `prefixCache/ca3State` 增加 run-finished 清理与 TTL/LRU 上限治理。
- [ ] 2.6 增加 CA3 stage2“无增量跳过”开关，并补签名不变/语义不变测试。
- [ ] 2.7 接入 `check-context-production-hardening-benchmark-regression.sh/.ps1` 并将其纳入 A64 性能回归主门禁子步骤。

## 3. S2 Diagnostics and RuntimeRecorder Hotpath

- [ ] 3.1 将 diagnostics 查询路径改为“锁内快照、锁外过滤/排序/聚合”。
- [ ] 3.2 优化 percentile/trend 聚合，降低重复 copy+sort 开销。
- [ ] 3.3 优化 `run.finished` 映射构建，减少大对象重复分配。
- [ ] 3.4 增加并接入 `BenchmarkRuntimeRecorderRunFinished*`、`BenchmarkDiagnosticsQueryRuns*`、`BenchmarkDiagnosticsMailboxAggregates*` 基准与 diagnostics query 回归 gate。
- [ ] 3.5 接入 inferential feedback advisory 聚合路径（复用 `runtime.eval.*` 与运行态质量信号），并保证 readiness/admission deny 语义不变。
- [ ] 3.6 增加 inferential feedback replay fixture 与 drift 分类回归，验证 Run/Stream/replay 等价。
- [ ] 3.7 接入 `check-diagnostics-query-performance-regression.sh/.ps1` 并将其纳入 A64 性能回归主门禁子步骤。

## 4. S3 Scheduler/Mailbox/Recovery File Persistence

- [ ] 4.1 为 scheduler/mailbox/composer file store 引入 debounce/group-commit 可选路径。
- [ ] 4.2 明确 flush 边界与 crash recovery 一致性断言并补测试。
- [ ] 4.3 为 task-board/mailbox query 增加增量索引/缓存策略。
- [ ] 4.4 增加并接入 `BenchmarkSchedulerFileStorePersist*`、`BenchmarkMailboxFileStorePersist*` 基准与 multi-agent shared contract 回归。
- [ ] 4.5 引入持久化节流/批次参数并增加 fail-fast 校验。
- [ ] 4.6 为 S3 参数热更新补回滚测试，确保非法更新原子回滚。
- [ ] 4.7 为 realtime interrupt/resume cursor 与 isolate-handoff 关键状态补齐可恢复持久化边界（复用 unified snapshot 合同扩展，不引入第二套事实源）。
- [ ] 4.8 增加 interaction-state crash/restart/replay 一致性回归，确保 A66/A67-CTX/A68 语义不漂移。
- [ ] 4.9 增加 multi-agent 并发/交错/重试/重放矩阵回归，固化 scheduler/mailbox/composer 场景的 emergent drift 分类与阻断断言。
- [ ] 4.10 接入 `check-multi-agent-performance-regression.sh/.ps1` 并将其纳入 A64 性能回归主门禁子步骤。

## 5. S4 MCP Transport and Diagnostics Store

- [ ] 5.1 收敛 stdio/http `invokeAsync` 每调用 goroutine 包装层。
- [ ] 5.2 优化 MCP 事件发射与短生命周期对象分配。
- [ ] 5.3 为 `mcp/diag.Store` 引入 ring-buffer，替代 overflow 切片整体复制。
- [ ] 5.4 增加并接入 `BenchmarkMCPInvokePath*`、`BenchmarkMCPEventEmit*` 基准并执行 transport + shared contract 回归。

## 6. S5 Skill Loader Discover/Compile/Scoring

- [ ] 6.1 引入基于 `path + mtime + size` 的 discover/compile 元数据缓存。
- [ ] 6.2 优化 tokenization/关键词优先级排序路径，加入预编译缓存。
- [ ] 6.3 保持 `agents.md|folder|hybrid` 结果一致并补 parity 测试。
- [ ] 6.4 增加并接入 `BenchmarkSkillLoaderDiscover*`、`BenchmarkSkillLoaderCompile*`、`BenchmarkSkillSelectionScore*` 基准与 `skill/loader` 相关回归。

## 7. S6 Memory Filesystem Engine

- [ ] 7.1 拆分 query 读路径与 TTL 写路径，默认查询不触发写锁。
- [ ] 7.2 优化 snapshot/index 持久化为增量/流式编码路径。
- [ ] 7.3 校验 WAL/fsync/compaction 合同语义不变。
- [ ] 7.4 增加并接入 `BenchmarkMemoryFilesystemWrite*`、`BenchmarkMemoryFilesystemQuery*`、`BenchmarkMemoryFilesystemCompaction*` 基准与 memory conformance/scope-search 回归。
- [ ] 7.5 为 WAL 增加批量 fsync/组提交可选策略（默认 durability 语义不变）。

## 8. S7 Runner and Local Dispatch

- [ ] 8.1 优化 runner 循环配置读取与 `runFinishedPayload` 构建开销。
- [ ] 8.2 优化 local dispatch `drop_low_priority` 分类链路缓存策略。
- [ ] 8.3 将 timeline/run-finished 构建优化扩展到 teams/workflow 热路径。
- [ ] 8.4 增加并接入 `BenchmarkRunnerLoopHotpath*`、`BenchmarkRunnerTimelineEmit*`、`BenchmarkLocalDispatchPriorityClassify*` 基准并执行 security policy/event/delivery/sandbox contract 回归。

## 9. S8 Provider Adapter Stream/Decode

- [ ] 9.1 优化 provider stream 事件映射与 meta/payload 分配路径。
- [ ] 9.2 在 Anthropic/Gemini 非流式响应优先 typed fast-path，收敛 `json.Marshal + gjson` 回退。
- [ ] 9.3 优化 tool-call 参数解码缓冲复用并保持错误分类语义不变。
- [ ] 9.4 增加并接入 `BenchmarkProviderStreamEventMap*`、`BenchmarkProviderResponseDecode*` 基准与 provider parity（`check-react-contract.*`）回归。

## 10. S9 Runtime Config Read Path and Policy Resolve

- [ ] 10.1 提供只读配置快照引用接口，降低热路径值拷贝。
- [ ] 10.2 为 policy resolve 增加 `profile + override` 可失效缓存（reload 原子失效）。
- [ ] 10.3 校验 `env > file > default`、fail-fast、热更新回滚语义不变。
- [ ] 10.4 增加并接入 `BenchmarkRuntimeConfigReadPath*`、`BenchmarkMCPPolicyResolve*` 基准并执行 policy/budget/sandbox-rollout contract 回归。
- [ ] 10.5 新增 snapshot 熵预算治理参数（`retention/quota/cleanup`）并补 fail-fast 校验与热更新原子回滚测试。
- [ ] 10.6 补齐 snapshot entropy 字段的 bounded-cardinality、parser compatibility 与 replay idempotency 回归。

## 11. S10 Observability Event Pipeline

- [ ] 11.1 为 exporter 引入 `max_batch_size + max_flush_latency` 双阈值批量导出。
- [ ] 11.2 为 dispatcher 增加可配置 fanout 与慢 handler 隔离策略。
- [ ] 11.3 优化 JSON logger 编码缓冲与字段构建路径。
- [ ] 11.4 增加并接入 `BenchmarkRuntimeExporterBatch*`、`BenchmarkEventDispatcherFanout*`、`BenchmarkJSONLoggerEmit*` 基准并执行 observability export + diagnostics replay 回归。

## 12. A64 Gate Wiring, Hygiene, and Documentation

- [ ] 12.1 新增 `check-a64-semantic-stability-contract.sh/.ps1` 并接入主质量门禁。
- [ ] 12.2 新增 `check-a64-performance-regression.sh/.ps1` 并接入主质量门禁。
- [ ] 12.3 扩展 repo hygiene：扫描已跟踪与未跟踪临时工件（含 `*.go.<digits>`）。
- [ ] 12.4 更新 `docs/development-roadmap.md`、`docs/mainline-contract-test-index.md` 与相关运行时文档。
- [ ] 12.5 执行最小验证：`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`。
- [ ] 12.6 执行 A64 全量阻断门禁并记录未执行项与风险说明。
- [ ] 12.7 固化 `A64 impacted-contract suites` 的 S1~S10 显式命令映射（含 shell/PowerShell 等价）。
- [ ] 12.8 增加硬约束回归：`backpressure`、`fail_fast`、`timeout/cancel`、`decision trace` 语义不变断言。
- [ ] 12.9 新增 `check-a64-harnessability-scorecard.sh/.ps1`，输出 machine-readable scorecard（contract 覆盖、drift 统计、gate 覆盖、docs consistency）。
- [ ] 12.10 将 harnessability scorecard 接入主质量门禁并配置阈值阻断与基线更新流程。
- [ ] 12.11 为 harness 深度（planner/evaluator/sensor/garbage-collection）建立 ROI 指标与阈值（token/latency/quality），并定义超阈值降级策略。
- [ ] 12.12 将 ROI/depth 指标纳入 scorecard 报告与质量门禁阻断，支持按任务复杂度选择轻量/标准/增强深度档位。
- [ ] 12.13 固化 `computational-first, inferential-second` 校验分层：客观 correctness 阻断必须由 computational suites 给出，inferential 检查仅作主观补充。
- [ ] 12.14 为 inferential 结论补结构化证据输出（输入快照、提示版本、评分摘要），并补不确定性回归用例防止误阻断。
- [ ] 12.15 固化 A64 性能回归主门禁聚合映射：`context-production-hardening`、`diagnostics-query-performance`、`multi-agent-performance` 三类脚本必须与 `check-a64-performance-regression.*` 同步演进。
- [ ] 12.16 新增 `check-a64-impacted-gate-selection.sh/.ps1`：按 changed-files 校验 `fast/full` 选择与 mandatory suites 完整性，并接入主质量门禁。
- [ ] 12.17 新增 `check-a64-gate-latency-budget.sh/.ps1`：输出 gate step 级耗时报告并执行预算回归阻断。
- [ ] 12.18 固化门禁耗时基线与更新流程（含审批口径、证据留存、Shell/PowerShell parity）。
