# Strict security vulnerability remediation verification

日期：2026-10-16。分支：`feat/security-strict-vulnerability-remediation`；基准：`master` / `7aac3d7`。本记录区分修复前 advisory、环境访问失败和修复后的严格门禁；不以 warn、disabled scan 或单个测试替代完整质量门禁。

## 修复前严格扫描证据

| 项目 | 命令 | 实际结果 |
| --- | --- | --- |
| Go toolchain | `go version` | exit=0；`go1.26.8 windows/amd64` |
| Scanner identity | `govulncheck -version` | exit=0；`govulncheck@v1.1.4`、DB `https://vuln.go.dev` |
| Strict reachable scan | `govulncheck ./...` | exit=1；在可访问真实 Go build cache 的环境中复现实际 advisory，不是 cache/权限失败 |

扫描报告当前代码受 9 个标准库漏洞影响，涉及 `net/http`、`net/textproto`、`crypto/tls`、`os` 和 `html/template`，修复版本均为 Go `1.26.9`。报告也显示 `golang.org/x/net@v0.55.0` 的 `GO-2026-6617/6612/6611/6610/6603`，其修复版本为 `v0.60.0`；调用链包括 HTTP/2 transport/framer 路径。扫描另列出 imported/required-but-not-called 的 advisory，尚未作为本 change 的升级依据。

在受限沙箱中，扫描可能以 Go build cache 文件不存在或访问拒绝失败；该环境错误不能解释或掩盖上表在真实 cache 上稳定出现的 advisory exit=1。Windows Git Bash 也可能因 signal-pipe 的 Win32 error 5 失败；双平台门禁必须记录相同命令在具备必要访问权限后的实际结果。

## 最小依赖图与直接验证

`go.mod` 保留语言版本 `go 1.26.0`，声明 `toolchain go1.26.9`。`go version` 实际返回 `go1.26.9 windows/amd64`（exit=0）。现有 CI 均以 `go-version-file: go.mod` 读取声明，无需为各 job 重复写死版本。

| 模块 | 修复前 | 修复后 | 升级依据 |
| --- | --- | --- | --- |
| golang.org/x/net | v0.55.0 | v0.60.0 | 原始严格扫描报告的修复版本 |
| golang.org/x/crypto | v0.51.0 | v0.57.0 | x/net v0.60.0 的最低 require |
| golang.org/x/sys | v0.45.0 | v0.48.0 | x/net 与 x/crypto 的最低 require |
| golang.org/x/text | v0.39.0 | v0.42.0 | x/net 与 x/crypto 的最低 require |
| golang.org/x/sync | v0.21.0 | v0.23.0 | x/text v0.42.0 的最低 require |

`go get golang.org/x/net@v0.60.0` 和 `go mod tidy` 均 exit=0；`go mod graph` 证实上述最小版本要求。`go.sum` 同步清理旧版本并补齐依赖/测试图的校验值，不存在 blanket latest 升级。

修复后直接执行 `govulncheck ./...`，exit=0：`No vulnerabilities found`，`Your code is affected by 0 vulnerabilities`。仍报告 imported-but-not-called 的 2 个 finding 和 required-but-not-called 的 1 个 finding；未将它们用作继续扩大升级范围的依据。

## 文档与治理验证

- `openspec validate remediate-strict-security-vulnerabilities --strict`：exit=0；明确接受 `.openspec.yaml` 的 `skip_specs: true`，已有 strict scan 合同未变。
- documentation-impact、roadmap status、example-impact declaration 三项 PowerShell gate：均 exit=0。
- `go test ./tool/contributioncheck -run TestReleaseStatusParityDocsConsistency -count=1`：exit=0。
- `pwsh -NoProfile -File scripts/check-docs-consistency.ps1`：exit=0。
- `git diff --check`：exit=0。

首次 docs/full gate 在 roadmap 的 active 条目解析处 exit=1：额外的版本反引号被既有解析器纳入提案名。仅修正文档条目格式后，上述现有状态一致性测试与 docs gate 已通过；没有修改解析器或门禁策略。首次受限测试另出现标准库 cache 不可见，允许相同命令访问真实缓存后通过。

