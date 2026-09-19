## Example Impact Assessment

无需示例变更（附理由）

本变更只新增离线 fixture、conformance 归一化、离线 replay 与 gate，不修改 `examples/agent-modes` 的配置键、runtime path、expected markers 或任何可观察语义，也不修改 `model/<provider>` 的运行时投影行为。若后续增量 change 落地原生 role / tool-result 投影并改变示例可观察输出，必须重新评估并先完成 `MATRIX.md` 与对应模式 README 的文档基线。

## 1. Baseline and Projection Contract

- [x] 1.1 复核 `core/runner` 的 `toModelRequest`、Skill bundle system fragment 注入、`toolcontract.CanonicalInput` 丢弃语义，以及三个适配器到 SDK 边界的投影路径；把可复现事实写入 proposal 的 Baseline Audit Evidence。
- [x] 1.2 在 `model/conformance` 定义 `provider_request_projection.v1`：有界 schema、`source`/`observed` 双投影、canonical 投影与 digest、稳定 gap/drift 词表；覆盖 malformed、unknown-version、超界与缺字段校验用例。
- [x] 1.3 定义 `declared_gap` 一致性语义（缺失申报、已修复、未知码三条分支）并落测试，确保缺口既不能被静默引入也不能被静默修复。

## 2. Provider Request Projection Audit

- [x] 2.1 `model/openai`：通过既有 `newResponse`/`newStream` 缝隙捕获 `responses.ResponseNewParams`，断言 `Input.OfString` 有值、`Input.OfInputItemList` 为空、`Instructions` 为空，并断言 system/assistant 消息未进入请求。
- [x] 2.2 `model/anthropic`：通过 `Config.StreamFn`/`GenerateFn` 捕获适配器→SDK 边界 `input string`，断言其等于 `toolcontract.CanonicalInput(req)`、单 part、system fragment 内容缺失、tool result 仅以文本信封出现。
- [x] 2.3 `model/gemini`：同 2.2 口径验证投影等价性，并覆盖 Generate 与 Stream 两条路径。
- [x] 2.4 三个适配器统一验证 Run/Stream 请求投影等价：同一 `ModelRequest` 在 `Generate` 与 `Stream` 下归一化投影一致（除允许的调用形状差异外）。

## 3. Cache Usage Observability Compatibility

- [x] 3.1 定义 `CacheUsageProjection` 的 additive + nullable + default 口径，并断言当前适配器投影为 `available=false` 且不伪造 read/write 值。
- [x] 3.2 添加历史 fixture（缺 cache 字段）与未知字段 fixture 的兼容用例，证明解析不失败且未知字段被安全忽略。
- [x] 3.3 断言本变更未修改 `types.TokenUsage` 现有字段与 `ModelResponse.Usage` / `RunResult.TokenUsage` 的既有语义。

## 4. Replay and Contract Gates

- [x] 4.1 在 `tool/diagnosticsreplay` 实现 `provider_request_projection.v1` 解析、有界归一化、离线只读执行与幂等校验。
- [x] 4.2 添加 drift/gap 分类测试：schema、role 投影、tool-result 原生性、part 顺序、稳定前缀、工具顺序、能力投影、Run/Stream parity、cache usage、overflow。
- [x] 4.3 新增版本化 fixture `tool/diagnosticsreplay/testdata/model_request_projection.v1.json`，覆盖 OpenAI/Anthropic/Gemini × Run/Stream、历史缺字段、未知字段与有界边界。
- [x] 4.4 新增 `scripts/check-provider-request-projection-contract.sh` 与 `.ps1` 对等 gate，覆盖适配器所有权、SDK 请求构造形状、无 raw payload 诊断、fixture 有界、taxonomy 稳定、replay 幂等。
- [x] 4.5 把 gate 接入 `scripts/check-quality-gate.sh` 与 `scripts/check-quality-gate.ps1`，并验证缺失 fixture 或 gate 证据会阻断质量门禁。

## 5. Diagnostics and Documentation

- [x] 5.1 验证本变更不新增任何诊断字段、不落盘 raw prompt/reasoning/credentials，且 `RuntimeRecorder` 单写入口未被绕过。
- [x] 5.2 更新 `docs/mainline-contract-test-index.md`（新 gate 映射）、`model/README.md`（请求侧投影契约与 owner 边界）、`docs/development-roadmap.md`（当前状态与候选收敛）、`README.md`（里程碑快照）与 `docs/pi-agent-comparison-and-adoption-study.md`（证据基线）。
- [x] 5.3 运行 `bash scripts/check-openspec-roadmap-status-consistency.sh` 与 `pwsh -File scripts/check-docs-consistency.ps1`，确认无 `roadmap-status-drift` 与文档不一致。

## 6. Integrated Verification and Handoff

- [x] 6.1 运行聚焦套件（provider 三个包、`model/conformance`、`tool/diagnosticsreplay`、`core/runner`）并记录正/负/边界/幂等证据。
- [x] 6.2 运行 `openspec validate --all`、`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`、`pwsh -File scripts/check-quality-gate.ps1`、`pwsh -File scripts/check-docs-consistency.ps1` 与两个 provider contract gate；记录精确结果与环境受限命令及原因。
- [x] 6.3 对照非目标做最终范围复核：确认未新增 provider、runtime 配置键、credential store、远程 catalog/gateway、`context/*` SDK 依赖、raw payload 诊断字段或任何运行时投影行为变更。
