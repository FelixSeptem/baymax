## 1. 骨架与矩阵（a62-T00~T05）

- [x] 1.1 （a62-T00）建立 `examples/agent-modes/` 统一目录与入口 README。
- [x] 1.2 （a62-T01）创建 `examples/agent-modes/MATRIX.md`，定义 `pattern -> minimal -> production-ish -> contracts -> gates -> replay` 列结构。
- [x] 1.3 （a62-T02）为所有必选模式创建最小目录骨架（`main.go`、`README.md`、运行命令占位）。
- [x] 1.4 （a62-T03）新增 `scripts/check-agent-mode-pattern-coverage.sh/.ps1` 并接入主质量门禁。
- [x] 1.5 （a62-T04）新增/扩展 `scripts/check-agent-mode-examples-smoke.sh/.ps1`，支持最小冒烟矩阵执行。
- [x] 1.6 （a62-T05）更新 `README.md` 与 `docs/mainline-contract-test-index.md` 的模式化示例索引映射。
- [x] 1.7 （a62-T06）建立历史示例 TODO 占位基线清单（`examples/` 全量扫描），确认 `TODO/TBD/FIXME/待补` 清理范围。

## 2. P0 模式落地（a62-T10~T18）

- [x] 2.1 （a62-T10）落地 `rag-hybrid-retrieval`（minimal: memory 检索；production-ish: memory + MCP + fallback）。
- [x] 2.2 （a62-T11）落地 `structured-output-schema-contract`（schema 校验 + parser compatibility + drift fixture）。
- [x] 2.3 （a62-T12）落地 `skill-driven-discovery-hybrid`（`AGENTS.md|folder|hybrid` 发现顺序与映射）。
- [x] 2.4 （a62-T13）落地 `mcp-governed-stdio-http`（单传输 + 双传输 failover）。
- [x] 2.5 （a62-T14）落地 `hitl-governed-checkpoint`（await/resume/reject/timeout/recover）。
- [x] 2.6 （a62-T15）落地 `context-governed-reference-first`（reference-first + isolate handoff + edit gate + tiering，完成判定依赖 A69 收敛）。
- [x] 2.7 （a62-T16）落地 `sandbox-governed-toolchain`（allowlist/egress/sandbox allow/deny + fallback）。
- [x] 2.8 （a62-T17）落地 `realtime-interrupt-resume`（cursor 幂等 + 恢复语义）。
- [x] 2.9 （a62-T18）落地 `multi-agents-collab-recovery`（协作编排 + mailbox/task-board 控制 + recovery 回放）。

## 3. P1/P2 增强落地（a62-T20~T38）

- [x] 3.1 （a62-T20）落地 `workflow-branch-retry-failfast`。
- [x] 3.2 （a62-T21）落地 `mapreduce-large-batch`。
- [x] 3.3 （a62-T22）落地 `state-session-snapshot-recovery`（导出/恢复/回放）。
- [x] 3.4 （a62-T23）落地 `policy-budget-admission`（precedence + budget 协同）。
- [x] 3.5 （a62-T24）落地 `tracing-eval-smoke`（tracing/eval 最小闭环）。
- [x] 3.6 （a62-T25）落地 `react-plan-notebook-loop`（ReAct + plan-notebook）。
- [x] 3.7 （a62-T26）落地 `hooks-middleware-extension-pipeline`（onion-chain 顺序/错误冒泡/上下文透传）。
- [x] 3.8 （a62-T27）落地 `observability-export-bundle`（导出 + bundle + replay）。
- [x] 3.9 （a62-T28）落地 `adapter-onboarding-manifest-capability`（manifest/capability/profile-replay/scaffold drift）。
- [x] 3.10 （a62-T29）落地 `security-policy-event-delivery`（policy/event/delivery）。
- [x] 3.11 （a62-T30）落地 `config-hot-reload-rollback`（`env > file > default` + fail-fast rollback）。
- [x] 3.12 （a62-T31）落地 `workflow-routing-strategy-switch`（输入/置信度/成本/capability 路由切换）。
- [x] 3.13 （a62-T32）落地 `multi-agents-hierarchical-planner-validator`。
- [x] 3.14 （a62-T33）落地 `mainline-mailbox-async-delayed-reconcile`。
- [x] 3.15 （a62-T34）落地 `mainline-task-board-query-control`。
- [x] 3.16 （a62-T35）落地 `mainline-scheduler-qos-backoff-dlq`。
- [x] 3.17 （a62-T36）落地 `mainline-readiness-admission-degradation`。
- [x] 3.18 （a62-T37）落地 `custom-adapter-mcp-model-tool-memory-pack`。
- [x] 3.19 （a62-T38）落地 `custom-adapter-health-readiness-circuit`。

