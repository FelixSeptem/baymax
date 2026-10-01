## Example Impact Assessment

无需示例变更（附理由）：实施范围仅为离线 evidence fixture、replay、contribution gate 和文档映射；不改变 `examples/agent-modes` 的配置、runtime path、expected markers 或 rollback notes。若任一实现任务证明示例可观察行为变化，必须先更新 `examples/agent-modes/MATRIX.md` 与受影响 README，并将本声明改为“修改示例”后才可勾选代码任务。

## 1. Evidence contract and deterministic verdict

- [x] 1.1 在 `model/conformance` 为 `provider_context_cache_evidence.v1` 建立有界、SDK-neutral 的 model、canonical serialization、digest 和 stable reason vocabulary；添加正向、未知版本、冲突 identity、reference/privacy/overflow 反例单测，并验证 `go test ./model/conformance -count=1`。
- [x] 1.2 实现 required dimensions 与 `no-drift` / `drift-confirmed` / `insufficient-evidence` 的纯函数 verdict derivation；覆盖 role、Skill/tail、tool-result、ordering、Run/Stream cache parity 和缺失 evidence 的优先级边界，并验证对应 `model/conformance` 单测。
- [x] 1.3 实现宿主成本/P50/P95 摘要的单位、样本量、排序、阈值和 non-negative 校验；覆盖稳定 P95 regression、mixed unit、未达样本数与不可重算阈值，并验证 drift 不会生成价格模型或 runtime 配置。

## 2. Offline replay and fixture coverage

- [x] 2.1 在 `tool/diagnosticsreplay` 实现对既有 `provider_request_projection.v1` case/digest 的只读引用校验与 `provider_context_cache_evidence.v1` replay；覆盖不调用 provider/网络/工具和 historical projection fixture 不受影响，并验证 `go test ./tool/diagnosticsreplay -count=1`。
- [x] 2.2 添加 OpenAI、Anthropic、Gemini 的 reference-only seed fixtures，以及 no-drift、insufficient-evidence、confirmed structured-context/cache drift、invalid reference、raw payload、bounds 和 P95 regression cases；验证 replay 输出稳定 verdict、digest 和 reason codes。
- [x] 2.3 增加 Run/Stream cache parity 与重复 replay idempotency 覆盖；验证完整 fixture 集合在不同输入排序下的 canonical 结果一致，并对冲突重复项 fail-fast。

## 3. Contribution gate and owner routing

- [x] 3.1 在 `tool/contributioncheck` 增加 evidence admission aggregation 与 review-only owner route；验证 `drift-confirmed` 只指向既有 `model/<provider>` owner，绝不自动修改 adapter、fixture expectation、Prompt、Skill、Tool、Policy、配置或 gate。
- [x] 3.2 添加 shell 与 PowerShell gate entry point，覆盖相同 fixture 的 verdict/digest/reason parity、缺失 fixture 与 malformed fixture fail-fast；验证两种入口的定向 gate 测试均通过。

## 4. Documentation and proposal-state alignment

- [x] 4.1 更新 `docs/development-roadmap.md`：将 141、149、150、151 标为归档基线，列出本 evidence-first Provider change 的启动/停止边界，并验证 Roadmap 候选与 `openspec list --json` 状态一致。
- [x] 4.2 更新 `docs/mainline-contract-test-index.md`、`model/README.md` 和 gate/replay 说明，映射 fixture、reason codes、owner、验证命令、风险与 rollback；验证不泄漏 raw payload 或引入 Provider SDK 到非 `model/<provider>` 包。
- [x] 4.3 根据用户授权，对既有 A63 staged-split line-budget exception 清单中的 11 条已到期路径执行一次性窄范围续期至 `2026-12-31`；保持 owner、reason、baseline_lines、allow_growth 与阈值不变，并记录到期前拆分/复审责任。

## 5. Verification and delivery evidence

- [x] 5.1 运行受影响包的正向、负向、边界和 integration/replay 测试，包括 `go test ./model/conformance ./tool/diagnosticsreplay ./tool/contributioncheck -count=1`，并保存可复现结果。
- [x] 5.2 运行 `go test ./... -count=1` 与 `go test -race ./...`；若 Windows 文件占用出现，使用隔离 `GOCACHE`/`GOTMPDIR` 逐包复现并记录实际失败原因，不得把环境错误标为通过。
- [x] 5.3 运行 `golangci-lint run --config .golangci.yml`、`pwsh -File scripts/check-quality-gate.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`openspec validate --all` 和 `git diff --check`，确认 proposal、spec、代码、测试与文档一致后再勾选任务。

验证备注：受影响包测试、`go test ./... -count=1`、全量 `go test -race ./...`、`golangci-lint run --config .golangci.yml`、`pwsh -NoProfile -File scripts/check-quality-gate.ps1`、`pwsh -NoProfile -File scripts/check-docs-consistency.ps1`、`openspec validate --all`、change strict validation、Example Impact gate 与 `git diff --check` 均通过。完整质量门禁包含 A64 语义稳定性、性能回归、全链路示例 smoke 与 govulncheck；此前 Windows 文件占用导致的 race 复现已通过隔离缓存逐包验证，本次全量 race 也稳定通过。既有 11 条 A63 line-budget exception 已按用户授权窄范围续期至 `2026-12-31`，未放宽阈值、未新增路径、未修改业务代码。
