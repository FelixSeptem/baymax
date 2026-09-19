## Why

Baymax 已归档跨 Provider handoff 与 stream edge conformance（137），具备 provider-neutral 的响应侧归一化、tool-call/tool-result 关联、终态与 replay/gate 证据。但**请求侧**（`ModelRequest` → provider SDK request）没有任何版本化 fixture、离线 replay 或 gate 覆盖，语义在适配器内部被压平且不可观测。

现状审计（本提案 1.1 阶段实测，非推测）：

- `core/types.ModelRequest` 同时携带 `Input`、`Messages []Message{Role,Content}`、`ToolResult []ToolCallOutcome`、`Capabilities`。
- `core/runner/runner.go` 的 `toModelRequest` 把 `req.Messages`（其中已由 Skill bundle mapping 追加 `Role: "system"` 的 prompt fragment）原样传入 `ModelRequest`。
- 三个适配器（`model/openai`、`model/anthropic`、`model/gemini`）在调用 SDK 前统一执行 `toolcontract.CanonicalInput(req)`，其语义是：`base = trim(req.Input)`，**仅当 `Input` 为空时**才回退到 `req.Messages[len-1].Content`；随后把 tool result 序列化为 `[tool_result_feedback.v1]` 文本信封追加到 `base`。
- 结果：当 `Input` 非空（主线运行路径）时，**全部 `Messages` 被静默丢弃**，其中包含 system 角色 Skill fragment 与 assistant/tool 历史；`ToolResult` 只以文本信封形式进入**单个 user 文本块**，不投影为 provider 原生 tool role / tool_result part；`Capabilities` 不进入请求；`PrefixMetadata`/`PrefixHash` 只用于装配侧记账，不形成可验证的稳定前缀投影。
- `types.TokenUsage` 仅有 `InputTokens`/`OutputTokens`/`TotalTokens`，**未表达 cached input / cache read / cache write**，因此 prompt cache 成本不可观测。

当前缺口不是"再建一套 provider 协议"，而是：请求侧投影缺少一份**版本化、有界、可回放、可阻断**的证据基线。没有它，任何 role/tool-result/ordering 的语义漂移都无法被 fixture 捕获，也无法区分"设计选择"与"回归缺陷"。这正好是 roadmap「观察候选：Provider 结构化上下文投影与 Prompt Cache 可观测性」声明的启动条件（fixture 证明语义丢失/顺序漂移）。

## What Changes

- 新增版本化、provider-neutral 的请求投影契约 `provider_request_projection.v1`：有界 schema、canonical 归一化投影、确定性 digest、稳定 gap/drift 分类词表。
- 新增 provider 侧审计测试：通过既有注入缝隙（openai 的 `newResponse`/`newStream`、anthropic/gemini 的 `GenerateFn`/`StreamFn`）捕获**真实送达到 SDK 边界的内容**，把 role 存在性、part 顺序、tool result 是否原生、工具顺序、稳定前缀、cache usage 可用性归一化为有界事实。
- 新增离线、只读、确定性的 replay：解析 `provider_request_projection.v1`，对 declared gap 与 computed gap 做一致性校验，保证缺口被**钉住**（修复或回归都会显式改变 fixture，不允许静默漂移），并校验 replay 幂等。
- 新增 shell/PowerShell 对等 gate：适配器所有权、SDK 请求构造形状、无 raw payload 诊断、fixture 有界、taxonomy 稳定、Run/Stream parity、replay 幂等；失败分类码可审计。
- 明确 cache usage 的兼容演进口径：只允许 `additive + nullable + default`，历史 fixture 缺失字段按 unavailable/0 处理，未知字段安全忽略。
- 本变更**不修改运行时可观测行为**：请求侧投影修复（原生 role/tool-result part、稳定前缀结构化）留给由本提案证据触发的后续增量 change，本提案只交付证据基线、契约与门禁，因此回滚只删除新增文件。
- `example impact`：`无需示例变更（附理由）`。

## Capabilities

### New Capabilities

