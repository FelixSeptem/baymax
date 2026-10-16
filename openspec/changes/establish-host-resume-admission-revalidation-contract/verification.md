# Host resume admission verification

日期：2026-10-15（包含 2026-10-14 的原始回归证据）。功能分支：`feat/host-resume-admission-revalidation`；起草基准：`7aac3d7`。当前为实现收尾验证；归档、提交、合并和推送状态单独记录，不以代码通过代替交付完成。

## 已取得的证据

| 范围 | 命令/证据 | 结果 |
| --- | --- | --- |
| TDD 原始 gap | 完整 identity 变更、in-flight duplicate、恢复 iteration 回归在旧实现上失败 | 旧实现错误接受 7 类 identity 变化；in-flight err=nil；iteration=1 |
| DTO UTF-8 回归 | `go test ./core/types -run 'TestHostResume(DecisionUTF8\|EnvelopeExactBoundary)' -count=1` | 修复前非法 UTF-8 key 被接受；修复后通过 |
| DTO 正负边界 | `go test ./core/types -run TestHostResume -count=1` | exit=0；包括 256-byte reference、128-byte digest、32-KiB 精确 envelope 与超出一字节 |
| 原生 runner | `go test ./core/runner -run 'TestHostResume(Duplicate\|Concurrent)' -count=1` | exit=0；guard 开/关完整 identity、normalized duplicate、并发唯一 continuation/resolution |
| 新 Run/retry 范围 | `go test ./core/runner -run TestHostResumeNewRunAndRetry -count=1` | exit=0；两种模式均不消费或绕过原 checkpoint |
| Host integration | 真实 Engine 的 source guard/response barrier/business CAS 用例，专用门禁覆盖 | exit=0；通用 Authorizer 通过仍可业务拒绝，duplicate 仍须独立读授权 |
| Replay | 14-case golden fixture、invalid fixture、语义/隐私 mutation suite | exit=0；纯离线，不持有 verifier/provider/tool/store |
| PowerShell 专用门禁 | `pwsh -File scripts/check-host-resume-admission-contract.ps1` | exit=0；故意 drift 子命令 exit=1 / host_resume.replay_drift |
| Bash 专用门禁 | `C:/Program Files/Git/bin/bash.exe scripts/check-host-resume-admission-contract.sh` | exit=0；故意 drift 子命令 exit=1 / host_resume.replay_drift |
| 离线真实 runner 示例 | 两套专用门禁执行 `go run ./examples/host-resume-admission` | 全部 8 个 expected markers 已验证 |
| 全量普通测试 | `go test ./... -count=1 -timeout 15m` | exit=0；后续 UTF-8 与补充回归由最终 quality gate 重新覆盖 |
| 全量 race 首次顺序运行 | `GOMAXPROCS=2 go test -race ./... -timeout 20m -p 1` | exit=0；最终结构整理后仍由 quality gate 覆盖 |
| 结构整理回归 | `GOMAXPROCS=2 go test ./core/runner ./integration -run 'TestHostResume\|TestDynamicAction' -count=1` 与 Git Bash 专用 gate | exit=0；runner、integration、replay、示例与故意 drift 均通过 |

## 完整交付门禁

本轮独立审查发现并修复了两类结果保留问题：公开 pause 的 PendingAction 不再引用 source checkpoint；完成与 duplicate 结果隔离 warnings、tool errors、terminal、嵌套 map/slice/array/pointer/struct details。已提交准入后的 continuation error 使用每次独立创建的 `DynamicActionContinuationError`，保留 `errors.Is`/`errors.As`/`Unwrap`，Host 的首次及 duplicate command response 均保持 accepted，原 provider error 不写入 admission ReasonCode。