## 完整验证

PowerShell 完整门禁 `pwsh -NoProfile -File scripts/check-quality-gate.ps1` 已 exit=0，日志保留在 `.artifacts/security-remediation/quality-powershell-fixed.log`。2026-10-16 07:03:23 开始，77 个步骤全部通过，总耗时 3101.96 秒；最终 strict `govulncheck` 报告 `No vulnerabilities found`、可达漏洞为 0。执行参数仅控制资源和超时预算：`GOFLAGS=-p=1`、gate parallelism=1、total=14400s、step=3600s；保持默认 16 CPU，显式 strict/enabled，测试范围、baseline、性能阈值均不变。

| 完整门禁内命令/范围 | 实际结果 |
| --- | --- |
| `go test ./...` | exit=0；434.09s |
| `go test -race ./...` | exit=0；717.91s |
| `golangci-lint run --config .golangci.yml` | exit=0；0 issues；43.25s |
| a64 semantic stability | exit=0；95.96s |
| a64 performance regression | exit=0；211.60s；context、diagnostics、multi-agent 及文件持久化 benchmark 均通过 |
| full-chain example smoke | exit=0；`FULL_CHAIN_SUCCESS`，Run/Stream 对齐 |
| final strict `govulncheck ./...` | exit=0；可达漏洞为 0；2 个 imported-but-not-called 和 1 个 required-but-not-called finding 未扩大修复范围 |

Git Bash 完整入口 `C:/Program Files/Git/bin/bash.exe scripts/check-quality-gate.sh` 在受限沙箱内因 signal pipe Win32 error 5 返回 `-1073741502`；相同命令允许正常进程/cache 访问后运行，日志 `.artifacts/security-remediation/quality-bash.log`，实际 exit=1。2026-10-16 09:03:21 开始，1895.02 秒后在第 76 步性能门禁失败；前 75 步均通过（ordinary tests 276.45s、race 312.94s、lint 0 issues、semantic stability 通过）。失败项为 `BenchmarkRuntimeRecorderRunFinishedSandboxEnriched`，5 次 ns/op 样本为 176540、188049、247990、278689、290609，中位数 247990，较基线 199542 慢 24.2796%，超过原有 12% 阈值；其 p95 与 allocs 均通过。此轮未执行 full-chain smoke 与最终 strict scan，不能算作完整通过。

随后未修改代码、baseline、阈值或 CPU 设置，单独执行 `scripts/check-diagnostics-query-performance-regression.ps1`，exit=0，日志 `.artifacts/security-remediation/diagnostics-performance-investigation.log`。该项样本为 184318、182520、226220、222822、227122，中位数 222822，较基线慢 11.6667%；全部 diagnostics 指标通过。此前 PowerShell 完整门禁同项中位数为 174395。这些结果显示测量波动，尚不足以确定具体环境原因，且单项复核不能替代完整门禁。

保持相同资源/超时参数的第二次完整 Git Bash 复核（`.artifacts/security-remediation/quality-bash-revalidation.log`）也实际 exit=1：2026-10-16 09:42:32 开始，2081.97 秒后在第 76 步失败。前 75 步均通过，ordinary tests 302s、race 326.61s、lint 0 issues（19.9s）、semantic stability 121.28s。普通 recorder ns/op 样本为 272247、194991、237175、185325、202285，中位数 202285（+16.6452%）；sandbox-enriched 样本为 231716、226327、227644、227570、268976，中位数 227644（+14.0833%）；两项均超过 12%，p95/allocs 仍通过。此轮同样没有执行 full-chain smoke 和最终 strict scan。

### 受控性能诊断与延长采样复核

停止重复默认完整门禁后，在独立 detached master worktree（`7aac3d7`）与修复分支顺序执行相同 recorder benchmark；两边均显式使用 Go `1.26.9`、相同 cache、`GOFLAGS=-p=1`、默认 16 CPU，先用 200ms/count=5，再用 1s/count=5。只有依赖图和采样时长作为对照变量，没有修改 recorder、benchmark、门禁脚本、baseline 或阈值。