- `provider-request-projection-and-cache-observability`: 版本化、有界、可回放的请求侧投影契约与 cache usage 兼容演进口径，覆盖 role/part 顺序、tool-result 归属、工具顺序、稳定前缀与 usage 可观测性。

### Modified Capabilities

- `cross-provider-handoff-and-stream-edge-conformance`: 从响应侧扩展到请求侧投影，要求 supported adapter 的请求投影可被同一 conformance 家族捕获、归一化与分类，且不得泄漏 provider-only 字段。
- `diagnostics-replay-tooling`: 扩展 replay 命名空间与确定性分类，新增 `provider_request_projection.v1` 与请求侧 drift/gap 码。

## Example Impact Assessment

`无需示例变更（附理由）`

理由：本变更只新增离线 fixture、conformance 归一化、replay 与 gate，不改变 `examples/agent-modes` 的配置键、runtime path、expected markers 或任何可观察语义；请求侧投影修复被显式排除在本提案范围之外，因此示例无需跟随变更。若后续增量 change 落地原生 role/tool-result 投影并改变示例可观察输出，必须重新评估并遵循文档先行规则（先更新 `MATRIX.md` 与对应模式 README）。

## Impact

- 受影响实现 owner：`model/conformance`（新增投影契约）、`model/openai`、`model/anthropic`、`model/gemini`（仅新增审计测试，不改运行时行为）、`tool/diagnosticsreplay`（新增 replay 命名空间）。
- 受影响测试与证据：provider 审计测试、conformance 正向/负向/边界测试、replay fixture 与幂等测试、shell/PowerShell gate。
- 受影响文档：`docs/mainline-contract-test-index.md`、`model/README.md`、`docs/development-roadmap.md`、`README.md`、`docs/pi-agent-comparison-and-adoption-study.md`。
- 明确不做：不新增 provider、不新增 runtime 配置键、不新增 credential store、不建设远程 model catalog / gateway / 全局路由、不改 `context/*`、不在非 `model/<provider>` 包引入 provider SDK、不为 prompt cache 新增策略或自动重排、不修改 `TokenUsage` 现有字段、不落盘 raw prompt/raw reasoning/credentials。
- 回滚：删除新增的 conformance 投影契约、provider 审计测试、replay 命名空间与两个 gate 脚本，并从 `scripts/check-quality-gate.*` 摘除接线；无持久化迁移、无配置变更、无运行时语义回退。

## Baseline Audit Evidence (Tasks 1.1–1.3)

