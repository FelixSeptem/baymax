## 1. 文档基线先行（A72-T00~T08）

- [x] 1.1 （A72-T00）为 28 个模式补齐 `semantic anchor / real runtime path evidence / expected verification markers / failure rollback` 文档字段基线（`MATRIX.md`）。
- [x] 1.2 （A72-T01）为 28 个模式的 `minimal` README 补齐必需章节：`Run / Prerequisites / Real Runtime Path / Expected Output/Verification / Failure/Rollback Notes`。
- [x] 1.3 （A72-T02）为 28 个模式的 `production-ish` README 补齐必需章节，并明确与 `minimal` 的行为差异。
- [x] 1.4 （A72-T03）在 `PLAYBOOK.md` 固化“文档先行 -> 实现替换 -> 门禁验收”的流程与回滚步骤。
- [x] 1.5 （A72-T04）在 `MATRIX.md` 增加 `doc-baseline-ready` 与 `impl-ready` 状态列，作为实施前置判定。
- [x] 1.6 （A72-T05）补齐模式级 `contract -> gate -> replay` 映射，确保与文档语义字段一一对应。
- [x] 1.7 （A72-T06）冻结文档基线版本并输出变更摘要，作为代码替换的唯一输入。
- [x] 1.8 （A72-T07）执行 docs consistency 自检，确保新 active change 状态与 roadmap 对齐。
- [x] 1.9 （A72-T08）确认“未完成文档基线的模式不得进入代码任务”约束已写入提案与协作文档。

## 2. 反模板门禁与顺序门禁（A72-T10~T18）

- [x] 2.1 （A72-T10）新增 `scripts/check-agent-mode-anti-template-contract.sh`，实现结构同构度与 wrapper-only 检测。
- [x] 2.2 （A72-T11）新增 `scripts/check-agent-mode-anti-template-contract.ps1`，保持与 shell 等价语义。
- [x] 2.3 （A72-T12）在 anti-template gate 中增加“模式语义必须模式内自有”检查项与失败分类码。
- [x] 2.4 （A72-T13）在 anti-template gate 中增加“minimal/production-ish 不得仅 marker 差异”检查项。
- [x] 2.5 （A72-T14）新增 `scripts/check-agent-mode-doc-first-delivery-contract.sh`，校验文档基线先于代码变更。
- [x] 2.6 （A72-T15）新增 `scripts/check-agent-mode-doc-first-delivery-contract.ps1`，保持与 shell 等价语义。
- [x] 2.7 （A72-T16）将 A72 两类新门禁接入 `scripts/check-quality-gate.sh/.ps1` 阻断路径。
- [x] 2.8 （A72-T17）补齐门禁脚本单测/集成测试，覆盖 pass/fail 分类与异常分支。
- [x] 2.9 （A72-T18）验证 shell/PowerShell parity（同输入同结论同分类码）。

## 3. P0 模式按文档替换（A72-T20~T28）

- [x] 3.1 （A72-T20）替换 `rag-hybrid-retrieval`：实现模式内自有检索语义链路（非模板骨架）。
- [x] 3.2 （A72-T21）替换 `structured-output-schema-contract`：实现 schema 校验与兼容窗口的真实行为分支。
- [x] 3.3 （A72-T22）替换 `skill-driven-discovery-hybrid`：实现来源优先级与评分映射真实路径。
- [x] 3.4 （A72-T23）替换 `mcp-governed-stdio-http`：实现传输选择/failover 的真实行为分支。
- [x] 3.5 （A72-T24）替换 `hitl-governed-checkpoint`：实现 await/resume/reject/timeout/recover 真实分支。
- [x] 3.6 （A72-T25）替换 `context-governed-reference-first`：实现 reference-first/isolate/edit gate/tiering 真实分支。
- [x] 3.7 （A72-T26）替换 `sandbox-governed-toolchain`：实现 allow/deny/egress/fallback 真实行为。
- [x] 3.8 （A72-T27）替换 `realtime-interrupt-resume`：实现 interrupt/resume 的真实状态迁移分支。
- [x] 3.9 （A72-T28）替换 `multi-agents-collab-recovery`：实现 mailbox/task-board/recovery 真实协作语义。

