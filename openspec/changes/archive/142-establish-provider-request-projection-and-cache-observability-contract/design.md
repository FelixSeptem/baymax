## Context

本提案是 roadmap「观察候选：Provider 结构化上下文投影与 Prompt Cache 可观测性」的立项落地，启动信号是该候选声明的触发条件：需要"真实 SDK 形状的版本化 fixture、replay 与 benchmark 证明消息语义丢失、顺序漂移"。审计（proposal 的 Baseline Audit Evidence）已确认丢失是结构性的、可稳定复现的，因此具备立项条件。

前序基线（不重复建设）：归档 137 已建立响应侧 `provider_handoff_stream_edge.v1`、`model/conformance` 的 provider-neutral 归一化、`tool/diagnosticsreplay` 的离线 replay 与 shell/PowerShell 对等 gate。本提案沿用同一套层次与命名，只把覆盖范围从"Provider → Runtime"扩展到"Runtime → Provider"。

## Goals / Non-Goals

### Goals

1. 建立版本化、有界、provider-neutral 的请求投影契约 `provider_request_projection.v1`。
2. 在三个 supported adapter 上通过既有缝隙捕获真实投影，把语义事实归一化为可比较的投影结构。
3. 让"已知语义丢失"变成**被钉住的、可审计的声明**，而不是隐式行为：任何修复或回归都必须显式改动 fixture。
4. 给出 cache usage 的兼容演进口径，使未来新增 cache 字段不需要破坏历史 fixture。
5. 交付 shell/PowerShell 对等 gate，并把失败分类码固定为稳定词表。

### Non-Goals

- 不修改 `model/<provider>` 的运行时投影行为（不引入原生 role / 原生 tool_result part / 结构化 stable prefix）。
- 不新增 provider、runtime 配置键、credential store、远程 catalog、gateway 或全局路由。
- 不改 `TokenUsage` 已有字段，不实现 prompt cache 策略、自动重排或 cache 命中优化。
- 不落盘 raw prompt、raw reasoning、完整 transcript、credentials 或任何无界 payload。
- 不建立第二套 event ordering / cursor / terminal 状态机；复用既有 replay 与失败分类 owner。

## Decisions

### D1. 复用 `model/conformance` 作为 provider-neutral 契约家，不新建包

137 已把 provider-neutral 归一化放在 `model/conformance`，并由 `model/<provider>` 的测试与 `tool/diagnosticsreplay` 消费。请求侧沿用同一包，保证"响应侧 + 请求侧"属于同一 conformance 家族，且 `model/conformance` 不引入任何 provider SDK（守住"Provider 细节必须落在 `model/<provider>`"）。

### D2. 用 `source` / `observed` 双投影表达语义丢失，而不是把丢失写成断言失败

每个 case 携带：

- `source`：runner 实际上送进 `ModelRequest` 的 provider-neutral 事实（有序 role 列表、tool result 归属与关联、工具顺序候选、能力需求、Skill fragment 计数）。
- `observed`：归一化后真正到达 provider SDK 边界的事实（有序 part 种类、role 存在性、tool result 是否原生、工具顺序、内容 digest、cache usage 可用性）。

计算 `gap := ClassifyRequestGap(source, observed)`，并与 case 声明的 `declared_gap` 比对：

- 一致 → 缺口被**钉住**，通过。
- 声明为空但计算出缺口 → 未申报的语义漂移，按对应 drift 码失败。
- 声明非空但计算不出缺口 → 投影行为已改变（修复或误改），必须显式更新契约，按 `provider_request_contract_drift` 失败。

这样避免了"让 gate 常红"（无法阻断回归）也避免了"把缺陷写成期望值"（无法发现修复）。对已经符合预期的维度（bounded、digest 稳定、工具顺序、cache 缺省、Run/Stream parity、幂等）使用 `expected == observed` 的常规等值断言。

### D3. 观测缝隙只用既有注入点，不新增运行时 API

- `model/openai`：包内测试替换 `client.newResponse` / `client.newStream`，直接断言 `responses.ResponseNewParams`（`Input.OfString` 有值、`Input.OfInputItemList` 为空、`Instructions` 为空）。
- `model/anthropic` / `model/gemini`：使用 `Config.GenerateFn` / `Config.StreamFn`，捕获适配器→SDK 边界的 `input string`，并断言 `input == toolcontract.CanonicalInput(req)`、单个 part、system fragment 内容缺失、tool result 仅以 `[tool_result_feedback.v1]` 信封出现。