| 依赖图 / 采样 | 普通 recorder 中位数 ns/op | sandbox-enriched 中位数 ns/op | 日志（均在 `.artifacts/security-remediation/`） |
| --- | --- | --- | --- |
| master / 200ms | 189725 | 208339 | `performance-master-control.log` |
| patched / 200ms | 202387 | 205406 | `performance-patched-control.log` |
| patched / 1s | 188877 | 207317 | `performance-patched-long-sample.log` |
| master / 1s | 189677 | 218868 | `performance-master-long-sample.log` |

四次 benchmark 命令均 exit=0（benchmark 命令本身不执行阈值判定）；两边 allocs/op 均为 2232/2714。长采样中 patched 未显示比 master 更慢，短采样存在明显波动，但仍不能确定具体调度/环境原因或排除所有性能影响。不能以这些单项数据宣称默认完整门禁通过。

基于此诊断，通过既有环境参数 `BAYMAX_DIAGNOSTICS_QUERY_BENCH_BENCHTIME=1s` 延长 diagnostics 每次测量，保持 count=5、中位数、原 baseline、12% ns/allocs 阈值和 15% p95 阈值，重新执行完整 Git Bash 入口；其他资源/超时和 strict/enabled 设置不变。该设置增加测量工作量，不改变产品或门禁代码。日志 `.artifacts/security-remediation/quality-bash-long-sampling.log`；2026-10-16 10:26:55 开始，2361.9 秒后 exit=1，前 75 项（含普通测试 312.45s、race 351.37s、lint 0 issues、semantic stability 118.89s）通过，第 76 项性能失败。普通 recorder 样本 166421、163669、195068、193196、204745 ns/op，中位数 193196（+11.4042%，通过）；sandbox-enriched 样本 244001、259854、938718、714208、326245 ns/op，中位数 326245（+63.4969%，失败）。两者 p95/allocs 均通过，未运行第 77 项 full-chain 与最终 strict scan。默认 200ms 的两次完整失败仍作为验证限制保留。

完整门禁停止后、没有同时运行的 benchmark 时，独立运行 `pwsh -NoProfile -File scripts/check-diagnostics-query-performance-regression.ps1`（`GOFLAGS=-p=1`、`BAYMAX_DIAGNOSTICS_QUERY_BENCH_BENCHTIME=1s`），exit=1；普通 recorder 样本 166476、171360、202446、198068、207294 ns/op，中位数 198068（+14.2136%，失败），sandbox-enriched 样本 242122、220536、216511、217123、232014 ns/op，中位数 220536（+10.5211%，通过）。一次误启动的并行 benchmark 已中断，未作为有效证据；本次为其结束后的顺序复核。性能波动的具体原因仍未知，不能以对照 benchmark 或 PowerShell 路径通过宣称 Git Bash 完整门禁通过。安全维护任务 3.1/3.3 及归档、合并、推送仍待完整可接受的门禁证据。

进一步核对确认修复分支相对 master 的 `integration/`、`runtime/`、`observability/`、诊断性能脚本及基线均无差异；当前两个 recorder benchmark 的分配数在 master 与修复分支相同，分别为 2232/2714 allocs/op，而 2026-04-08 的原始基线分别为 2028/2510。归档 160 的验证记录也曾在未涉及 recorder wiring 的变更中记录 sandbox-enriched recorder 单次超阈值和后续通过。以上可以限定安全修复的直接变更范围，但无法证明当前主机的耗时差异完全由环境引起，也不能充当门禁通过证据。

以仓库已有参数组合 `BAYMAX_DIAGNOSTICS_QUERY_BENCH_BENCHTIME=500ms`、`BAYMAX_DIAGNOSTICS_QUERY_BENCH_COUNT=9` 顺序执行诊断性能脚本，实际 exit=1：普通 recorder 中位数 194900 ns/op，相对原基线 173419 为 +12.3868%，超过未变的 12% 阈值；sandbox-enriched 中位数 215414 ns/op，相对原基线 199542 为 +7.9542%，通过。两项的 p95 和 allocs 均通过。更长采样和更多样本未能稳定消除失败，因此不再据此重试完整 Git Bash 门禁；此轮没有修改 baseline、阈值、采样默认值或检查逻辑。Docker Linux 引擎可连接，但本机没有预置 Go 镜像；没有执行 Linux 门禁，不能将 Docker 可用性当成验证结果。