## 4. P1 模式按文档替换（A72-T30~T37）

- [x] 4.1 （A72-T30）替换 `workflow-branch-retry-failfast`，实现分支/重试/fail-fast 真实行为链路。
- [x] 4.2 （A72-T31）替换 `mapreduce-large-batch`，实现 shard/reduce/retry 真实行为链路。
- [x] 4.3 （A72-T32）替换 `state-session-snapshot-recovery`，实现导出/恢复/回放一致性真实链路。
- [x] 4.4 （A72-T33）替换 `policy-budget-admission`，实现 precedence + budget admission 真实仲裁路径。
- [x] 4.5 （A72-T34）替换 `tracing-eval-smoke`，实现 tracing + eval 真实闭环路径。
- [x] 4.6 （A72-T35）替换 `react-plan-notebook-loop`，实现 plan/notebook/change-hook 真实交互路径。
- [x] 4.7 （A72-T36）替换 `hooks-middleware-extension-pipeline`，实现 middleware onion + bubble 真实链路。
- [x] 4.8 （A72-T37）替换 `observability-export-bundle`，实现 export/bundle/replay 真实链路。

## 5. P2 模式按文档替换（A72-T40~T50）

- [x] 5.1 （A72-T40）替换 `adapter-onboarding-manifest-capability`，实现 manifest/capability/fallback 真实路径。
- [x] 5.2 （A72-T41）替换 `security-policy-event-delivery`，实现 policy/event/delivery 真实治理路径。
- [x] 5.3 （A72-T42）替换 `config-hot-reload-rollback`，实现 fail-fast + 原子回滚真实路径。
- [x] 5.4 （A72-T43）替换 `workflow-routing-strategy-switch`，实现 routing strategy switch 真实行为路径。
- [x] 5.5 （A72-T44）替换 `multi-agents-hierarchical-planner-validator`，实现 planner/validator/correction 真实路径。
- [x] 5.6 （A72-T45）替换 `mainline-mailbox-async-delayed-reconcile`，实现 async/delayed/reconcile 真实路径。
- [x] 5.7 （A72-T46）替换 `mainline-task-board-query-control`，实现 query/control/idempotency 真实路径。
- [x] 5.8 （A72-T47）替换 `mainline-scheduler-qos-backoff-dlq`，实现 qos/backoff/dlq 真实路径。
- [x] 5.9 （A72-T48）替换 `mainline-readiness-admission-degradation`，实现 readiness/admission/degradation 真实路径。
- [x] 5.10 （A72-T49）替换 `custom-adapter-mcp-model-tool-memory-pack`，实现 adapter pack 真实链路。
- [x] 5.11 （A72-T50）替换 `custom-adapter-health-readiness-circuit`，实现 health/readiness/circuit 真实链路。

## 6. 测试、门禁与收口（A72-T60~T70）

- [x] 6.1 （A72-T60）为 P0 模式补齐正向 + 退化/失败路径集成测试，确保语义分支可回归。
- [x] 6.2 （A72-T61）为 P1 模式补齐正向 + 退化/失败路径集成测试，确保语义分支可回归。
- [x] 6.3 （A72-T62）为 P2 模式补齐正向 + 退化/失败路径集成测试，确保语义分支可回归。
- [x] 6.4 （A72-T63）更新并执行 `check-agent-mode-examples-smoke.*`，验证 28 模式双变体真实行为证据。
- [x] 6.5 （A72-T64）执行 `go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`。
- [x] 6.6 （A72-T65）执行 `pwsh -File scripts/check-docs-consistency.ps1`，确认文档与状态一致。
- [x] 6.7 （A72-T66）执行 `pwsh -File scripts/check-quality-gate.ps1`（full）并确认 A72 门禁阻断有效。
- [x] 6.8 （A72-T67）逐项核对任务“四证据”并仅对满足条件的任务勾选。
- [x] 6.9 （A72-T68）产出 A72 验收报告（模式覆盖、失败分类、回归统计、风险与回滚点）。
- [x] 6.10 （A72-T69）完成归档前自检，确保 proposal/design/specs/tasks 与代码/文档一致。
- [x] 6.11 （A72-T70）通过 OpenSpec 归档脚本准备归档重命名并更新归档索引。
