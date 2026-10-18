## Why

Danny 集成观察表明，宿主可能同时维护 role/history/private-memory 的筛选、capability digest、预算、压缩与 handoff；虽然 Baymax 已拥有 Context Assembler、派生预算投影、`handoff.v1` 和恢复边界，但尚未定义宿主预过滤 projection 如何以可验证、reference-first 的方式进入这些既有 owner。若继续由每个宿主自行拼装 context，会产生权限扩大、digest 漂移、预算双账本及恢复时复用旧授权的风险。

现在具备立项条件：归档 162 已固定可选的宿主 plan identity/digest，归档 164 已固定恢复时不得把历史 digest 当作当前授权；本 change 在两者之间补齐 context assembly 与 handoff 的最小契约，而不把业务数据模型或授权职责迁入运行时。

## What Changes

- 新增显式 opt-in 的 `host_context_projection.v1` 接入合同：宿主提交版本化、bounded、detached 的预过滤 projection descriptor，包含 opaque source references、projection/capability digest、可选 host-plan correlation 与 child capability-subset 声明；原始私有 memory、知识库正文、凭证、业务 effect 与未过滤内容不进入该新合同或诊断。
- 在既有 Context Assembler 前提供薄验证/解析接缝，固定版本、大小、canonical digest、reference、关联和权限只收窄的校验顺序。通过后才由宿主当前授权的 resolver 提供装配所需内容；bridge 不持有 source body、不创建 context store，也不直接写诊断。
- 复用既有 `budget_projection.v1`、prefix/journal、compaction 与 `handoff.v1` owner。新增关联仅以 bounded reference/digest 投影到 handoff；它不回写预算事实、不替代 admission/terminal，也不把 digest 解释成业务授权。
- 明确 restore 必须先通过既有 handoff/source validation，再校验 projection identity/reference/digest，并由宿主按当前权限和业务 revision 重新授权/解析。任何缺失、漂移、授权拒绝或 capability expansion 都 fail-closed，且不得注入 context、继续运行或提交业务 effect。
- 交付正向/负向/边界 unit、真实 runner + fake model/tool 的 integration、Run/Stream parity、版本化 reference-only replay、shell/PowerShell contract gate 和离线示例。Replay 不解析真实 source body，不调用 provider/tool/network，不恢复运行态。
- 将路线图候选移入进行中状态，并同步维护公开的 context、integration、module-boundary、diagnostics、contract-index 和 example 文档；不修改 `examples/agent-modes`。

## Capabilities

### New Capabilities

- `host-context-assembly-handoff`: 宿主预过滤 context projection 的版本化身份、边界/完整性校验、reference-first 解析、capability 收窄、assembler/handoff/restore 接缝、Run/Stream 对等及离线 conformance。

### Modified Capabilities

- `context-compression-runtime-handoff`: 允许合法的宿主 projection correlation 以 bounded reference/digest 进入既有 `handoff.v1`，并要求 restore 在 source validation 后进行宿主 projection 重验证。
- `diagnostics-replay-tooling`: 支持 `host_context_projection.v1` 的离线、只读、确定性 replay 与稳定的 projection/reference/digest/capability/restore drift 分类。
- `go-quality-gate`: 将 host context assembly handoff 的双平台 contract gate 作为阻断性质量门禁接线，验证 replay、隐私、Run/Stream parity 和拒绝前无副作用。

## Impact

预期实现会涉及一个最小的 host-context bridge、`context/assembler` 与 `context/handoff` 的既有扩展点、真实 `core/runner` integration、`tool/diagnosticsreplay`、`tool/contributioncheck`、双平台脚本和 `examples/host-context-assembly`。实现前任务会先完成 Baymax 现状审计与最小 gap fixture；若现有接缝足够，运行时代码改动必须保持为零或最小化。

宿主继续拥有角色访问、来源信任、business reference/tool allowlist、private memory/knowledge、业务 revision、业务授权和 commit gate。Baymax 继续拥有装配、压缩、预算派生、safe cut、source checkpoint/handoff 及 Run/Stream 语义。此 change 不依赖 Danny、Stele、Provider SDK、业务数据库或新增后台 worker；不新增 env/file/hot-update 配置、QueryRuns 字段、预算账本、执行/终态/恢复状态机。

## Risks and Rollback

