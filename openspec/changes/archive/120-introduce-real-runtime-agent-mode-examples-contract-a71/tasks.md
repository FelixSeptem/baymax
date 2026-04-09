## 1. 基线与判定标准（a71-T00~T05）

- [x] 1.1 （a71-T00）建立 `agent-modes` 真实语义验收矩阵：为 28 模式定义 `semantic anchor / runtime path evidence / expected verification markers`。
- [x] 1.2 （a71-T01）建立模式分组实施计划（P0/P1/P2）并在 `MATRIX.md` 标注 a71 替换范围与状态列。
- [x] 1.3 （a71-T02）抽取可复用示例组件（避免 28 个 `main.go` 同构复制），包括运行入口、验证输出、错误分类与签名工具。
- [x] 1.4 （a71-T03）为 `minimal` 与 `production-ish` 变体定义差异化要求（production-ish 必须包含治理语义证据，不得是最小变体复制）。
- [x] 1.5 （a71-T04）建立“模式语义证据字段”统一命名规范并在示例中落地。
- [x] 1.6 （a71-T05）完成 a71 与 a62 边界声明：a71 仅追踪真实替换，不回写/复用 a62 任务勾选状态。

## 2. P0 模式真实替换（a71-T10~T18）

- [x] 2.1 （a71-T10）替换 `rag-hybrid-retrieval` 为真实语义示例（检索候选/排序/fallback 证据）。
- [x] 2.2 （a71-T11）替换 `structured-output-schema-contract` 为真实语义示例（schema 校验/兼容窗口/漂移证据）。
- [x] 2.3 （a71-T12）替换 `skill-driven-discovery-hybrid` 为真实语义示例（发现来源优先级/评分与映射证据）。
- [x] 2.4 （a71-T13）替换 `mcp-governed-stdio-http` 为真实语义示例（传输选择/failover/治理证据）。
- [x] 2.5 （a71-T14）替换 `hitl-governed-checkpoint` 为真实语义示例（await/resume/reject/timeout/recover 证据）。
- [x] 2.6 （a71-T15）替换 `context-governed-reference-first` 为真实语义示例（reference-first/isolate/edit gate/tiering 证据）。
- [x] 2.7 （a71-T16）替换 `sandbox-governed-toolchain` 为真实语义示例（allow/deny/egress/fallback 证据）。
- [x] 2.8 （a71-T17）替换 `realtime-interrupt-resume` 为真实语义示例（cursor 幂等/中断恢复证据）。
- [x] 2.9 （a71-T18）替换 `multi-agents-collab-recovery` 为真实语义示例（协作编排/mailbox/task-board/recovery 证据）。

## 3. P1 模式真实替换（a71-T20~T27）

- [x] 3.1 （a71-T20）替换 `workflow-branch-retry-failfast` 为真实语义示例。
- [x] 3.2 （a71-T21）替换 `mapreduce-large-batch` 为真实语义示例。
- [x] 3.3 （a71-T22）替换 `state-session-snapshot-recovery` 为真实语义示例（导出/恢复/回放一致性证据）。
- [x] 3.4 （a71-T23）替换 `policy-budget-admission` 为真实语义示例（precedence + budget admission 证据）。
- [x] 3.5 （a71-T24）替换 `tracing-eval-smoke` 为真实语义示例（trace + eval 闭环证据）。
- [x] 3.6 （a71-T25）替换 `react-plan-notebook-loop` 为真实语义示例（plan-notebook 同步与变更钩子证据）。
- [x] 3.7 （a71-T26）替换 `hooks-middleware-extension-pipeline` 为真实语义示例（onion 顺序/错误冒泡/透传证据）。
- [x] 3.8 （a71-T27）替换 `observability-export-bundle` 为真实语义示例（导出/bundle/replay 证据）。

## 4. P2 模式真实替换（a71-T30~T40）

