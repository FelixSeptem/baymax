## Why

The strict `govulncheck` quality gate correctly blocks the repository because the validation toolchain is Go `1.26.8` and the module graph retains vulnerable baseline dependencies. The host-resume admission change did not introduce those versions, but it cannot legitimately complete while the required security gate fails.

This maintenance change restores a passing strict security baseline without weakening scan policy, disabling checks, or folding unrelated feature behavior into the dependency update.

## What Changes

- Pin the repository's CI and documented validation toolchain to the patched Go `1.26.9` release.
- Upgrade the minimum dependency set that `govulncheck` reports as reachable, beginning with `golang.org/x/net`, and expand only when a fresh strict scan proves another reachable finding remains.
- Preserve the existing strict-by-default `govulncheck` behavior, shell/PowerShell parity, and non-zero failure semantics.
- Refresh only the two stale runtime-recorder diagnostics baselines from a controlled current-`master` sample; retain all benchmark defaults, thresholds, and gate logic.
- Record the resolved findings, verification commands, risk, rollback, and any scan limitations in this change's verification evidence.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. The existing `go-quality-gate` and `security-baseline-s1` requirements already require strict blocking vulnerability scans. This change remediates versions so the existing requirements can pass; it does not change the public quality-gate, configuration, or runtime contract. `skip_specs: true` is therefore set in `.openspec.yaml`.

## Impact

- CI Go toolchain configuration, `go.mod`, and `go.sum`.
- Development and security verification documentation, including the minimum Go patch level and strict-scan evidence.
- Full quality gates and `govulncheck`; the diagnostics recorder gate baseline is calibrated from current `master`, while its defaults, thresholds and failure semantics remain unchanged. No runtime API, protocol, event schema, or example behavior is intentionally changed.

## Example Impact Assessment

无需示例变更（附理由）

This is a toolchain and dependency vulnerability remediation. Existing examples retain their source code, runtime paths, semantic markers, and expected output; they are nevertheless covered by the normal full Go test and quality-gate verification.

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
