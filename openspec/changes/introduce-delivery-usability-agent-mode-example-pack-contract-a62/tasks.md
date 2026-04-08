## 1. 骨架与矩阵（a62-T00~T05）

- [ ] 1.1 （a62-T00）建立 `examples/agent-modes/` 统一目录与入口 README。
- [ ] 1.2 （a62-T01）创建 `examples/agent-modes/MATRIX.md`，定义 `pattern -> minimal -> production-ish -> contracts -> gates -> replay` 列结构。
- [ ] 1.3 （a62-T02）为所有必选模式创建最小目录骨架（`main.go`、`README.md`、运行命令占位）。
- [ ] 1.4 （a62-T03）新增 `scripts/check-agent-mode-pattern-coverage.sh/.ps1` 并接入主质量门禁。
- [ ] 1.5 （a62-T04）新增/扩展 `scripts/check-agent-mode-examples-smoke.sh/.ps1`，支持最小冒烟矩阵执行。
- [ ] 1.6 （a62-T05）更新 `README.md` 与 `docs/mainline-contract-test-index.md` 的模式化示例索引映射。
- [ ] 1.7 （a62-T06）建立历史示例 TODO 占位基线清单（`examples/` 全量扫描），确认 `TODO/TBD/FIXME/待补` 清理范围。

## 2. P0 模式落地（a62-T10~T18）

- [ ] 2.1 （a62-T10）落地 `rag-hybrid-retrieval`（minimal: memory 检索；production-ish: memory + MCP + fallback）。
- [ ] 2.2 （a62-T11）落地 `structured-output-schema-contract`（schema 校验 + parser compatibility + drift fixture）。
- [ ] 2.3 （a62-T12）落地 `skill-driven-discovery-hybrid`（`AGENTS.md|folder|hybrid` 发现顺序与映射）。
- [ ] 2.4 （a62-T13）落地 `mcp-governed-stdio-http`（单传输 + 双传输 failover）。
- [ ] 2.5 （a62-T14）落地 `hitl-governed-checkpoint`（await/resume/reject/timeout/recover）。
- [ ] 2.6 （a62-T15）落地 `context-governed-reference-first`（reference-first + isolate handoff + edit gate + tiering，完成判定依赖 A69 收敛）。
- [ ] 2.7 （a62-T16）落地 `sandbox-governed-toolchain`（allowlist/egress/sandbox allow/deny + fallback）。
- [ ] 2.8 （a62-T17）落地 `realtime-interrupt-resume`（cursor 幂等 + 恢复语义）。
- [ ] 2.9 （a62-T18）落地 `multi-agents-collab-recovery`（协作编排 + mailbox/task-board 控制 + recovery 回放）。

## 3. P1/P2 增强落地（a62-T20~T38）

- [ ] 3.1 （a62-T20）落地 `workflow-branch-retry-failfast`。
- [ ] 3.2 （a62-T21）落地 `mapreduce-large-batch`。
- [ ] 3.3 （a62-T22）落地 `state-session-snapshot-recovery`（导出/恢复/回放）。
- [ ] 3.4 （a62-T23）落地 `policy-budget-admission`（precedence + budget 协同）。
- [ ] 3.5 （a62-T24）落地 `tracing-eval-smoke`（tracing/eval 最小闭环）。
- [ ] 3.6 （a62-T25）落地 `react-plan-notebook-loop`（ReAct + plan-notebook）。
- [ ] 3.7 （a62-T26）落地 `hooks-middleware-extension-pipeline`（onion-chain 顺序/错误冒泡/上下文透传）。
- [ ] 3.8 （a62-T27）落地 `observability-export-bundle`（导出 + bundle + replay）。
- [ ] 3.9 （a62-T28）落地 `adapter-onboarding-manifest-capability`（manifest/capability/profile-replay/scaffold drift）。
- [ ] 3.10 （a62-T29）落地 `security-policy-event-delivery`（policy/event/delivery）。
- [ ] 3.11 （a62-T30）落地 `config-hot-reload-rollback`（`env > file > default` + fail-fast rollback）。
- [ ] 3.12 （a62-T31）落地 `workflow-routing-strategy-switch`（输入/置信度/成本/capability 路由切换）。
- [ ] 3.13 （a62-T32）落地 `multi-agents-hierarchical-planner-validator`。
- [ ] 3.14 （a62-T33）落地 `mainline-mailbox-async-delayed-reconcile`。
- [ ] 3.15 （a62-T34）落地 `mainline-task-board-query-control`。
- [ ] 3.16 （a62-T35）落地 `mainline-scheduler-qos-backoff-dlq`。
- [ ] 3.17 （a62-T36）落地 `mainline-readiness-admission-degradation`。
- [ ] 3.18 （a62-T37）落地 `custom-adapter-mcp-model-tool-memory-pack`。
- [ ] 3.19 （a62-T38）落地 `custom-adapter-health-readiness-circuit`。

