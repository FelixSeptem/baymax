## Why

宿主任务的权限、业务 revision 和写范围可能在 `input_required` 暂停期间变化；当前 checkpoint 校验和通用 Host authorization hook 没有定义它们与原始执行绑定的重验证合同。归档 158/159/160/162 已提供 native resume、resolution/terminal、真实事件与计划 identity，当前需要在源 runner 提交恢复 decision 前补齐宿主准入接缝。

## What Changes

- 新增显式 opt-in 的 `host_resume_admission.v1` reference-only 合同，在 runner 原生 `ResumeDynamicAction` admission 边界重验证宿主绑定，直接 API 与 Host command 使用同一 source 路径。
- 将 authoritative paused checkpoint 与宿主保存的 plan identity/version/digest、authority/tool/write-scope digest、业务 revision、confirmation/effect references 绑定；宿主读取当前事实并决定 confirm/deny/timeout 的权限，Baymax 不解释业务 policy。
- 拒绝缺失绑定、失效权限/revision、plan/checkpoint mismatch、无效重验证结果及取消的 admission；拒绝不消费 checkpoint、不记录已接收 decision、不发 resolution/terminal、不执行 provider/tool/effect。
- 在 runner 原有 checkpoint owner 内完成锁外 callback 与锁内再次校验/单次 decision 提交，不建立第二套 pending/resume 状态机。精确 duplicate、in-flight duplicate、冲突 key、第二次动态暂停和 cancel/expiry race 均有确定性 contract。
- 保持归档 159 事件语义：被接收的 confirm/deny/timeout 各发一次 resolved；deny/timeout 才发 canceled，confirm 继续原生 loop。duplicate 不重新触发业务授权回调或执行，但传输/结果读取权限仍由宿主验证。
- 明确当前 runner continuation 仅在原 Engine 内恢复；仅有 metadata SPI、相同 run ID、snapshot 或 event replay 不构成跨进程恢复。本 change 不开放 planbridge resumed/team/workflow/mailbox continuation。
- 交付真实 runner + fake model/tool 的 conformance、版本化离线 replay、双平台 gate、宿主中立示例和同步维护文档；修正 roadmap/README 对归档 162 的旧状态。

## Capabilities

### New Capabilities

- `host-resume-admission-revalidation`：宿主绑定与当前业务准入、source commit 顺序、并发/幂等、恢复支持边界及离线 conformance。

### Modified Capabilities

- `dynamic-action-gate-and-native-run-resume`：强化完整 normalized decision/checkpoint/mode 的重复等价性，定义 opt-in 宿主重验证先于 source decision commit。
- `embedded-host-command-response-and-event-correlation-contract`：新增既有 action-resume command 与 source revalidation 的委托合同，保留 command admission/event 分离及通用 authorization。

## Impact

实现修改 `core/types`、`core/runner`、`integration`、`tool/diagnosticsreplay`、`tool/contributioncheck`、双平台专用 gate 和 `examples/host-resume-admission`。既有 `host/action_resume.go` 已直接委托同一 runner source，无需增加平行接线。公开入口为 `runner.WithHostResumeVerifier` 与 `types.HostResumeVerifier`/函数适配器；不新增 adapter store，不引入 Danny/Stele/业务数据库/Provider SDK 依赖。

审计与回归证据：runner 的 `dynamicCheckpoints` 保留 request/outcomes/control 的进程内 continuation；`RunCheckpointStore` 定义 metadata SPI，但 runner 没有接线到该 SPI。Host 通用 readiness/authorization 在 command 委托前执行，action-resume 直接调用 runner。旧实现仅比较 key 与 decision kind、in-flight 返回空成功、resume iteration 从头计数的 gap 已由真实 runner 回归钉住并修正；未独立审计 Danny 仓库。

未启用 guard 的合法 native resume 继续有效；强化 mismatched duplicate 的 fail-closed 判定会影响依赖宽松 key/kind 匹配的调用方，必须明确迁移说明。启用 guard 后绑定/验证器缺失必须拒绝，不能静默退回未验证路径。无 runtime 配置键、QueryRuns 字段、后台 worker 或持久化迁移。

## Risks and Rollback

