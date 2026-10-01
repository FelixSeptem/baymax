## Why

归档 142、144、148 已分别固定并交付了 Provider 请求投影审计、原生角色/tool-result 请求投影和 `CacheUsage` 可观测性。Roadmap 的下一 Provider 方向不应重复这些运行时语义；但当前仍缺少一个离线、可回放的准入层，用于把新的 SDK 边界捕获或宿主成本样本与既有 `provider_request_projection.v1` 证据绑定，并确定性地区分“未发现漂移”“已证实漂移”和“证据不足”。

该准入层使后续 adapter 修复只在新的 role、Skill/tail、tool-result、顺序、Run/Stream cache parity 或稳定 cache 成本/P95 漂移被证实时才立项，避免将缺少观测数据误判为合规或直接修改 Provider 行为。

## What Changes

- 新增 versioned、reference-only 的 `provider_context_cache_evidence.v1` fixture contract：记录受支持 Provider 的有界 SDK 身份、既有 projection fixture/canonical digest 引用、结构化上下文与 cache evidence 结论、可选成本样本摘要，以及稳定准入结论。
- 在离线 replay 中验证 evidence 的引用完整性、隐私/边界、结论可推导性、Run/Stream cache parity、成本样本统计一致性和确定性 canonicalization；为 schema、reference、evidence、parity、cost 与 verdict drift 定义稳定分类。
- 增加 contribution gate，在 shell 与 PowerShell 上以同一 fixture 得出同一结论；`drift-confirmed` 只能输出对既有 Provider owner 的最小修复建议，绝不自动修改 adapter、Prompt、Skill、Tool、Policy、配置或 gate。
- 用三家 supported Provider 的离线 seed cases 覆盖无漂移、证据不足、已证实结构化上下文/cache 漂移、无效引用、原始 payload 泄漏、超界和历史 fixture 回归场景；不调用 live provider/network，也不存储原始 SDK request/response 或成本日志。
- 同步 Roadmap、主线 contract-test index 和相关模块文档，使归档 141、149、150、151 不再被作为活跃候选，且 Provider 方向的后续启动条件可追溯到本准入证据。
- 为完成验证而续期 11 条已存在且不允许增长的 A63 staged-split line-budget 例外至 2026-12-31；续期仅限既有超长文件，不放宽阈值、不新增路径、不改变业务代码，并要求后续治理维护者在到期前拆分或再次审查。
- **不包含** `ModelRequest`、`ModelResponse.CacheUsage`、`TokenUsage`、`RuntimeRecorder` schema、Provider SDK 请求构造、prompt cache 创建/刷新/失效策略、价格模型、远程 catalog、credential store、runtime 配置或 `examples/agent-modes` 行为变更。

## Capabilities

### New Capabilities

- `provider-context-cache-drift-admission-evidence`: 对既有 Provider 请求投影与 cache usage 的 SDK 边界/宿主成本证据进行有界、离线、引用式准入判定，并以稳定结论控制后续增量修复是否可以立项。

### Modified Capabilities

无。既有 `provider-request-projection-and-cache-observability` 保持其请求投影和 cache usage 运行时契约；本 change 只引用其 canonical evidence，不改变其 requirement。

## Example Impact Assessment

无需示例变更（附理由）：本 change 只增加离线 evidence fixture、replay、contribution gate 和文档映射，不改变 `examples/agent-modes` 的配置、runtime path、expected markers 或 rollback notes。若实施中发现任一 agent-mode 的可观察 marker 必须变化，必须先暂停代码任务，完成 `MATRIX.md` 与受影响模式 README 的文档基线，并将声明改为“修改示例”。

## Impact

- 主要代码 owner：`model/conformance`（SDK-neutral evidence normalization 与 verdict）、`tool/diagnosticsreplay`（离线 replay）、`tool/contributioncheck`（准入 gate）；`model/<provider>` 仅复用既有 SDK 边界捕获测试缝隙，不改变请求构造。
- 主要证据：`provider_request_projection.v1` 既有 fixture/canonical digest、`provider_context_cache_evidence.v1` reference-only fixture、离线 replay、shell/PowerShell gate，以及 Provider adapter 边界测试。
- 兼容性：新增 fixture contract 不改变既有 request-projection fixture 的版本、digest 或历史 replay 行为；缺少成本样本时只允许得到 `insufficient-evidence`，不得合成 cache 成本或 P95。
- 文档：`docs/development-roadmap.md`、`docs/mainline-contract-test-index.md`、相关 `model/README.md` 与 gate 说明；不改变外部 runtime API。
- 风险与回滚：结论词汇、统计边界和 reference 校验必须保持小而固定；line-budget 续期仅是可回滚的治理数据变更，回滚时恢复原到期日即可；整体回滚仍只移除该 evidence fixture/replay/gate 与文档映射，不触碰 Provider adapter、配置或持久化数据。
