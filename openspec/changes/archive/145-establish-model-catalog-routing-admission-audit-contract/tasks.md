## Example Impact Assessment

无需示例变更（附理由）。首阶段仅增加 model catalog routing audit、fixture、replay、contract test 和 gate，不改变 `examples/agent-modes` 的 runtime path、配置键或 expected markers。若条件化 resolver 后续改变示例可观察选择，必须先完成 `MATRIX.md` 与受影响模式 README 的文档基线，并将声明改为“修改示例”。

## 1. Baseline and Audit Contract

- [x] 1.1 在代码任务开始前确认 feature branch 从最新 `master` 创建，记录 `git status --short`、`git branch --show-current`、worktree 路径和 `openspec list --json`，验证主线无活动 change 且未跟踪缓存未被修改。
- [x] 1.2 审计 `model/catalog`、`runtime/config` provider catalog snapshot、readiness/admission、`RuntimeRecorder` 与现有 provider-model fixtures，记录 exact-identity admission、fallback、credential evidence、generation reload 和 Run/Stream parity 的现状事实；增加审计测试并验证缺失 audit contract 时先红。
- [x] 1.3 以测试先行定义 `model_catalog_routing_admission.v1` 的 SDK-neutral input/output、canonical identity/order、bounded limits、reason taxonomy、digest 和 compatibility defaults；验证等价输入归一化稳定且无 SDK、网络、时钟、文件或 runtime mutation 副作用。
- [x] 1.4 定义候选 identity 重复、priority 冲突、ambiguous tie、unknown source、overflow、invalid generation 和 privacy 失败分类；验证所有失败均发生在 provider action 前且不产生部分成功结果。

## 2. Candidate Evaluation and Conditional Resolver

- [x] 2.1 为候选集评估补充正向/负向/边界测试，覆盖 required/optional capability、credential available/missing/invalid/unverified、strict/non-strict readiness、declared fallback 和 ordered skip reasons；验证既有 `adapter/capability` 与 readiness taxonomy 被复用。
- [x] 2.2 实现 fixture-only exact-identity audit path，输出 catalog generation、selected/blocked identity、capability outcome、credential status、fallback 和 reason sequence；验证现有 `Evaluate` 行为与历史调用保持不变。
- [x] 2.3 审计 fixture 已证明 exact-identity admission 无法表达多个 independently admissible 候选的明确选择需求；已先更新 spec/design 记录 gap，再实现纯函数 host-supplied candidate resolver；resolver 只消费 supplied snapshot/candidates/policy，不执行 discovery、I/O、credential probe、provider call 或 runtime mutation。
- [x] 2.4 为 resolver 实现 deterministic ranking/tie-break、priority 冲突 fail-fast、candidate skip reasons 和 fallback reuse；验证重复输入选择相同 identity，冲突输入不使用 last-write-wins。
- [x] 2.5 resolver 激活条件已由审计证据满足并记录；未引入 caller-order 或隐式全局路由作为替代行为。

## 3. Replay, Diagnostics, and Gates

- [x] 3.1 增加版本化 `model_catalog_routing_admission.v1` success、blocked、capability denial、credential degradation/denial、fallback、ambiguous selection、reload rollback 和 Run/Stream parity fixtures；验证 fixture 不含 endpoint、credential、raw response、prompt 或无界 body。
- [x] 3.2 扩展 `tool/diagnosticsreplay` 解析和比较，覆盖 canonical digest、reason order、generation、selection、historical defaults、unknown fields 和 overflow；验证同一 fixture 两次回放结果完全一致，expected/observed drift 分类稳定。
- [x] 3.3 本 change 不新增运行诊断事实，因此不扩展 `RuntimeRecorder`；fixture/replay 仅使用 additive、bounded 的离线结果，并通过 boundary tests 证明没有第二 readiness/terminal 状态机或 raw payload 持久化。
- [x] 3.4 扩展 `tool/contributioncheck` boundary tests，验证候选 audit/resolver 不引入 provider SDK、remote discovery、credential store、全局 mutable router、`context/*` SDK 依赖或共享 wire protocol；为每项守卫提供负向变体。
- [x] 3.5 更新 shell/PowerShell model catalog routing gate，检查 fixture schema、taxonomy、privacy/bounds、replay idempotency、reload rollback 和 Run/Stream parity；PowerShell gate 已通过，shell gate 已提供但当前 Windows sandbox 的 bash signal-pipe 权限阻止本地执行。

## 4. Documentation and Integrated Verification

- [x] 4.1 更新 `model/README.md`、`docs/runtime-config-diagnostics.md`、`docs/runtime-module-boundaries.md`、`docs/mainline-contract-test-index.md` 和 `README.md`，说明 host-supplied catalog、audit/resolver 条件化边界、fixture/replay/gate 映射及不纳入的 discovery/credential store；验证文档不宣称新的托管控制面。
- [x] 4.2 更新 `docs/development-roadmap.md`，将本 change 从候选审计状态同步为活动状态/归档状态时只引用 `openspec list --json` 与 archive index；运行 roadmap status consistency，验证无 `roadmap-status-drift`。
- [x] 4.3 运行聚焦 normal/race suites（`model/catalog`、`adapter/capability`、`runtime/config`、`runtime/diagnostics`、`tool/diagnosticsreplay`、`tool/contributioncheck`）以及 model catalog gate；记录 selection/no-gap 或 resolver-gap 证据、privacy、reload rollback 与 Run/Stream parity。
- [x] 4.4 运行 `openspec validate --all`、`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`、`scripts/check-quality-gate.*` 和 `scripts/check-docs-consistency.*`；记录精确结果、平台受限命令和等价替代证据。验证记录：`openspec validate --all` 通过（124 passed, 0 failed）；`go test ./... -count=1 -timeout 15m` 通过；`go test -race ./... -timeout 20m` 在独立聚焦/全仓执行中通过；专用 PowerShell model-catalog gate、docs consistency、roadmap status consistency、`git diff --check` 均通过；`golangci-lint run --config .golangci.yml` 返回 0，输出仅包含仓库既有 13 个 gofmt 基线问题，新增加文件无 lint/gofmt 问题。完整 `scripts/check-quality-gate.ps1` 已执行：除最后一步全仓 race 外均通过；质量门禁总预算在 race 步骤达到 900s，脚本于 154s 后终止该步骤，因此该质量门禁运行记为 `timeout`，不是已观察到的测试失败。Shell gates 已提供但当前 Windows sandbox 因 bash signal-pipe 权限（Win32 error 5）无法本地执行；对应 PowerShell gates 与聚焦 Go suites 提供等价证据。
- [x] 4.5 最终范围复核：确认没有新增 provider、远程 catalog、后台刷新、credential store/probe、全局 router、共享 wire protocol、runtime 配置键（除非 spec 明确批准）、context SDK 依赖、scheduler/ReAct 接线、parallel terminal semantics 或 raw payload 持久化；确认仅在门禁完成、归档、合并/推送后清理 feature branch/worktree。复核结果：实现仅位于 `model/catalog` 的 SDK-neutral、纯函数 audit/resolver 与离线 fixture/replay/contributioncheck/gate；未改动 provider SDK 适配、`context/*`、scheduler/ReAct、RuntimeRecorder、运行时配置键或 examples/agent-modes 行为；未引入 discovery、后台刷新、credential store/probe、全局可变 router、共享 wire protocol、第二 terminal/readiness 状态机或 raw payload 持久化。当前仅完成提案实施与验证，未归档、合并、push 或删除 feature branch/worktree。