风险包括 checkpoint callback 后漂移、callback 重入/不合作、两个 confirm 并发、旧 checkpoint duplicate 覆盖新 pause、授权与业务 commit 间的竞态，以及把 digest 当作权限。source 锁内重检与宿主 effect/revision CAS 分别守住 runtime 与业务边界；本 change 不承诺两者的分布式原子事务。

回滚停用新 guard 并回到显式 source-native API；要求恢复重授权的宿主应停止接收恢复而非静默绕过。保留暂停 checkpoint、application execution record、effect/outbox 事实；不能通过回滚撤销已经发生的写入。

## Example Impact Assessment

新增示例

`examples/host-resume-admission` 使用真实 runner、fake model/tool、内存业务 revision/effect fixture 演示拒绝后 checkpoint 仍可被合法确认恢复，以及 exact duplicate 不重复 resolution/effect。已先建立 README 的 semantic anchor、runtime path、expected markers、rollback notes；本 change 不修改 agent-modes。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`CHANGELOG.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/guides/host-resume-admission.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`examples/host-resume-admission/README.md`
- 维护 owner：runner、host integration 与 documentation maintainers
- 验证证据：`verification.md` 记录真实 runner Run/Stream、Host integration、并发重验证、fixture/replay、双平台专用 gate、OpenSpec strict validation、documentation/example/roadmap/docs 与完整 Go/race/lint/quality 门禁的实际命令、退出结果和环境限制

“已同步”指同一功能分支中的维护文档已按实际实现更新：`runner.WithHostResumeVerifier`、`types.HostResumeVerifier`/函数适配器、reference-only DTO、默认关闭、256-byte reference/128-byte digest/32-KiB envelope、有限拒绝分类、完整 duplicate identity 和 same-Engine 恢复矩阵。后续公开语义变化仍须同步维护文档及三份工件；任务勾选与归档以 `verification.md` 的实际通过证据为准。

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runner/host maintainers | guard 不拥有执行/业务状态，callback 锁外运行，RuntimeRecorder 单写 |
| components | 新增文档 | `docs/guides/host-resume-admission.md` | runner/host maintainers | checkpoint owner、绑定接缝、并发重检和支持矩阵对照实际 API |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime config maintainers | constructor-only opt-in，不新增 env/file/hot-update 键，不改变配置优先级 |
| contract/API | 修改文档 | `docs/mainline-contract-test-index.md`、`docs/guides/host-resume-admission.md` | contract maintainers | delta requirement 与正负边界/integration/replay/gate 对照，规划与已交付分开 |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | observability maintainers | bounded reason/reference，不记录业务正文，单写入口，无新增 QueryRuns 字段 |
| examples | 新增文档 | `examples/host-resume-admission/README.md` | examples maintainers | 文档先行后已实现真实 runner 示例，覆盖合法恢复/revision drift/duplicate/effect CAS markers |
| CLI/integration | 修改文档 | `docs/guides/integration.md`、`docs/guides/host-resume-admission.md` | host integration maintainers | 既有 action-resume command 接线，不增加 command kind 或 wire vocabulary |
| best practices | 新增文档 | `docs/guides/host-resume-admission.md` | documentation maintainers | digest 不等于授权，恢复不是 retry/replay，effect 提交仍须宿主 CAS |
| roadmap | 修改文档 | `docs/development-roadmap.md`、`README.md` | roadmap maintainers | archive 162 状态修正，active slug 与 openspec list 一致 |

## Validation Strategy

提案阶段：`openspec validate establish-host-resume-admission-revalidation-contract --strict`；`pwsh -File scripts/check-openspec-documentation-impact.ps1`；`pwsh -File scripts/check-openspec-example-impact-declaration.ps1`；`pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1`；`pwsh -File scripts/check-docs-consistency.ps1`；`git diff --check`。

实施阶段：受影响包 `go test ./core/types ./core/runner ./host/... ./integration/... ./tool/diagnosticsreplay ./tool/contributioncheck -count=1`，专用 `scripts/check-host-resume-admission-contract.sh/.ps1`，以及 AGENTS.md 的完整 Go/race/lint/quality/docs 门禁。实际 exit-code、风险、回滚与未执行项见 verification.md；实现状态与归档/合并状态分别记录。
