# Host resume admission：离线原生恢复示例

状态：已实施；README 基线先于代码交付。运行 `go run ./examples/host-resume-admission`，每个 marker 都在真实 runner/fake model/tool 与内存宿主 fixture 的行为断言之后产生。公开 API 与恢复范围见 [接入指南](../../docs/guides/host-resume-admission.md)。

## Semantic anchor

`host_resume_admission.v1`：source checkpoint identity + 可信宿主 execution binding + 当前业务准入，必须先于 native decision commit。guard rejection 不消费 checkpoint，不触发 provider/tool/effect 或 resolution/terminal；accepted confirm/deny/timeout 保持原生事件语义。duplicate 使用完整 normalized identity，业务 effect 提交独立 CAS。

## Runtime path

使用真实 `runner.Engine` 与 `modelcapability.Wrap` 的 fake model、本地 fake tool。工具产生 PendingAction，原 Run/Stream 暂停；宿主内存 fixture 保存业务 revision、authority、plan/scope/confirmation/effect references；`runner.WithHostResumeVerifier` 读取并重验证 fixture，`ResumeDynamicAction` 继续同一 source 路径。示例的 EventHandler 直接计数 source resolution/canceled events，不从 RunResult 合成事件。Host `run.action.resume` 同入口及 response-before-event 由 integration 测试覆盖。

示例无网络、真实凭证或数据库。内存 effect fixture 只证明宿主 commit gate 的责任，不为 Baymax 新增生产 effect store。第一版只演示原 Engine continuation，不使用 planbridge resumed 或 metadata-only 新进程恢复。

## Expected markers

| 场景 | 已实施断言/marker |
| --- | --- |
| authority 暂时拒绝 | `authority_denied`；checkpoint 仍 input_required，provider/tool/effect 与 resolved 计数无增量 |
| 恢复匹配权限后合法 confirm | `resume_accepted`、`completed`；同 correlation 与保留的工具结果，resolved 为 1 |
| revision 在 pause 后改变 | `revision_drift`；保持暂停，不能靠修改请求 digest 继续执行 |
| 完成后 exact duplicate | `duplicate_no_effect`；verifier/resolution/effect 计数不增加 |
| 执行中 exact duplicate | `resume_in_progress`；不返回空 completed，也不继续第二次 |
| admission 后业务 revision 漂移 | `effect_cas_rejected`；runtime 准入不能代替宿主业务提交验证 |
| Run/Stream parity | `run_stream_parity`；归一化 admission/resolution/terminal/effect 语义一致 |

## Contracts、gates 与 replay

正向/负向/边界单测位于 `core/types/host_resume_admission_test.go` 与 `core/runner/host_resume_admission_test.go`，Host 与业务 CAS 见 `integration/host_resume_admission_test.go`；版本化 fixture/replay 位于 `tool/diagnosticsreplay/host_resume_admission*`。运行 `pwsh -File scripts/check-host-resume-admission-contract.ps1` 或 `bash scripts/check-host-resume-admission-contract.sh`，两者均验证故意 drift exit=1 / `host_resume.replay_drift` 和全部示例 markers。示例测试：`go test ./examples/host-resume-admission -count=1`。

marker 名称不代替真实执行断言。公开 API、错误、limits 或支持范围变化须同时更新此 README、指南及 OpenSpec 三份工件。完整门禁与两平台实际证据见 change 的 verification.md。

## Rollback notes

停用 guard 回到既有 source-native API；业务要求重授权时停止恢复请求，禁止静默绕过。保留 checkpoint/application/effect/outbox 记录；不能用 runtime 回滚撤销已提交 effect。此目录不涉及 agent-modes，也不修改其矩阵。