## 4. 迁移手册与示例一致性治理（a62-T90~T100）

- [x] 4.1 （a62-T90）为每个模式补齐 `minimal/prod-ish` 运行说明与边界不覆盖声明。
- [x] 4.2 （a62-T91）为示例统一注入 diagnostics/tracing 标记并补 replay 夹具。
- [x] 4.3 （a62-T92）增强 smoke 脚本，支持按模式子集执行（CI 分片）。
- [x] 4.4 （a62-T93）补齐 shell/PowerShell parity 验证，required native command 非零即 fail-fast。
- [x] 4.5 （a62-T95）新增 `examples/agent-modes/PLAYBOOK.md`，固化 `example -> production` 迁移路径。
- [x] 4.6 （a62-T96）为每个 `production-ish` README 增加 `prod delta` 检查清单（配置/权限/容量/观测/回放/门禁）。
- [x] 4.7 （a62-T97）新增 `scripts/check-agent-mode-migration-playbook-consistency.sh/.ps1`，校验 `MATRIX.md`、`PLAYBOOK.md`、`prod delta` 一致性。
- [x] 4.8 （a62-T98）将 migration playbook consistency 门禁接入 `check-quality-gate.*` 并固化 `missing-checklist/missing-gate` 分类。
- [x] 4.9 （a62-T94）执行 `check-quality-gate.*` 与 docs consistency 全绿后标记 A62 完成。
- [x] 4.10 （a62-T99）清理 `examples/` 历史示例中的 `TODO/TBD/FIXME/待补` 占位，并将未完项迁移至 `MATRIX.md`/`PLAYBOOK.md`/`tasks.md` 可追踪条目。
- [x] 4.11 （a62-T100）新增 `scripts/check-agent-mode-legacy-todo-cleanup.sh/.ps1`，阻断 `examples/` TODO 类占位回流并接入 `check-quality-gate.*`。

## 5. 受影响门禁与验证收口

- [x] 5.1 将 `check-agent-mode-examples-smoke.*` 接入 CI required-check 候选 `agent-mode-examples-smoke-gate`。
- [x] 5.2 将 `check-agent-mode-legacy-todo-cleanup.*` 纳入 A62 required-check 候选与 CI 报告输出。
- [x] 5.3 确保 A62 新增/修改 contract 均具备对应测试覆盖（单测/integration/replay）。
- [x] 5.4 执行最小验证：`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`。
- [x] 5.5 执行 `pwsh -File scripts/check-docs-consistency.ps1` 并记录未执行项与风险说明（如有）。
- [x] 5.6 （a62-D01）当变更触及 `context-governed` 示例时，强制执行 `check-context-compression-production-contract.sh/.ps1` 与 `check-context-jit-organization-contract.sh/.ps1`。
- [x] 5.7 （a62-D02）`a62-T15` 标记完成前，确认 A69 replay/gate 结果已在当前分支稳定通过并留存验证记录。
- [x] 5.8 （a62-D03）在后续提案治理中新增 `example impact assessment` 必选项：涉及行为/配置/契约变化时，必须声明 `新增示例`、`修改示例` 或 `无需示例变更（附理由）`。
- [x] 5.9 （a62-D04）将 `example impact assessment` 约束同步落到 `AGENTS.md`（提案协作规范）并保持与 A62 spec 文案一致。
- [x] 5.10 （a62-D05）建立 `agent-mode` 示例回归稳定性基线：记录 smoke 时延（建议至少 P50/P95）、失败率、重试率与 flaky 分类口径。
- [x] 5.11 （a62-D06）若 smoke 时延或 flaky 超过阈值，必须在 a62 内增量实现治理项（分片执行、耗时预算、flaky 分类与重试策略），禁止拆分平行提案。
- [x] 5.12 （a62-D07）将触发后的示例稳定性治理检查接入 `check-quality-gate.*`（shell/PowerShell parity），并输出可审计分类（如 `example-smoke-latency-regression`、`example-smoke-flaky-regression`）。

