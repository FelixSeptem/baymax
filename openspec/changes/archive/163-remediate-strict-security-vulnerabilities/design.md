## Context

See `proposal.md` for the motivation. The repository already requires strict `govulncheck` in both quality-gate implementations and CI reads its Go version from `go.mod`. The baseline uses Go `1.26.8` at validation time and resolves `golang.org/x/net v0.55.0`; a fresh strict scan reports reachable standard-library and module vulnerabilities. The security scan must remain a blocking check.

## Goals / Non-Goals

**Goals:**

- Make the repository declare and use patched Go `1.26.9` for local automatic toolchain selection and CI setup.
- Remediate every vulnerability that a fresh strict scan reports as reachable, using the smallest compatible module upgrades.
- Keep the dependency graph tidy and reproducible, document the new validation baseline, and produce complete strict-gate evidence.
- Recalibrate only the two runtime-recorder diagnostics baselines proven stale against current `master`, with recorded source commit, toolchain, CPU and repeated default-parameter samples.

**Non-Goals:**

- Do not add a warn-mode exception, disable `govulncheck`, suppress findings, or edit scanner/benchmark degradation thresholds.
- Do not alter diagnostics benchmark defaults or gate decision logic; do not refresh the four diagnostics-query baselines without current-master evidence.
- Do not redesign runtime behavior, APIs, configuration, diagnostics, or examples.
- Do not proactively upgrade every transitive module merely because a non-reachable advisory exists.

## Decisions

### Use a Go toolchain directive as the single patched toolchain source

`go.mod` will retain the language line and add/update `toolchain go1.26.9`. Go's automatic toolchain selection and `actions/setup-go`'s `go-version-file` then derive the same patch floor from repository state.

An untracked local Go installation alone was rejected because CI and other developers would remain on vulnerable `1.26.8`. Replacing every workflow's `go-version-file` with duplicated literal versions was rejected because it creates drift across the many jobs.

### Upgrade vulnerabilities by fresh reachable-scan evidence

The first module update targets `golang.org/x/net` at or above `v0.60.0`, the fixed version for the reachable findings. After `go mod tidy`, strict `govulncheck ./...` determines whether another reachable module must be upgraded. Each additional change must be justified by a scan result and use the advisory's fixed version or the lowest graph-compatible patched version.

A blanket update to all latest dependencies was rejected because it widens compatibility and semantic regression risk without being required to satisfy the strict reachable scan.

### Treat direct strict-scan success as the security acceptance signal

The verification sequence first proves `go version` resolves to `go1.26.9` or later and runs `govulncheck ./...` directly. It then runs the normal test, race, lint, docs, shell/PowerShell parity, and full quality gates. This separates a real advisory failure from cache/sandbox access failures while retaining the mandatory full-gate conclusion.

### Recalibrate only recorder baselines from controlled master evidence

The current `master` recorder benchmarks no longer match the 2026-04-08 sample: both current `master` and the security branch report 2232/2714 allocs/op, rather than the older 2028/2510 values. A controlled current-master run at commit `7aac3d7`, Go `1.26.9`, 16 CPU and `GOFLAGS=-p=1` uses the gate's default 200ms benchtime and 11 independent samples. Its medians become the baselines for `BenchmarkRuntimeRecorderRunFinished` and `BenchmarkRuntimeRecorderRunFinishedSandboxEnriched` only, including their ns/op, p95-ns/op and allocs/op metrics.

The existing 12% ns/allocs and 15% p95 limits remain unchanged. Updating a threshold, the benchmark defaults, the checker, or any of the four unrelated query baselines is explicitly out of scope. The refreshed gate must still pass through both supported entry points.

## Risks / Trade-offs

- [A module upgrade changes indirect graph selections] → Run `go mod tidy`, inspect `go.mod`/`go.sum`, execute package and full regression suites, and retain only scan-required changes.
- [Automatic toolchain download is unavailable on a developer machine] → The documented prerequisite makes `go1.26.9` explicit; the gate fails rather than silently using an older vulnerable standard library.
- [The vulnerability database changes between runs] → Record the toolchain, scanner version, scan output classification, and execution date in `verification.md`; do not downgrade strict mode to make a changing database pass.
- [Windows sandbox cannot access a shared Go build cache or Git Bash signal pipe] → Distinguish permission failures from advisory results, rerun read-only verification with required filesystem access, and record that environment limitation without weakening the repository gate.
- [Short recorder samples are noisy] → Use an odd repeated-sample count and median on a fixed master commit; record all environment facts, leave thresholds intact, and require a fresh full-gate result after calibration.

## Migration Plan

1. Add the toolchain pin and update the scan-required module graph.
2. Run `go mod tidy`, inspect the dependency diff, and update developer/security documentation and roadmap status for the active change.
3. Calibrate only the two recorder baselines from the controlled current-master medians, preserving thresholds/defaults/checker behavior, and document the provenance in the performance policy.
4. Verify direct strict scanning and the full quality gate on the patched toolchain.
5. Roll back by restoring the preceding `go.mod`, `go.sum`, recorder baseline values and documentation in the same security change if regression verification fails; never retain a workaround that disables or downgrades strict scanning.

## Open Questions

None. A fresh strict scan after the minimal `x/net` update is the explicit decision point for whether further module upgrades are required.

## Example Impact Assessment

无需示例变更（附理由）

工具链和依赖漏洞修复不改变任何示例的源代码、runtime path、semantic anchor 或 expected marker；所有现有示例仍由完整 Go 测试和 quality gate 覆盖。

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