## 4. 迁移手册与示例一致性治理（a62-T90~T100）

- [ ] 4.1 （a62-T90）为每个模式补齐 `minimal/prod-ish` 运行说明与边界不覆盖声明。
- [ ] 4.2 （a62-T91）为示例统一注入 diagnostics/tracing 标记并补 replay 夹具。
- [ ] 4.3 （a62-T92）增强 smoke 脚本，支持按模式子集执行（CI 分片）。
- [ ] 4.4 （a62-T93）补齐 shell/PowerShell parity 验证，required native command 非零即 fail-fast。
- [ ] 4.5 （a62-T95）新增 `examples/agent-modes/PLAYBOOK.md`，固化 `example -> production` 迁移路径。
- [ ] 4.6 （a62-T96）为每个 `production-ish` README 增加 `prod delta` 检查清单（配置/权限/容量/观测/回放/门禁）。
- [ ] 4.7 （a62-T97）新增 `scripts/check-agent-mode-migration-playbook-consistency.sh/.ps1`，校验 `MATRIX.md`、`PLAYBOOK.md`、`prod delta` 一致性。
- [ ] 4.8 （a62-T98）将 migration playbook consistency 门禁接入 `check-quality-gate.*` 并固化 `missing-checklist/missing-gate` 分类。
- [ ] 4.9 （a62-T94）执行 `check-quality-gate.*` 与 docs consistency 全绿后标记 A62 完成。
- [ ] 4.10 （a62-T99）清理 `examples/` 历史示例中的 `TODO/TBD/FIXME/待补` 占位，并将未完项迁移至 `MATRIX.md`/`PLAYBOOK.md`/`tasks.md` 可追踪条目。
- [ ] 4.11 （a62-T100）新增 `scripts/check-agent-mode-legacy-todo-cleanup.sh/.ps1`，阻断 `examples/` TODO 类占位回流并接入 `check-quality-gate.*`。

## 5. 受影响门禁与验证收口

- [ ] 5.1 将 `check-agent-mode-examples-smoke.*` 接入 CI required-check 候选 `agent-mode-examples-smoke-gate`。
- [ ] 5.2 将 `check-agent-mode-legacy-todo-cleanup.*` 纳入 A62 required-check 候选与 CI 报告输出。
- [ ] 5.3 确保 A62 新增/修改 contract 均具备对应测试覆盖（单测/integration/replay）。
- [ ] 5.4 执行最小验证：`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`。
- [ ] 5.5 执行 `pwsh -File scripts/check-docs-consistency.ps1` 并记录未执行项与风险说明（如有）。
- [ ] 5.6 （a62-D01）当变更触及 `context-governed` 示例时，强制执行 `check-context-compression-production-contract.sh/.ps1` 与 `check-context-jit-organization-contract.sh/.ps1`。
- [ ] 5.7 （a62-D02）`a62-T15` 标记完成前，确认 A69 replay/gate 结果已在当前分支稳定通过并留存验证记录。
- [ ] 5.8 （a62-D03）在后续提案治理中新增 `example impact assessment` 必选项：涉及行为/配置/契约变化时，必须声明 `新增示例`、`修改示例` 或 `无需示例变更（附理由）`。
- [ ] 5.9 （a62-D04）将 `example impact assessment` 约束同步落到 `AGENTS.md`（提案协作规范）并保持与 A62 spec 文案一致。
