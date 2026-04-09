## 1. Baseline and Rule Freeze

- [x] 1.1 （a70-T01）整理当前治理漂移样例：roadmap 状态漂移、example impact 声明缺失/非法值。
- [x] 1.2 （a70-T02）固化 A70 规则文档与最小必需字段，明确失败分类码集合。
- [x] 1.3 （a70-T03）建立 shell/PowerShell parity 基线样例（同输入同结论）。

## 2. Roadmap and OpenSpec Status Consistency Check

- [x] 2.1 （a70-T10）实现 `scripts/check-openspec-roadmap-status-consistency.sh`。
- [x] 2.2 （a70-T11）实现 `scripts/check-openspec-roadmap-status-consistency.ps1`。
- [x] 2.3 （a70-T12）校验逻辑覆盖：`openspec list --json`、`openspec/changes/archive/INDEX.md` 与 `docs/development-roadmap.md` 三方一致性。
- [x] 2.4 （a70-T13）输出稳定分类码与可读修复建议（至少包含 `roadmap-status-drift`）。

## 3. Proposal Example Impact Declaration Check

- [x] 3.1 （a70-T20）实现 `scripts/check-openspec-example-impact-declaration.sh`。
- [x] 3.2 （a70-T21）实现 `scripts/check-openspec-example-impact-declaration.ps1`。
- [x] 3.3 （a70-T22）校验后续 proposal 声明值限定为：`新增示例`、`修改示例`、`无需示例变更（附理由）`。
- [x] 3.4 （a70-T23）输出稳定分类码与修复建议（至少包含 `missing-example-impact-declaration`、`invalid-example-impact-value`）。

## 4. Gate Integration

- [x] 4.1 （a70-T30）将 `check-openspec-roadmap-status-consistency.*` 接入 `scripts/check-docs-consistency.*`。
- [x] 4.2 （a70-T31）将 `check-openspec-example-impact-declaration.*` 接入 `scripts/check-quality-gate.*`。
- [x] 4.3 （a70-T32）在 CI 中暴露 required-check 候选并记录 job 名称与触发条件。

## 5. Documentation and Traceability

- [x] 5.1 （a70-T40）更新 `docs/development-roadmap.md`，标记 A70 的目标、边界与 DoD。
- [x] 5.2 （a70-T41）更新 `docs/mainline-contract-test-index.md`，补齐 A70 gate 与脚本映射。
- [x] 5.3 （a70-T42）更新 `AGENTS.md` 的提案协作条目，引用 A70 自动化校验路径。

## 6. Verification and Closure

- [x] 6.1 （a70-T50）补齐脚本测试用例（正向、缺字段、非法值、状态漂移、parity）。
- [x] 6.2 （a70-T51）执行 `go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`。
- [x] 6.3 （a70-T52）执行 `pwsh -File scripts/check-quality-gate.ps1` 与 `pwsh -File scripts/check-docs-consistency.ps1`，记录未执行项（如有）。