新增回归 `TestHostResumePublicPauseCannotMutateSource`、`TestHostResumeRetainedResultIsDetached`、`TestHostResumeRetainedFailureAndCollections`、`TestHostResumeContinuationFailureKeepsAdmissionAccepted` 在修复前观察到失败，修复后 `GOMAXPROCS=2 go test ./core/runner ./integration -run 'TestHostResume(PublicPause|Retained|ContinuationFailure)' -count=1 -p 1 -timeout 2m` 已 exit=0。指南、诊断参考及 CHANGELOG 同步解释上述公开语义。

受影响包集合 `GOMAXPROCS=2 go test ./core/types ./core/runner ./host/... ./integration/... ./tool/diagnosticsreplay ./tool/contributioncheck -count=1 -p 1 -timeout 15m` 已 exit=0；strict OpenSpec validation 和 `git diff --check` 已 exit=0。完整质量门禁中普通测试、race、lint（0 issues）、semantic stability 均通过，但 diagnostics 性能子门禁失败，因此整体 quality exit=1，tasks 6.2 保持未勾选。

性能失败已做单变量复现：`GOFLAGS=-p=1`、相同 Go 缓存、相同代码与默认 200ms/count=5/median/阈值下，`GOMAXPROCS=2 pwsh -NoProfile -File scripts/check-diagnostics-query-performance-regression.ps1` exit=1，QueryMailbox 中位数 2,544,476 ns/op、p95 8,350,500 ns/op，MailboxAggregates 中位数 885,838 ns/op；取消 GOMAXPROCS 覆盖、恢复机器默认 16 线程后，同一命令 exit=0，分别为 1,124,743 ns/op、3,001,700 ns/op 和 510,520 ns/op，两项 allocs/op 均保持 48/259。历史原始样本使用 `-16`。降低执行线程数的内存控制设置会影响该性能比较；没有修改实现、性能基线或阈值。完整质量门禁以默认执行线程数、`GOFLAGS=-p=1`、`BAYMAX_QUALITY_GATE_PARALLELISM=1` 重新运行：`pwsh -NoProfile -File scripts/check-quality-gate.ps1` 最终 exit=1。该次运行中的 `go test ./...`（exit=0）、`go test -race ./...`（exit=0）、`golangci-lint --config .golangci.yml`（`0 issues`）、a64 semantic stability 与 a64 performance regression（包含 `[diagnostics-query-bench] passed`）均已通过；最终被 strict `govulncheck ./...` 的 exit=3 阻断。

对该阻断的复现与归因如下：沙箱内的 `govulncheck ./...` 首先因共享 Go build cache 的文件缺失/访问拒绝而无法加载包；在允许访问真实 cache 后，扫描器可稳定复现实际可达漏洞，而非 cache 误报。当前扫描器为 `govulncheck@v1.1.4`，Go 为 `go1.26.8`。报告至少包括 `GO-2026-6617/6612/6611/6610/6603` 的 `golang.org/x/net@v0.55.0`（修复版本 `v0.60.0`），以及 9 个标准库可达漏洞（`net/http`、`net/textproto`、`crypto/tls`、`os`、`html/template`，均要求 `go1.26.9`）。verbose 输出还列出需后续评估的 `google.golang.org/grpc@v1.83.1`、`github.com/go-viper/mapstructure/v2@v2.3.0`、`golang.org/x/text@v0.39.0`、`golang.org/x/crypto@v0.51.0`。`git diff master -- go.mod go.sum` 为空，且 `master` 也固定 `go 1.26.0` 与 `golang.org/x/net v0.55.0`：这些并非本提案引入，但严格门禁不允许将其视为通过。没有降级 `BAYMAX_SECURITY_SCAN_MODE`、禁用扫描或修改基线/阈值。

最终独立收尾检查：`pwsh -NoProfile -File scripts/check-docs-consistency.ps1` exit=0；`openspec validate establish-host-resume-admission-revalidation-contract --strict` exit=0；`git diff --check` exit=0；`pwsh -File scripts/check-host-resume-admission-contract.ps1` exit=0（此前记录）；`C:/Program Files/Git/bin/bash.exe scripts/check-host-resume-admission-contract.sh` 在沙箱内因 Git Bash signal pipe 的 Win32 error 5 失败、允许访问后 exit=0。Bash 门禁覆盖 core/types、core/runner、host、integration、diagnostics replay、contributioncheck 和离线示例，故意 drift 仍为 exit=1 / `host_resume.replay_drift`。

