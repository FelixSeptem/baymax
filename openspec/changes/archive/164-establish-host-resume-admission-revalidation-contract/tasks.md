## 1. 文档基线与现状回归

- [x] 1.1 对照实际 runner/Host API 固定 source owner、通用 command authorization、进程内恢复与 metadata SPI 的差别，并完善接入指南及示例 README；验证维护路径均存在且 documentation-impact/docs-consistency 门禁通过。
- [x] 1.2 用真实 runner + fake model/tool 写最小 gap 回归：未校验宿主 revision、宽松 key/kind duplicate、in-flight 空结果及第二次 PendingAction；验证测试在旧实现上明确失败且不依赖网络或 sleep。

## 2. 有界宿主重验证合同

- [x] 2.1 在 core/types 定义 host_resume_admission.v1 的 reference-only request/binding/result 与 verifier 接缝，支持可信暂停绑定与当前事实的分离；验证正常、空 identity、非正 plan version、unsupported version、非法字段、256-byte reference、128-byte digest 和 32-KiB envelope 边界单测。
- [x] 2.2 实现 detached snapshot、完整 normalized decision equivalence 与有限 host_resume.* rejection 分类，禁止 raw callback error/body 泄漏；验证每个 identity 字段变化、alias mutation、错误脱敏及无效 callback result 测试。
- [x] 2.3 增加 runner constructor opt-in，未安装时保留合法 native 路径，显式启用却缺 verifier/binding 时 fail-closed；验证默认兼容、nil 配置和安装后不能按请求绕过的正负测试，并同步公开 API 文档。

## 3. Source admission 与并发控制

- [x] 3.1 接入 source 初检、锁外同步 callback、锁内 generation/state/mode/expiry/control/context 重检和唯一 decision commit；验证 callback 中查询/取消不被锁阻塞、panic/error/context cancellation 不消费 checkpoint。
- [x] 3.2 按 decision 区分 confirm execution/write authority 与 deny/timeout cancellation authority，并验证 revision/plan/tool/scope/confirmation/effect eligibility；验证所有 drift 分类、拒绝无模型/工具/effect/iteration/resolution/terminal 增量以及同 key 后续合法重试。
- [x] 3.3 修正 full-identity duplicate：完成后返回保留结果，执行中返回稳定 in-progress，非等价请求拒绝；验证 guard 启用/关闭两种模式、重复请求不再调用 verifier、所有 normalized 字段与 mode 变更均不能得到 duplicate success。
- [x] 3.4 隔离 continuation 完成与下一代 PendingAction checkpoint，保留有限 source retention；通过 barrier 驱动双 confirm、conflict key、cancel/expiry/replacement race 与二次暂停测试，验证单次 commit 且旧结果不覆盖新 checkpoint。

## 4. 原生 Run/Stream 与 Host conformance

- [x] 4.1 完成真实 runner Run/Stream 的合法恢复与拒绝路径 integration，计数 provider/tool/业务 fixture/resolution/terminal；验证 confirm 单次 resolved 后正常完成、deny/timeout 单次 canceled、duplicate/rejection 不加事件，原 iteration/tool results/correlation 得以延续。
- [x] 4.2 将既有 Host run.action.resume 委托到同一 source guard，保持 readiness/Authorizer 和 response-before-event barrier；验证通用授权通过但业务拒绝、不同 message 的 source duplicate、错误反馈及 direct/Host parity。
- [x] 4.3 固定 same-Engine 支持矩阵与进程重启反例，保持 planbridge resumed/team/workflow/mailbox unsupported；验证新 Engine + metadata/same run ID 不触发 provider/tool，新 Run、retry、replay 不消费动态 checkpoint。
- [x] 4.4 使用内存 revision/effect fixture 证明 admission 之后业务 executor 仍需 CAS；验证 admission 后 revision/authority/effect 变动会拒绝业务提交，明确 runtime completion 不是业务 commit，也不承诺 exactly-once。

## 5. Replay、门禁与示例

- [x] 5.1 增加版本化 reference-only fixture 与纯离线 replay/drift 检查，覆盖合法恢复、授权/revision/binding drift、重复/并发、恢复范围和 parity；验证 fixture 不调用 verifier/provider/tool/store 且隐私与负向 drift 测试通过。
- [x] 5.2 新增语义化命名的 check-host-resume-admission-contract.sh/.ps1 并接入既有质量门禁与 contributioncheck；验证正常和故意 drift 的 exit-code/classification 一致，并记录两平台实际执行证据与环境未执行项。
- [x] 5.3 在已完成 README 基线后实现 examples/host-resume-admission 离线真实 runner 示例；验证 authority denied 后合法恢复、revision drift、exact duplicate 和 effect CAS markers，禁止仅输出预置事件来充当 runtime 证据。

## 6. 文档同步与交付验证

- [x] 6.1 按最终 API、limits、默认关闭、错误分类、事件顺序、duplicate 迁移和恢复矩阵同步全部承诺维护文档与 proposal/design/tasks；验证三份 Public Semantic Documentation Commitment 一致、文档真实进入 Git 差异及 documentation/example/roadmap/docs 门禁通过。
- [x] 6.2 运行受影响包单测/integration、专用双平台 gate、openspec strict validation，以及 go test ./...、go test -race ./...、golangci-lint run --config .golangci.yml、pwsh -File scripts/check-quality-gate.ps1、pwsh -File scripts/check-docs-consistency.ps1 和 git diff --check；验证实际 exit-code 并将风险、回滚、未执行项及原因写入 verification.md，禁止以计划替代执行证据。
- [ ] 6.3 完整验证后使用 scripts/openspec-archive-seq.ps1 归档并同步 roadmap/索引；按 AGENTS.md 在获准交付时提交、合并最新 master、验证合并结果、推送并清理已合并功能分支，验证 Git 状态与归档序号无漂移。

## Example Impact Assessment

新增示例

examples/host-resume-admission 已先交付 README 的 semantic anchor、runtime path、expected markers、rollback notes，再实施并验证真实 runner 的可运行离线示例；不修改 agent-modes。任务勾选依据实际代码、测试、文档与门禁证据。

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