历史工作量核对：`416b931`（2026-04-08 的基线提交）到当前 `master`，`integration/benchmark_test.go` 中这两项 recorder fixture 未变，但 `runtime/diagnostics/store.go` 的 `RunRecord` 字段由 357 增至 447（其中 string 字段由 161 增至 235）；`observability/event/runtime_recorder.go` 的 `run.finished` 投影也增加了字段，且写入仍经过 `applyCardinalityGovernance` 的逐字段反射检查。当前 master 与 patched 的 2232/2714 allocs/op 均比旧基线 2028/2510 高 204；这证明旧样本与当前主分支的结构和分配量不一致，尚不能仅凭字段数量归因全部 204 次分配，也不能排除真实性能回归。

后续待办（独立性能维护提案，不属于本安全修复）：在固定的最新 master 提交、Go 版本、CPU/系统和采样参数下重复采集两项 benchmark，与历史提交及 patched 分支做同环境对照，分析 cardinality/recorder 热点、耗时波动和新增分配，记录原始样本与异常解释。先评估是否需要优化代码；仅当稳定主分支样本证明旧 baseline 失配，且符合 `docs/performance-policy.md` 时，才在独立提案中修改对应 baseline 并运行双平台性能及完整质量门禁，保留原回归阈值。当前 Git Bash 完整门禁仍 exit=1，安全任务 3.1/3.3 不勾选、不归档或合并。

更新上述记录后再次运行 `pwsh -NoProfile -File scripts/check-docs-consistency.ps1`、`openspec validate remediate-strict-security-vulnerabilities --strict` 和 `git diff --check`，均 exit=0；直接 `govulncheck ./...` 再次 exit=0，可达漏洞为 0（另有 2 个 imported-but-not-called、1 个 required-but-not-called）。这些独立通过项不替代失败的完整 Git Bash 门禁。

### Runtime recorder baseline calibration (user-authorized)

在获得直接更新 baseline 的授权后，未调整 12% ns/allocs、15% p95 阈值、默认 `200ms` / `count=5` 参数或任一检查脚本。为使新值与门禁参数一致，在受控的 `master` `7aac3d7969a20e904cc83694b6f3082a8c6f308a` 源码快照上，以 Go `1.26.9`、AMD Ryzen 7 7840HS w/ Radeon 780M Graphics（16 CPU）和 `GOFLAGS=-p=1` 运行两个 recorder benchmark，使用 `-benchtime=200ms -count=11` 并取每个指标的中位数。

| Benchmark | ns/op samples median | p95-ns/op samples median | allocs/op median | Refreshed baseline |
| --- | ---: | ---: | ---: | --- |
| `BenchmarkRuntimeRecorderRunFinished` | 326370 | 1070000 | 2232 | `RUN_FINISHED_{NS_OP,P95_NS_OP,ALLOCS_OP}` |
| `BenchmarkRuntimeRecorderRunFinishedSandboxEnriched` | 334217 | 1009100 | 2714 | `RUN_FINISHED_SANDBOX_ENRICHED_{NS_OP,P95_NS_OP,ALLOCS_OP}` |

两项旧值分别为 `173419/1511600/2028` 与 `199542/1519100/2510`。刷新仅覆盖已确认失配的 recorder 基线；其余四项 diagnostics-query 基线原样保留。`docs/performance-policy.md` 与 baseline 文件均记录提交、Go、CPU、采样参数、样本数和中位数。编译器 escape analysis 还显示 `applyCardinalityGovernance` 会将 `RunRecord` 移到 heap，并在 governance 的 map/slice 路径产生逃逸；该观察说明需保留性能回归门禁，但不是修改阈值的依据。