- **过滤顺序或 child capability 处理错误** 可能将未经授权的上下文注入模型；通过宿主先过滤、bridge 只接受收窄声明、并在 restore 时重新授权来阻断。
- **descriptor 与 source reference 或 digest 脱节** 可能导致使用过期投影；通过 detached snapshot、canonical vectors、source-owner re-resolution 和 drift fixture 拒绝。
- **预算 projection 被误当成事实源** 可能形成第二账本或改变终态；contract 仅消费现有只读 projection，单测和 gate 验证无回写、无终态影响。
- **把 handoff 或 replay 误当成恢复授权** 可能绕过宿主当前权限；restore 始终要求 source validation 与宿主当前授权，replay 完全离线。

回滚时停止向 bridge 提交 projection，回到宿主预过滤后使用既有 assembler 的路径；保留已有 checkpoint、handoff、journal、application execution record 与真实业务 effects。回滚不能撤销已经发生的宿主业务写入，也不得通过运行时伪造补偿或终态。

## Example Impact Assessment

新增示例

新增 `examples/host-context-assembly`。README 将在任何示例代码之前定义 semantic anchor、runtime path、支持/拒绝边界、expected markers 与 rollback notes；示例采用离线 fake model/tool 和 synthetic opaque references，不接入 agent-modes、live provider 或业务数据库。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`context/README.md`、`context/handoff/README.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/guides/integration.md`、`docs/mainline-contract-test-index.md`、`examples/host-context-assembly/README.md`
- 维护 owner：context、runner、host integration 与 documentation maintainers
- 验证证据：OpenSpec strict validation、documentation/example impact、roadmap/docs consistency；实施后补齐 context contract/replay、真实 runner Run/Stream parity、双平台 gate 与完整 Go/race/lint/quality/docs 门禁的实际命令和退出结果

“已同步”指本 change 已在同一功能分支建立 active proposal 的公开语义、owner、范围、文档与示例基线；它不表示运行时代码已经交付。最终 API、limits、错误、支持矩阵与实现证据仍须在任务 6 中更新并以 `verification.md` 的实际命令结果为准。没有上述维护路径的实际 Git diff、对应测试与门禁证据，任务不得勾选、change 不得归档。

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md`、`context/README.md` | context maintainers | bridge 不拥有 source body、业务授权、预算或恢复状态；`context/*` 不引入 Provider SDK |
| components | 修改文档 | `context/handoff/README.md`、实施新增 bridge README | context/runner maintainers | projection → assembler → handoff/restore owner 映射与实际 API 对照 |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime config maintainers | 明确 explicit opt-in、constructor-only limits、无新 env/file/hot-update 键与现有优先级不变 |
| contract/API | 修改文档 | `docs/guides/integration.md`、`docs/mainline-contract-test-index.md` | host integration/contract maintainers | spec requirement、正负边界/integration/replay/gate 逐项映射 |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | observability maintainers | bounded opaque reference/digest、RuntimeRecorder 单写、不新增 QueryRuns 字段或业务正文 |
| examples | 新增文档 | `examples/host-context-assembly/README.md`、`README.md` | examples maintainers | 文档基线先行、离线 smoke、expected markers 与 rollback 验证 |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | host integration maintainers | explicit opt-in、宿主 resolver 与无新增 wire command 的 active-proposal 边界 |
| best practices | 无需文档变更（附理由） | `docs/guides/best-practices.md` | documentation maintainers | 该 contract 的集成 guardrails 已由 integration/context 文档覆盖；尚未增加通用操作建议或独立 best-practice 语义 |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | active slug 与 `openspec list --json` 状态一致，归档后同步索引 |

## Validation Strategy

提案阶段执行 `openspec validate establish-host-context-assembly-handoff-contract --strict`、OpenSpec documentation/example impact 校验、roadmap/docs consistency 与 `git diff --check`。实施阶段将执行受影响包单测、真实 runner integration、reference-only replay、双平台专用 gate、`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml ./...`、`pwsh -File scripts/check-quality-gate.ps1` 与 `pwsh -File scripts/check-docs-consistency.ps1`；实际命令、退出码、风险、回滚点和未执行项记录在 `verification.md`。显式的 `./...` 是必需的，因为仓库根目录不是 Go 包，当前 linter 版本会拒绝无 Go 源文件的根目录分析。
