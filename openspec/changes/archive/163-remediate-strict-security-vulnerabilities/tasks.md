## 1. 严格扫描基线与最小修复

- [x] 1.1 记录 Go `1.26.8` 与 `golang.org/x/net v0.55.0` 下的 strict `govulncheck ./...` 失败证据、可达分类和修复版本；验证扫描的非零退出来自 advisory 而非缓存/权限错误。
- [x] 1.2 在 `go.mod` 声明 `toolchain go1.26.9`，将 `golang.org/x/net` 升级至至少 `v0.60.0` 并执行 `go mod tidy`；验证 `go version` 解析为 `1.26.9` 或更高、`go.mod`/`go.sum` 只保留扫描所需的最小一致图。
- [x] 1.3 在 toolchain/module 更新后运行 strict `govulncheck ./...`；仅在仍有可达 finding 时按报告修复版本逐一升级模块并重复扫描，验证最终 exit=0，且不使用 warn/disabled scan 或 threshold 改动。

## 2. 文档和治理同步

- [x] 2.1 同步 `CONTRIBUTING.md`、`docs/versioning-and-compatibility.md` 与 `docs/runtime-config-diagnostics.md` 的 Go `1.26.9` patched baseline、strict scan/自动 toolchain 行为和失败策略；验证文档不承诺降低扫描或改变 runtime/public API 语义。
- [x] 2.2 同步 `docs/development-roadmap.md`、proposal/design/tasks 的 active-change 状态、Example Impact Assessment 与 Public Semantic Documentation Commitment；验证 documentation-impact、roadmap status 和 example-impact declarations 通过，维护路径进入本分支 Git diff。
- [x] 2.3 从受控 current-`master` recorder 样本刷新仅两项 diagnostics baseline，并在 `docs/performance-policy.md` 与 verification 中记录提交、Go、CPU、默认采样参数、样本数和中位数；验证 baseline/阈值/default/checker 的 scope 与双入口 gate 结果。

## 3. 交付验证

- [x] 3.1 运行 `go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`、`pwsh -File scripts/check-docs-consistency.ps1`、`git diff --check`、PowerShell 与 Git Bash quality-gate 路径；验证实际 exit-code，记录 Windows cache/signal-pipe 环境限制及允许访问后的结果。
- [x] 3.2 使用 patched toolchain 运行 `pwsh -File scripts/check-quality-gate.ps1`，验证 strict `govulncheck` 与完整质量门禁 exit=0；将扫描版本、模块版本、风险、回滚和未执行项写入 `verification.md`，不得以计划或单项通过代替完整门禁。
- [x] 3.3 完整验证后使用 `scripts/openspec-archive-seq.ps1` 归档本安全维护变更并同步 roadmap/索引；仅在用户获准交付时提交、合并最新 master、验证合并结果、推送和清理已合并分支。

## Example Impact Assessment

无需示例变更（附理由）

工具链和依赖漏洞修复不改变任何示例的源代码、runtime path、semantic anchor 或 expected marker；所有现有示例仍由 `go test ./...` 和完整 quality gate 覆盖。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`CONTRIBUTING.md`、`docs/versioning-and-compatibility.md`、`docs/runtime-config-diagnostics.md`、`docs/performance-policy.md`、`docs/development-roadmap.md`
- 维护 owner：security baseline、performance gate、Go toolchain 与 documentation maintainers
- 验证证据：`verification.md` 记录受控 `master` recorder 采样、实际退出码；`govulncheck ./...`、`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`、`pwsh -File scripts/check-quality-gate.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`git diff --check`

## Documentation Impact Assessment

| Area | Assessment | Maintained paths | Owner | Evidence / reason |
| --- | --- | --- | --- | --- |
| architecture | 无需文档变更（附理由） | — | runtime maintainers | 工具链与依赖补丁不改变模块依赖方向或执行 owner。 |
| components | 无需文档变更（附理由） | — | component maintainers | 无新增组件、组件入口或生命周期变更。 |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md`、`docs/performance-policy.md` | security baseline/performance gate maintainers | 记录 patched Go baseline、自动工具链、strict 扫描失败策略及仅两项 recorder 基线的受控主分支来源；docs consistency gate。 |
| contract/API | 无需文档变更（附理由） | — | contract maintainers | 现有 strict 扫描合同保持不变，无公开 runtime API 或 wire schema 变化。 |
| diagnostics | 无需文档变更（附理由） | — | observability maintainers | 不新增诊断字段或写入路径，RuntimeRecorder ownership 不变。 |
| examples | 无需文档变更（附理由） | — | example maintainers | 示例源代码、runtime path、semantic anchor、expected marker 不变。 |
| CLI/integration | 无需文档变更（附理由） | — | integration maintainers | CLI 及宿主接入合同未变，CI 继续从 go.mod 读取工具链。 |
| best practices | 修改文档 | `CONTRIBUTING.md`、`docs/versioning-and-compatibility.md` | Go toolchain maintainers | 区分语言版本与 patched validation toolchain，说明离线预装和 strict 失败策略。 |
| roadmap | 修改文档 | `README.md`、`docs/development-roadmap.md` | documentation maintainers | 登记独立安全维护，并将已归档 162 的旧 active 链接同步至真实归档路径；roadmap status gate。 |
