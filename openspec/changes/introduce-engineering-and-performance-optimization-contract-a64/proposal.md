## Why

A63 正在实施命名与文档收敛，主干语义边界趋于稳定；但全仓主干路径仍存在明显的热路径开销与工程治理缺口（高频 map/JSON 分配、全量过滤排序、file backend 持久化放大、重复 goroutine 包装、临时产物回流噪声）。此时启动 A64，可以在不改变行为语义前提下一次性收口性能与工程优化同域需求，并为后续实施提供统一门禁与回归基线。

## What Changes

- 新增 A64 主合同：工程优化 + 性能优化（S1~S10）统一在单提案内吸收，不新增平行性能提案。
- 按模块固定优化分片并要求映射到 A64-S1~S10：
  - S1 `context/*` 与 stage2 provider/journal；
  - S2 `runtime/diagnostics` + `observability/event.RuntimeRecorder`；
  - S3 `orchestration/scheduler|mailbox|composer` file backend + task-board query；
  - S4 `mcp/http`、`mcp/stdio`、`mcp/diag`；
  - S5 `skill/loader` discover/compile/scoring；
  - S6 `memory/filesystem` 查询/写入/索引；
  - S7 `core/runner` + `tool/local` + teams/workflow timeline 热路径；
  - S8 `model/openai|anthropic|gemini` 流式映射与非流式解码；
  - S9 `runtime/config` 读路径与 policy resolve；
  - S10 `observability` dispatcher/logger/exporter 管线。
- 对齐 roadmap 的关键动作补齐：
  - S1 补齐 `prefixCache/ca3State` run-finished 清理 + TTL/LRU 上限治理，以及 CA3 stage2“无增量跳过”开关；
  - S3 补齐持久化节流/批次参数治理与 fail-fast 校验、热更新回滚测试；
  - S6 补齐 WAL 批量 fsync/组提交可选策略（默认 durability 语义不变）。
  - S2/S9 补齐 inferential feedback advisory 闭环：接入 `runtime.eval.*` 与运行态质量信号，仅输出可观测建议，不直接改写 readiness/admission 既有 deny 语义；
  - S3 补齐 realtime interrupt/resume cursor 与 isolate-handoff 关键状态的可恢复边界，优先复用统一 state/session snapshot 合同，不引入第二套事实源；
  - S3/S9 补齐 snapshot 熵预算治理：新增 `retention/quota/cleanup` 治理参数、fail-fast 校验与热更新原子回滚测试，默认行为保持不变。
  - 横切补齐 multi-agent 涌现行为治理：补齐并发/交错/重放场景下的 deterministic matrix 与 drift 分类，避免多 agent 级联错误在优化过程中被放大；
  - 横切补齐 harness ROI 与动态深度治理：为 planner/evaluator/sensor/garbage-collection 开销建立可量化 ROI 阈值与复杂度分层策略，避免“harness 过度工程”；
  - 横切补齐 harness 可测试性分层：强制 `computational sensors` 作为客观基线，`inferential sensors` 仅用于主观质量补充，不得单独替代基础 contract 校验。
  - 横切补齐门禁执行效率治理：按 changed-files 映射 impacted suites，支持 `fast/full` 分层执行但禁止跳过 mandatory contract/perf suites；
  - 横切补齐门禁耗时预算治理：按 gate step 输出耗时指标并执行阈值回归阻断，缩短反馈周期且不削弱阻断强度。
- 固化 A64 强门禁：
  - `check-a64-semantic-stability-contract.sh/.ps1`（Run/Stream 等价、diagnostics schema/reason taxonomy 不漂移、replay idempotency）；
  - `check-a64-performance-regression.sh/.ps1`（`ns/op`、`allocs/op`、`B/op` 阈值阻断）；
  - `check-a64-impacted-gate-selection.sh/.ps1`（changed-files -> impacted suites 映射、`fast/full` 选择正确性、mandatory suites 完整性）；
  - `check-a64-gate-latency-budget.sh/.ps1`（gate step 级耗时采集与预算回归阻断）；
  - `check-a64-performance-regression.*` 必须显式聚合已有性能回归子门禁：`check-context-production-hardening-benchmark-regression.*`、`check-diagnostics-query-performance-regression.*`、`check-multi-agent-performance-regression.*`；
  - `check-a64-harnessability-scorecard.sh/.ps1`（contract 覆盖、drift 统计、主干 gate 覆盖、docs consistency、ROI/depth 指标机器可读报告与阈值阻断）；
  - `A64 impacted-contract suites` 按模块最低必跑。
- 固化工程优化补漏：
  - repo hygiene 扩展到未跟踪临时工件扫描（`git ls-files --others --exclude-standard`），阻断 `*.go.<digits>` / `*.tmp` / `*.bak` / `*~` 回流；
  - A64 新增 benchmark 必须提供按模块可单独执行入口，避免单一 benchmark 文件持续膨胀。
- 全部优化必须满足：可开关、可回滚、默认语义不变。
- 硬约束显式对齐：
  - 不改变 Run/Stream、backpressure、fail_fast、timeout/cancel、reason taxonomy、decision trace 语义；
  - 不绕过现有 contract gate 与 replay 约束。

## Capabilities

### New Capabilities
- `engineering-and-performance-optimization-contract`: 统一定义 A64-S1~S10 的优化范围、语义稳定约束、基准回归与门禁阻断标准。

### Modified Capabilities
- `go-quality-gate`: 增加 A64 语义稳定/性能回归/impacted suites 阻断接线与 shell/PowerShell 等价要求。
- `multi-agent-mainline-performance-baseline`: 扩展基准矩阵，纳入 A64 热点路径并固化阈值治理口径。
- `diagnostics-query-performance-baseline`: 增加 diagnostics 查询与聚合路径在 A64 语义不变前提下的性能回归断言。

## Impact

- 代码范围（按 S1~S10）：`core/*`、`context/*`、`runtime/*`、`observability/*`、`mcp/*`、`model/*`、`skill/*`、`memory/*`、`orchestration/*`、`tool/local/*`。
- 测试与基准：新增/扩展模块级 benchmark、contract suites、replay fixture 断言。
- 脚本门禁：新增 A64 专项 gate，并更新 `check-quality-gate.*` 与 repo hygiene 规则。
- 受影响合同套件：按 S1~S10 落地 `A64 impacted-contract suites` 显式映射与最低必跑命令。
- 文档与索引：同步 `docs/development-roadmap.md`、`docs/mainline-contract-test-index.md`、相关 runtime/diagnostics 文档。