- 请求构造单一入口：`core/runner/runner.go` 的 `toModelRequest` 是主线唯一的 `types.ModelRequest` 构造点（`context/assembler/compactor.go` 只用于压缩子调用）。它无条件转发 `Input`、`Messages`、`ToolResult` 与 `Capabilities`，因此请求侧语义丢失发生在适配器，而非 runner。
- Skill 与 system 角色的来源：`core/runner/runner.go` 的 `applySkillBundleMappings` 在 `prompt_mode=append` 时把 `bundle.SystemPromptFragments` 追加为 `types.Message{Role: "system"}`；这些消息随后随 `ModelRequest.Messages` 进入适配器。
- 适配器压平事实：`model/openai` 的 `Generate`/`Stream` 在 `toolcontract.WithCanonicalInput` 之后仅使用 `req.Input`，构造 `responses.ResponseNewParams{Model, Input: {OfString}}`；`model/anthropic` 与 `model/gemini` 调用 `toolcontract.CanonicalInput(req)` 后只把结果字符串传给内部 `generate`/`newStream`，SDK 侧统一构造单个 user 文本块（`anthropic.NewUserMessage(anthropic.NewTextBlock(input))`）。
- `toolcontract.CanonicalInput` 的丢弃语义：`base := strings.TrimSpace(req.Input)`，`Messages` 仅在 `Input` 为空时以 `Messages[len-1].Content` 形式参与；`ToolResult` 被序列化为 `[tool_result_feedback.v1]` JSON 信封拼接到 `base` 之后，最大 64 KiB，缺失 `call_id`/`tool_name` 时以 `feedback_invalid` 在 provider 调用前失败。
- 可观测缝隙：`model/openai` 的内部字段 `newResponse`/`newStream` 可在包内测试替换，直接观察到 `responses.ResponseNewParams`；`model/anthropic`/`model/gemini` 的 `Config.GenerateFn`/`StreamFn` 可在适配器→SDK 边界（`input string`）捕获投影结果。三个缝隙均为既有能力，本提案不新增任何注入点。
- usage 可观测性边界：`types.TokenUsage` 无 cache 相关字段，`types.ModelResponse.Usage` 与 `RunResult.TokenUsage` 直接引用该类型，因此任何 cache 字段都只能按诊断兼容规则（`additive + nullable + default`）演进。
- `CountTokens` 与生成路径的投影不对称（本提案记录但不修复）：`model/anthropic/client.go` 的 `CountTokens` 直接遍历 `req.Messages` 构造 system block 与 assistant/user message，`model/gemini/client.go` 的 `buildTokenContents` 同样遍历 `req.Messages` 映射 role（system→user、assistant→model）。也就是说 token 会计走的是原生 role 投影，而 `Generate`/`Stream` 走压平文本投影，二者可能不对应。本提案只把它写成稳定可复现的基线事实与后续增量触发证据，不在本 change 内改动任何适配器投影行为。
- 有界性与隐私口径：确认 `model/conformance` 只依赖 `core/types`，`tool/diagnosticsreplay` 通过本提案新增对 `model/conformance` 的依赖，两者都不引入 provider SDK；请求侧只记录 digest、计数器与有序标识，不落 raw prompt、reasoning 或 tool output body。

## Implementation Evidence (Tasks 4.1–6.3)

交付物：

- `model/conformance/request_projection.go`：`provider_request_projection.v1` 有界 schema、`source`/`observed` 双投影、canonical 投影与 SHA-256 digest、11 个稳定分类码、`ValidateCacheUsageBaselineUnavailable` 基线校验。
- `model/conformance/request_facts.go`：`RequestFactsFromModelRequest` 把 `types.ModelRequest` 归一化为 provider-neutral 事实（只记录结构、顺序、标识与 digest）。
- `model/openai`、`model/anthropic`、`model/gemini` 的 `request_projection_test.go`：通过既有缝隙捕获真实 SDK 请求形状，并钉住 `declared_gap`（三项：capability / role / tool-result-native）。
- `tool/diagnosticsreplay/request_projection.go`：离线只读 replay，复用 `model/conformance` 的常量与解析（不复制第二套词表）。
- `tool/diagnosticsreplay/testdata/model_request_projection.v1.json`：12 个 case（3 provider × run / stream + run_stream_parity + without_tool_results），23 401 字节。
- `tool/contributioncheck/request_projection_boundary_test.go`：provider-neutral 边界、适配器 canonical 投影入口、无 raw payload 字段、fixture 有界与覆盖、taxonomy 文档一致性。
- `scripts/check-provider-request-projection-contract.sh` 与 `.ps1`：对等 gate，已接入 `scripts/check-quality-gate.sh` 与 `.ps1`。

验证结果（精确命令与结果）：

| 命令 | 结果 |
| --- | --- |
| `go test ./model/conformance ./model/openai ./model/anthropic ./model/gemini ./core/types ./tool/diagnosticsreplay -count=1` | 全部 ok（顶层用例 21 / 13 / 12 / 11 / 73 / 97 通过） |
| `go test -race`（同上六包） | 全部 ok |
| `go test ./tool/contributioncheck -run 'TestProviderRequestProjectionContractBoundary' -count=1` | ok（5 个子用例） |
| `go test ./tool/diagnosticsreplay -run 'ProviderRequestProjection' -count=2` | ok（幂等；含 12 个分类分支与 3 个有界分支） |
| `bash scripts/check-provider-request-projection-contract.sh` | passed |
| `bash scripts/check-openspec-roadmap-status-consistency.sh` | passed |
| `go test ./tool/contributioncheck -run 'TestReleaseStatusParityDocsConsistency' -count=1` | ok |
| `openspec validate --all` | 122 passed, 0 failed |
| `openspec validate <change> --strict` | valid |
| `go test ./...` | 3 个包失败，均为环境受限且与改动无依赖关系，见下 |