选择理由：这三个缝隙都是既有能力，审计不引入新 API，因此可以"零运行时改动"完成证据采集；同时它天然证明了丢包发生在适配器而不是测试替身。

### D4. 稳定前缀只做 digest 投影，不做策略

`PrefixMetadata.PrefixVersion`/`PrefixHash` 由 `context/assembler` 拥有。请求侧只记录 `stable_prefix_digest` 与其在投影中的位置，用于检测"顺序漂移"，不解析、不重排、不生成前缀文本。owner 边界不变。

### D5. cache usage 只做可用性投影与兼容证明

`CacheUsageProjection{Available bool, ReadTokens int64, WriteTokens int64}`，全部 `omitempty` 且默认零值：

- 当前适配器不返回 cache 字段 → `available=false`，`read/write` 缺省。
- fixture 必须包含一个"历史 fixture 缺少 cache 字段"的 case，证明解析不失败、按 unavailable 处理、不伪造值。
- 必须包含一个"含未知字段"的 case，证明 `encoding/json` 默认忽略未知字段，新增字段不会破坏历史 fixture。

### D6. 有界性优先

沿用 137 的边界口径并收紧请求侧：

- `cases` ≤ 64，`parts` ≤ 128，`roles` ≤ 8，`tools` ≤ 64。
- 每个 digest/locator ≤ 256 字节，任何采集到的文本按 digest 记录、不落原文。
- fixture 文件 ≤ 2 MiB（由 gate 静态校验）。
- 超界统一归入 `provider_request_overflow_drift`，不回退、不截断后继续。

### D7. gate 与文档接线

新增 `scripts/check-provider-request-projection-contract.sh/.ps1`，检查项与 137 的 gate 对等：

1. fixture 存在且 ≤ 2 MiB；
2. 适配器所有权：三家的 SDK 请求构造仍在各自包内，且 `CanonicalInput` 仍是唯一入口；
3. 无 raw payload 诊断字段新增；
4. taxonomy 稳定：drift/gap 码集合与文档、replay 常量一致；
5. `model/conformance` 与 `tool/diagnosticsreplay` 不引入 provider SDK；
6. replay 幂等：`-count=2` 重复执行结果一致。

gate 接入 `scripts/check-quality-gate.sh` 与 `scripts/check-quality-gate.ps1`（对等位）。

## Risks / Trade-offs

| 风险 | 说明 | 缓解 |
| --- | --- | --- |
| 钉住缺口可能被误读为"接受缺陷" | `declared_gap` 非空意味着已知不一致 | 在 fixture、spec 与文档中明确 `declared_gap` 是**待修复项的证据锚点**，并给出后续增量 change 的触发条件；不写入任何"容忍"语义 |
| 测试依赖包内私有字段 | `model/openai` 的缝隙是未导出字段 | 只在包内测试使用（与既有测试一致），不提升为导出 API；断言内容为 SDK 参数形状而非内部实现细节 |
| anthropic/gemini 只见 `input string` | 无法直接断言 `MessageNewParams` | 以"适配器→SDK 边界即 `input string`"为契约事实，另由 gate 静态校验 SDK 参数构造形状（单个 user text block）补足 |
| 新增 gate 增加门禁耗时 | 两个 provider 测试 + replay | 复用 `-count=2`，用例数与 fixture 有界；不引入网络调用或 benchmark 长跑 |

## Migration Plan

不涉及数据或配置迁移。历史 `provider_handoff_stream_edge.v1` fixture 与 gate 保持原样；新 fixture 使用独立命名空间与版本号，二者互不影响。回滚即删除新增文件与 gate 接线。

## Verification

```bash
go test ./model/... ./tool/diagnosticsreplay/... ./core/... -count=1
go test -race ./model/conformance/... ./model/openai/... ./model/anthropic/... ./model/gemini/... ./tool/diagnosticsreplay/...
golangci-lint run --config .golangci.yml
bash scripts/check-provider-request-projection-contract.sh
```

```powershell
pwsh -File scripts/check-provider-request-projection-contract.ps1
pwsh -File scripts/check-quality-gate.ps1
pwsh -File scripts/check-docs-consistency.ps1
```