第一次同时运行全量普通测试、race、lint 与 Bash gate 时触发机器内存不足：lint 报 `out of memory allocating heap arena map`，race 报 `runtime: cannot allocate memory`，两者退出非零。改为顺序执行，设置 `GOMAXPROCS=2`，race 使用 `-p 1`；没有放宽 contract 或跳过检查。

后续 lint 找到两处 gocritic 风格问题，已修正；精确 envelope 边界测试继续在调用时读取最新 DTO，避免方法值捕获旧副本。第一次 full quality gate 在 step 15 的文件行数门禁失败（runner.go 5919 行，基线 5916）；将 checkpoint 和准入字段组织为同包嵌入的 dynamicActionState 后，该文件为 5915 行，行数门禁 exit=0，未提高债务基线。测试在沙箱读取外部 Go 缓存时曾被拒绝，按授权提权重跑通过。

## 证据对应与边界

- `core/types/host_resume_admission_test.go`：版本、必填字段、UTF-8、正 plan version、大小边界、可信 binding/current drift 与 detached snapshot。
- `core/runner/host_resume_admission_test.go`：拒绝保留 key/checkpoint、nil/panic/error/cancellation fail-closed、锁外回调、commit 前取消/过期/generation 重检、并发 commit、完整 duplicate 与二次 pause。
- `integration/host_resume_admission_test.go`：Run/Stream 的 Host source guard、response-before-event barrier、独立结果读授权及 admission 之后的业务 CAS。
- `tool/diagnosticsreplay/host_resume_admission_test.go` 与双平台 gate：版本化 fixture、故意 drift、Run/Stream parity、恢复范围与无执行 owner。
- planbridge resumed/team/workflow/mailbox 继续 unsupported；same-Engine continuation 支持不等于跨进程 restore。新 Engine 不调用 provider/tool/verifier。

## 风险与回滚

宿主 verifier 必须读取可信暂停记录与当前事实；一致的 digest 不等于授权。callback 为锁外同步调用，需要合作处理 context；不强制中断不合作代码。准入与业务写入不是分布式原子事务，executor 仍须 CAS authority/revision/effect；runtime completed 不表示业务 effect 已提交，不承诺 exactly-once。

审查保留的次要边界：source checkpoint 初检后，已取消的 active control 可能仍使 verifier 被调用一次；提交前的 control/context 锁内重检会拒绝恢复，不消费 checkpoint 或产生 continuation/resolution。若 verifier 本身先拒绝，调用方可能先收到 verifier 的有限拒绝分类。callback 必须保持只读且无业务 effect，不能以调用次数作为 admission 证据。

默认不启用 guard；使用 `runner.WithHostResumeVerifier` 显式启用后 nil/error/无效结果 fail-closed。完整 duplicate 等价性是默认 source 行为修正，调用方需保持全部 normalized 字段和 mode 一致；checkpoint ID 不透明。回滚可恢复旧版本或停止恢复入口；要求业务重授权的宿主不得通过静默移除 guard 降低保障。保留暂停记录与业务 effect/outbox；已发生写入不能由 runtime 回滚撤销。

## 工作区与生命周期

预存 `.artifacts/a64/*.json` 两项改动及未跟踪 Go 缓存属于用户，保留且不纳入本 change。当前未归档、提交、合并或推送；严格安全门禁尚未通过，tasks 6.2 与 6.3 保持未勾选。解除阻断需要在独立依赖/Go 工具链安全更新中升级到 Go `1.26.9`、评估并更新受影响模块，随后重新执行完整质量门禁；不得以 warn/disabled 安全扫描替代 strict 通过证据。