刷新后，`$env:GOFLAGS='-p=1'; pwsh -NoProfile -File scripts/check-diagnostics-query-performance-regression.ps1` exit=0。完整 PowerShell 入口随后在 2026-10-16 13:33:50 开始，`pwsh -NoProfile -File scripts/check-quality-gate.ps1` exit=0：77 步均完成，总耗时 2521.18 秒。关键结果为 `go test ./...` exit=0（355.96s）、`go test -race ./...` exit=0（432.98s）、`golangci-lint run --config .golangci.yml` exit=0（0 issues）、A64 performance exit=0、full-chain smoke 的 `FULL_CHAIN_SUCCESS`，以及最终 strict `govulncheck ./...` exit=0（0 reachable vulnerabilities；仍有 2 imported-but-not-called 和 1 required-but-not-called finding）。完整门禁中的 diagnostics recorder 中位数为普通 `156204 ns/op` / `1002300 p95-ns/op` / `2232 allocs/op`，sandbox-enriched `221604 ns/op` / `1018600 p95-ns/op` / `2714 allocs/op`，均在未变阈值内。

Git Bash 完整入口随后以相同的 `GOFLAGS=-p=1`、parallelism=1、total=14400s、step=3600s 参数执行 `C:/Program Files/Git/bin/bash.exe scripts/check-quality-gate.sh`，实际 exit=0。Windows Bash wrapper 完成其 shell 入口检查后委托 PowerShell 执行 77 步门禁；该 Bash 入口运行至终态，质量门禁报告 `steps_completed=77`、`[quality-gate] done`，总耗时 2178.71 秒。最后的性能步骤使用刷新后的两项 recorder baseline 并通过：普通 recorder 中位数为 `154302 ns/op` / `1004400 p95-ns/op` / `2232 allocs/op`，sandbox-enriched 为 `223334 ns/op` / `1019400 p95-ns/op` / `2714 allocs/op`；其余四项 diagnostics 基线也保持通过。随后 full-chain smoke 输出 `FULL_CHAIN_SUCCESS`，最终 strict `govulncheck ./...` 输出 `No vulnerabilities found`、可达漏洞为 0（另有 2 个 imported-but-not-called 与 1 个 required-but-not-called finding，未扩大修复范围）。该运行没有更改阈值、checker、默认 benchmark 参数或 CPU 设置，构成 Windows PowerShell 与 Windows Git Bash 两个质量门禁入口的实际完成证据；它不是 native Linux 运行证据。

Windows Bash 会先执行 shell documentation/readiness 检查，再委托 PowerShell 完整门禁；这些执行不构成 native Linux 的运行证据。

## 独立代码审查

独立审查未发现 Critical/Important 阻断项；审查者直接核对模块缓存中的最低 require，确认五个模块升级均由 `x/net v0.60.0` 的最小一致图要求，并独立验证 `GOROOT` 指向 `go1.26.9`、`GOTOOLCHAIN=auto`。CI、扫描器策略、双平台门禁、阈值与 baseline 均无代码改动。审查结论以完整门禁实际通过为合并前提，不代替执行证据。

审查未能获取 setup-go v5 官方源码：本地无缓存，普通权限网络连接失败，重试后远端关闭连接。因此本次审查没有独立确认该 action 的 toolchain 解析实现；现有 `go-version-file: go.mod` 配置未见确切缺陷证据，未据此扩大 workflow 修改范围。本轮只有本地 Windows 执行证据，没有实际 GitHub-hosted CI 运行结果。

## 风险与回滚

依赖的间接升级可能改变 HTTP/TLS/文本处理内部行为，须由完整测试、race、lint、contract/replay/performance 与 strict scan 验证。自动工具链首次下载需要网络或离线预装。

若回归失败，回滚本 change 的 `go.mod`、`go.sum` 及工具链文档；严格扫描保持阻断，旧漏洞基线不能被当成合格交付。原 host-resume 功能仍在独立 worktree 中，尚未取得其与安全修复组合后的完整门禁结果。