## 6. 真实逻辑落地收口（a62-T101~T111）

- [x] 6.1 （a62-T101）建立“真实示例”完成判定：`examples/agent-modes/*/*/main.go` 禁止仅输出元数据，且禁止继续依赖 `examples/agent-modes/internal/agentmode`；必须执行主干真实运行时路径（至少触发 `core/runner`/`orchestration`/`tool/local`/`runtime`/`context`/`memory`/`mcp`/`model` 之一）。
- [x] 6.2 （a62-T102）替换 `rag-hybrid-retrieval`、`structured-output-schema-contract`、`mcp-governed-stdio-http` 为真实逻辑示例（不再依赖 `examples/agent-modes/internal/agentmode` 模拟执行），并同步更新对应 `minimal/production-ish` README。
- [x] 6.3 （a62-T103）替换 `skill-driven-discovery-hybrid`、`hitl-governed-checkpoint`、`context-governed-reference-first`、`sandbox-governed-toolchain`、`realtime-interrupt-resume` 为真实逻辑示例，并保持对应 contract 语义不降级，同时同步更新对应 README。
- [x] 6.4 （a62-T104）替换 `multi-agents-collab-recovery`、`workflow-branch-retry-failfast`、`mapreduce-large-batch`、`workflow-routing-strategy-switch`、`multi-agents-hierarchical-planner-validator` 为真实编排示例（主干流程可运行），并同步更新对应 README。
- [x] 6.5 （a62-T105）替换 `state-session-snapshot-recovery`、`policy-budget-admission`、`tracing-eval-smoke`、`react-plan-notebook-loop`、`hooks-middleware-extension-pipeline`、`observability-export-bundle` 为真实逻辑示例，并补齐 replay 可验证输出，同时同步更新对应 README。
- [x] 6.6 （a62-T106）替换 `adapter-onboarding-manifest-capability`、`security-policy-event-delivery`、`config-hot-reload-rollback`、`mainline-mailbox-async-delayed-reconcile`、`mainline-task-board-query-control`、`mainline-scheduler-qos-backoff-dlq`、`mainline-readiness-admission-degradation`、`custom-adapter-mcp-model-tool-memory-pack`、`custom-adapter-health-readiness-circuit` 为真实逻辑示例，并同步更新对应 README。
- [x] 6.7 （a62-T107）新增 `check-agent-mode-real-logic-contract.sh/.ps1`：阻断 `examples/agent-modes/*/*/main.go` 引用 `examples/agent-modes/internal/agentmode`；阻断“仅元数据打印”回流；并接入 `check-quality-gate.*`。
- [x] 6.8 （a62-T108）增强 `check-agent-mode-examples-smoke.*`：默认覆盖 `minimal + production-ish` 双变体，并校验真实执行关键输出（非纯文本占位）。
- [x] 6.9 （a62-T109）对 A62 新增/修改的 contract 补齐测试覆盖（单测 + integration + replay），并将示例变更与 contract 用例一一映射。
- [x] 6.10 （a62-T110）更新 `examples/agent-modes/**/README.md` 与 `MATRIX.md/PLAYBOOK.md`：删除“边界不覆盖”中的占位语义，补充真实依赖、运行前置条件、失败回滚、门禁映射与验收输出样例。
- [x] 6.11 （a62-T111）新增 `check-agent-mode-readme-sync-contract.sh/.ps1`：当 `examples/agent-modes/*/*/main.go` 发生行为改动时，强制对应 README 同步更新（至少覆盖运行命令、前置依赖、预期输出/验证步骤），并接入 `check-quality-gate.*`。