- [x] 4.1 （a71-T30）替换 `adapter-onboarding-manifest-capability` 为真实语义示例。
- [x] 4.2 （a71-T31）替换 `security-policy-event-delivery` 为真实语义示例。
- [x] 4.3 （a71-T32）替换 `config-hot-reload-rollback` 为真实语义示例。
- [x] 4.4 （a71-T33）替换 `workflow-routing-strategy-switch` 为真实语义示例。
- [x] 4.5 （a71-T34）替换 `multi-agents-hierarchical-planner-validator` 为真实语义示例。
- [x] 4.6 （a71-T35）替换 `mainline-mailbox-async-delayed-reconcile` 为真实语义示例。
- [x] 4.7 （a71-T36）替换 `mainline-task-board-query-control` 为真实语义示例。
- [x] 4.8 （a71-T37）替换 `mainline-scheduler-qos-backoff-dlq` 为真实语义示例。
- [x] 4.9 （a71-T38）替换 `mainline-readiness-admission-degradation` 为真实语义示例。
- [x] 4.10 （a71-T39）替换 `custom-adapter-mcp-model-tool-memory-pack` 为真实语义示例。
- [x] 4.11 （a71-T40）替换 `custom-adapter-health-readiness-circuit` 为真实语义示例。

## 5. README 与映射文档同步（a71-T50~T56）

- [x] 5.1 （a71-T50）更新全部 `examples/agent-modes/*/{minimal,production-ish}/README.md`，补齐 `Run / Prerequisites / Real Runtime Path / Expected Output/Verification / Failure/Rollback Notes`。
- [x] 5.2 （a71-T51）更新 `examples/agent-modes/MATRIX.md`，补齐 `semantic anchor -> contract -> gate -> replay` 对照列。
- [x] 5.3 （a71-T52）更新 `examples/agent-modes/PLAYBOOK.md`，加入“真实示例替换后的生产迁移检查项”。
- [x] 5.4 （a71-T53）更新 `README.md` 与 `docs/mainline-contract-test-index.md` 的示例索引与运行说明。
- [x] 5.5 （a71-T54）补充“模式语义证据字段”文档并与脚本校验规则对齐。
- [x] 5.6 （a71-T55）新增“模式行为变更必须更新 README”的贡献约束说明并链接到门禁脚本。
- [x] 5.7 （a71-T56）完成 a71 文档与示例路径一致性自检，防止漏改。

## 6. 门禁与验证收口（a71-T70~T80）

- [x] 6.1 （a71-T70）实现 `scripts/check-agent-mode-real-runtime-semantic-contract.sh/.ps1`，阻断模板化伪语义回流。
- [x] 6.2 （a71-T71）实现 `scripts/check-agent-mode-readme-runtime-sync-contract.sh/.ps1`，阻断代码/README 漂移。
- [x] 6.3 （a71-T72）增强 `scripts/check-agent-mode-examples-smoke.sh/.ps1`：默认 `minimal + production-ish` 双变体并校验语义证据。
- [x] 6.4 （a71-T73）将 `a71` 新增门禁接入 `scripts/check-quality-gate.sh/.ps1`，保持 shell/PowerShell parity。
- [x] 6.5 （a71-T74）补齐 a71 相关单测/integration/replay 覆盖，并与模式清单逐项映射。
- [x] 6.6 （a71-T75）执行最小验证：`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`。
- [x] 6.7 （a71-T76）执行 `pwsh -File scripts/check-docs-consistency.ps1` 并记录未执行项/风险（如有）。
- [ ] 6.8 （a71-T77）执行 `pwsh -File scripts/check-quality-gate.ps1` 全绿后标记 a71 完成。
  - T77 说明：`scope=full` 下已执行到 `a64 performance regression gate`，因总预算超时触发 kill；`scope=general` 与 a71 专项门禁均已全绿，待单独补跑 full-scope。
- [x] 6.9 （a71-T78）归档前复核：28 模式全部通过“真实语义 + README 同步 + gate/replay 映射”验收。
- [x] 6.10 （a71-T79）固化 a71 验收报告到变更目录（含失败分类、回归统计与覆盖矩阵）。
- [x] 6.11 （a71-T80）完成 OpenSpec 归档前检查并准备归档任务。