`go test ./...` 的 3 个失败包与原因（`go list -deps ./cmd/host-jsonl ./integration ./host/...` 已确认三者都不依赖本变更新增或修改的任何包）：

- `cmd/host-jsonl`：`TestConformanceHelperPreservesUnicodeAndSerializesFrames`、`TestConformanceHelperSerializesConcurrentEventsWithoutInterleaving`、`TestConformanceHelperEOFSettlesPendingRequest` 报 `frames=2 want ...`；同包 `TestConformanceHelperBackpressureExitsDeterministically` 通过，属该 helper 的既有投影行为，与本变更无关。
- `integration`：`TestAgentModeP0RegressionPaths/positive-rag-minimal` 报 `go run timeout for mode rag-hybrid-retrieval/minimal`，为本机 `go run` 超时的环境问题。
- `tool/contributioncheck`：`TestAgentModeDocFirstDeliveryContractExecutionPassFailAndExceptionalBranches` 在嵌套 PowerShell 中报 `The term 'git' is not recognized`，属沙箱 PATH 限制。

环境受限、未能在本机完整执行的命令（附原因与替代证据）：

- `golangci-lint run --config .golangci.yml`：在本机沙箱中**挂死**，不是报错而是长时间无输出。已定位为工具/环境问题而非代码问题，证据是复现到最小配置仍然挂死：
  - `timeout 300 golangci-lint run --config .golangci.yml ./model/conformance/` → `EXIT=124`（被 `timeout` 杀死）。
  - `timeout 120 golangci-lint run --no-config --default=none --enable=errcheck ./model/conformance/` → `EXIT=124`（单包、单 linter、无自定义配置仍挂死）。
  - 替代证据（已执行且通过）：`go vet` 对全部受影响包 `VET_EXIT=0`；`gofmt -l` 与 `goimports -l` 对全部新增/修改文件无输出（格式与导入均合规）；`go test -race` 六包全部 ok。
- PowerShell 版 gate 脚本：本机 PowerShell 沙箱无法派生 `go.exe`（既不能解析 `go`，直接调用绝对路径也无输出），因此只能验证其静态段（fixture 存在/版本/大小 + 分类词表一致性），静态段已通过；脚本内的三条 Go 测试调用与已验证通过的 shell 版逐字相同。
- `scripts/check-quality-gate.ps1` / `scripts/check-docs-consistency.ps1`：受同一 PowerShell 沙箱限制无法端到端执行；对应的 shell 侧检查（`check-openspec-roadmap-status-consistency.sh`、provider gate、状态一致性 Go 测试）已通过。

非目标复核（Task 6.3，基于 `git status` 精确核对）：

- `model/openai/client.go`、`model/anthropic/client.go`、`model/gemini/client.go`、`core/runner/runner.go` 与整个 `context/*`、`runtime/*` 均**未被修改**：本变更在 `model/*`、`core/*`、`context/*`、`runtime/*` 下唯一改动的文件是 `model/README.md`（文档）。
- 未新增 provider、未新增 runtime 配置键、未新增 credential store、未建设远程 catalog / gateway / 全局路由。
- 未新增任何诊断字段或诊断写入路径：新增的 `model/conformance` 与 `tool/diagnosticsreplay` 代码不引用 `observability/event`，`RuntimeRecorder` 单写入口未被绕过。
- 未落盘 raw prompt / reasoning / credentials / 无界 payload：契约只承载 digest、计数器、有序标识与枚举化 part/role 种类。
- `types.TokenUsage` 字段与 `ModelResponse.Usage` / `RunResult.TokenUsage` 的既有语义未变，并由 `core/types/token_usage_semantics_test.go` 冻结形状。
