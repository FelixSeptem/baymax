## 1. 接缝审计与文档基线

- [x] 1.1 在 `verification.md` 补齐 owner API/validator/defaults/support 审计与最小 gap fixtures；验证 composer Run/Stream 仅委托 runner、teams validator 私有、DispatchChild shared claim 限制，区分用户报告和本仓库复现。
- [x] 1.2 核对 `examples/host-execution-plan/README.md` semantic anchor、runtime path、expected markers、rollback 和支持矩阵；验证 code 实施前文档基线齐全，scope 变化先更新 proposal/design/spec/docs。
- [x] 1.3 固定 package API、结构事实与受信 opaque binding 责任于 `orchestration/planbridge/README.md`；验证与 spec 的 schema/axes/default limits/error families 一一对应，并将新增维护路径同步到三份工件承诺。

## 2. 版本化 descriptor 与 bounded 编译

- [x] 2.1 在 `orchestration/planbridge` 先补 descriptor/limits 正向、负向、边界测试，再实现 schema/identity/axes/ref 与 capability 校验；验证 `go test ./orchestration/planbridge -count=1` 覆盖未知版本、必填缺失、unsupported mailbox/resumed/shared queue 和全部 limit 的 exact/one-beyond。
- [x] 2.2 实现 bounded binding snapshot 与 immutable compiled facts；验证 cyclic/unsupported/deep/oversized input 拒绝、无提前无界 marshal/clone、caller map/slice mutation 不改变编译结果，并记录 callback/service 不能深拷贝的责任。
- [x] 2.3 实现 canonical bytes/digest 与版本化 golden vectors；验证 map order 等价、semantic list order/plan version/authority/tool/budget 改变 digest、source normalized defaults 固定，且输出不暴露 business body/credentials。
- [x] 2.4 实现 bounded field-path/error taxonomy 与原生错误 unwrap；验证 unsupported_version/invalid_plan/limit_exceeded/binding_mismatch/unsupported_capability 可稳定识别且不回显 payload。

## 3. 单一原生验证 owner 与全计划 preflight

- [x] 3.1 复用 `workflow.ValidateDefinition` 校验 flat workflow；验证 cycle/dangling/duplicate ID/invalid later step 在首次 source activation 前失败，subgraph/continuation 请求按支持矩阵拒绝。
- [x] 3.2 以 fixture 证明 teams 纯 validation/normalization 接缝，再按需要暴露调用现有 normalizePlan 的最小入口；验证 native/bridge positive/negative/default facts 一致，不复制 strategy/default/aggregation rules。
- [x] 3.3 复用 scheduler Guardrails/ValidateSpawn 与 native timeout resolver；验证 child depth/active limits、budget completeness、clamp 原生语义及所有 descriptor/binding facts mismatch 无副作用。
- [x] 3.4 建立 terminal-only composition 与 isolated exclusive local child binding；验证共享队列未 enqueue/claim、缺少可证明隔离时拒绝、pause-required binding 预检拒绝，不能用 bridge fanout/claim/retry loop 补能力。
- [x] 3.5 交付全计划 no-side-effect-before-rejection tests；用计数器和真实 store 确认 invalid later task/node、非法策略/预算/能力不会 model/tool/callback/enqueue/publish/claim/commit/diagnostic write。

## 4. 薄委托与 activation 边界

- [x] 4.1 实现 direct runner/composer、teams parallel/vote、flat workflow 的原生 Run/Stream 委托与 tagged native results；验证每次选择一个 owner、保留 errors/aggregation/correlation、不生成第二 terminal/retry/budget owner。
- [x] 4.2 实现隔离 local child 原生 dispatch；验证实际 claim/commit/task/attempt 与 parent refs，Stream 仅转发真实可用 timeline，token-streaming requirement 缺失时明确拒绝。
- [x] 4.3 实现 semantic config/capability drift 检查并保留 source live admission；验证 compile 后 parent budget 缩小/耗尽、readiness/action/host deny、lease 失效及重新编译要求，不以编译快照或 digest 替代权限/余额。
- [x] 4.4 接入归档 160 真实 source callback 与 PlanVersion/PlanDigest host refs；验证 parallel source 独立 observer、source identity/ordering/parent 保真、observer failure 独立，不从结果构造事件或新增诊断入口。
- [x] 4.5 验证 direct input_required 保持原生 pause；terminal-only runner adapter 遇到意外 pause 返回 unsupported continuation 和 checkpoint refs，测试不得转换为 successful task 或自动 resume。

## 5. Integration、replay 与双平台 gate

- [x] 5.1 新增 `integration/host_execution_plan_bridge_test.go`，使用真实 owners 与 fake model/tool；验证 direct/team parallel/vote/flat workflow/isolated child Run/Stream normalized parity，与既有直接 owner 结果/parent refs/budget 对照。
- [x] 5.2 在同一 integration suite 补 child failure/cancel/timeout、native retry 只由 owner 执行、unexpected pause 和 unsupported mailbox/resume/shared queue；验证 rejection counters/store 不变，pause/accepted/native terminal 不等于业务 commit。
- [x] 5.3 在 `tool/diagnosticsreplay` 新增版本化 reference-only plan fixtures/replay；验证 digest drift、native outcome/correlation/budget drift、missing source evidence 和 side-effect-before-rejection 的稳定分类；replay 不执行 callback 或恢复 Run。
- [x] 5.4 新增 `scripts/check-host-execution-plan-contract.sh/.ps1` 并接入 `check-quality-gate.*` 与 `tool/contributioncheck`；验证双平台 fixture selection/result 等价，故意 drift fixture 必须令 gate 失败。
- [x] 5.5 完成 gate 接线后同步 `docs/mainline-contract-test-index.md`；验证每项 spec requirement 可追踪到 unit/integration/replay/gate，不将计划中的 suite 提前列为主线证据。

