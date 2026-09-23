## Example Impact Assessment

无需示例变更（附理由）：实施范围仅包含离线 audit、fixture、replay、benchmark、boundary test、gate 和文档，不改变 `examples/agent-modes` 的运行路径、配置键或 expected markers。运行时投影若未来立项，必须进入新的 change 并重新评估。

## 1. Baseline and audit contract

- [ ] 1.1 从最新 `master` 确认 feature branch、worktree、`git status --short`、`openspec list --json` 和现有 active changes；保留未跟踪缓存，不将其视为提案交付物。
- [ ] 1.2 审计 `core/types.Tool`、`tool/local.Registry`、MCP metadata、adapter manifest/capability、allowlist、sandbox、Skill mapping 与现有 tool lifecycle/replay owner，记录已准入事实的来源和边界；为缺失 audit contract 先写失败测试。
- [ ] 1.3 以测试先行定义 `tool_schema_pressure_selection_audit.v1` snapshot DTO、canonical schema digest、bounded limits、`utf8_bytes_div4_v1` estimator、pressure levels、quality metrics、strategy enum、conclusion 和 reason taxonomy。
- [ ] 1.4 覆盖 duplicate identity、missing admission、schema overflow、identity/label overflow、gold-set conflict、unsupported strategy、forbidden source、privacy material 和 no-partial-result 行为。

## 2. Pressure and strategy scoring

- [ ] 2.1 实现 provider-neutral schema normalization、digest、byte/estimate metrics 和 pressure budget comparison；不得调用 Tool/MCP/Provider/tokenizer 或写 runtime state。
- [ ] 2.2 实现 `full_admitted_set`、`capability_filtered_set`、`priority_top_k`、`source_partitioned_set` 和 bounded `fixture_declared_strategy` 的确定性候选集合与 tie-break。
- [ ] 2.3 实现 synthetic expected/allowed/forbidden/fallback gold validation，以及 precision、recall、F1、expected-hit、forbidden-hit 和 fallback coverage 计算。
- [ ] 2.4 实现多策略比较与 conclusion 规则：只有压力超阈值、质量退化和双指标改善同时成立时才输出 `projection_candidate`；其余情况保持对应的非候选结论。
- [ ] 2.5 增加策略等价输入稳定性、重复输入稳定性、priority 冲突和 bounded skip reason 测试，验证评分结果不会接入 registry、ModelRequest 或 provider projection。

## 3. Corpus advisory, replay, and fixtures

- [ ] 3.1 增加 synthetic 强制 fixture，覆盖 within-budget、pressure-only、quality-only、projection-candidate、gold conflict、overflow、strategy drift、historical defaults 和 Run/Stream parity。
- [ ] 3.2 增加可选 Eval corpus/Badcase advisory fixture，覆盖缺失、未知字段、标签不足和覆盖率趋势；验证 advisory 不会改变 synthetic gate 结论。
- [ ] 3.3 扩展 `tool/diagnosticsreplay` 解析和比较，覆盖 digest、estimator version、metrics、strategy order、conclusion、unknown fields、historical defaults、overflow 和 parity drift；验证两次 replay 完全一致。
- [ ] 3.4 增加计算型 benchmark，测量工具数量、canonical bytes、estimate、策略运行耗时和质量指标；benchmark 必须离线、确定性、有界，不调用模型或 provider。
- [ ] 3.5 增加 `tool/contributioncheck` boundary tests，拒绝 provider SDK/tokenizer、network/download、credential store、global selector/router、runtime wiring、raw schema/prompt/output/tool-result persistence 和第二状态机；为守卫提供负向变体。

## 4. Documentation and gates

- [ ] 4.1 更新 `model/README.md` 或对应 tool 文档、`docs/development-roadmap.md`、`docs/runtime-module-boundaries.md`、`docs/mainline-contract-test-index.md` 和 `README.md`，说明 audit-only 双轨证据、策略评分和不纳入的 runtime projection。
- [ ] 4.2 增加 shell/PowerShell tool schema audit gate，检查 fixture schema、privacy/bounds、pressure metrics、quality metrics、strategy drift、corpus advisory isolation、replay idempotency 和 Run/Stream parity。
- [ ] 4.3 验证 Example Impact Assessment 合法且 `examples/agent-modes` 未发生行为变化；若未来触及示例，先更新 MATRIX/README 文档基线。

## 5. Integrated verification and scope review

- [ ] 5.1 运行聚焦 normal/race suites（audit package、tool/local、tool/diagnosticsreplay、tool/contributioncheck、相关 manifest/capability 与 integration parity）及专用 gate。
- [ ] 5.2 运行 `openspec validate --all`、`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`、quality gate 和 docs consistency；记录平台受限 shell 命令与等价 PowerShell 证据。
- [ ] 5.3 最终范围复核：确认没有 runtime ModelRequest/provider 接线、动态下载、marketplace、credential store/probe、provider tokenizer、全局 router、第二 admission/terminal 状态机、raw payload 持久化、scheduler/ReAct 改动或新配置键；确认归档前不清理 feature branch/worktree。