## 6. 离线示例与对外语义文档交付

- [x] 6.1 实现 `examples/host-execution-plan` 离线 fake model/tool runnable example；验证支持路径、非法 binding 和 unsupported capability 的 expected markers，Run/Stream parity、无 live provider/业务 DB，先文档后代码。
- [x] 6.2 同步 README、orchestration 两层组件文档、module boundaries、integration/best-practices 与 package README；验证实际 API/support matrix/opaque digest trust/terminal-only child/source streaming/rollback 与代码一致，不再保留“待实现”措辞。
- [x] 6.3 同步 `docs/runtime-config-diagnostics.md` 与三份工件 Documentation Impact Assessment/commitment；验证 constructor-only limits、无新增 env/file/hot-update/QueryRuns、RuntimeRecorder 单写，并保证每条声明路径存在且实际改动。
- [x] 6.4 执行 public documentation、example impact、roadmap consistency 和 docs gates；验证全部通过后才勾选依赖文档的实现任务，归档前每项必须具备代码/测试/文档/门禁证据。

## 7. 完整验证与交付

- [x] 7.1 执行受影响包 `go test ./orchestration/... ./integration/... ./tool/diagnosticsreplay ./tool/contributioncheck -count=1`、专用双平台 gate 和 example smoke；在 `verification.md` 记录命令/exit code/证据，未执行项写明原因且不能假称通过。
- [x] 7.2 执行 `go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`、`pwsh -File scripts/check-quality-gate.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、OpenSpec strict validation 与 `git diff --check`；验证通过后进行 contract/docs 自审与风险/回滚核对。
- [x] 7.3 仅在全部实施与文档证据完整后使用 `scripts/openspec-archive-seq.ps1 -ChangeName establish-host-execution-plan-bridge-contract` 归档；同步 roadmap/README/index，再验证门禁，禁止手工重命名 archive。
- [ ] 7.4 按实际授权执行提交、同步最新 master、合并、推送及合并结果验证；验证已合并后才删除本地功能分支及相关 worktree，保留用户缓存与未合并分支；记录交付 commit 和未执行项。

## Example Impact Assessment

新增示例

离线 runnable example、smoke 与实际结果均由本 change 交付。本 change 不修改 agent-modes；若后续范围触及，仍须先补 MATRIX 与模式 README。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/components/orchestration.md`、`orchestration/README.md`、`orchestration/planbridge/README.md`、`docs/runtime-module-boundaries.md`、`docs/guides/integration.md`、`docs/guides/best-practices.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`examples/host-execution-plan/README.md`
- 维护 owner：orchestration 与 host integration maintainers
- 验证证据：OpenSpec strict validation、documentation/example impact gate、roadmap/docs consistency；实施后补齐 plan bridge contract/replay、Run/Stream parity 与离线 example smoke

当前“已同步”指本 change 已在同一功能分支交付并同步的对外 API、limits/defaults、错误分类、支持矩阵、owner 边界、兼容性和回滚说明。新增维护路径已纳入三份工件的一致承诺；缺少实际测试或门禁 exit-code 证据时，仍不得勾选任务、宣称验证完成或归档。

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | orchestration maintainers | owner 边界审计与 docs consistency；禁止第二执行/预算/恢复状态机 |
| components | 修改文档 | `docs/components/orchestration.md`、`orchestration/README.md`；实施新增 `orchestration/planbridge/README.md` | orchestration maintainers | 支持矩阵与实际 owner API 对照、package tests |
| configuration | 修改文档 | 实施同步 `docs/runtime-config-diagnostics.md` | runtime config maintainers | 明确 constructor-only limits、既有配置优先级和无新增 env/file/hot-update 键 |
| contract/API | 修改文档 | 本 change spec；实施同步 `docs/mainline-contract-test-index.md` | contract maintainers | strict validation、逐 requirement 的单测/integration/replay/gate 证据；gate 接入后才更新主线索引 |
| diagnostics | 修改文档 | 实施同步 `docs/runtime-config-diagnostics.md` | observability maintainers | reference-only 错误/结果与 RuntimeRecorder 单写；不新增 QueryRuns 字段 |
| examples | 新增文档 | `examples/host-execution-plan/README.md`、`README.md` | examples maintainers | 文档基线先行，离线 smoke 与 expected markers；不依赖 live provider |
| CLI/integration | 修改文档 | `docs/guides/integration.md`；实施新增 `orchestration/planbridge/README.md` | host integration maintainers | plan/binding 编译、执行前拒绝、源结果透传与回滚示例 |
| best practices | 修改文档 | 实施同步 `docs/guides/best-practices.md` | documentation maintainers | digest 不等于授权、投递不等于完成、replay 不等于 resume 的反例核对 |
| roadmap | 修改文档 | `docs/development-roadmap.md`、`README.md` | roadmap maintainers | openspec list 与 roadmap status consistency；候选移入进行中 |
