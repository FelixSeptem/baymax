## Context

归档 142 的 `provider_request_projection.v1` 已有 `CacheUsageProjection` 和历史兼容规则，但当前三个 adapter 都将 cache usage 固定为 unavailable。当前锁定的 SDK 已分别提供 OpenAI `cached_tokens`、Anthropic `cache_read_input_tokens`/`cache_creation_input_tokens`、Gemini `cachedContentTokenCount` 等来源；这些字段的命名、生命周期和可用性并不完全相同。

本 change 只扩展既有请求/响应 conformance 的 usage projection。Provider SDK 类型和字段解释留在 `model/<provider>`；归一化 cache usage 通过独立的 `ModelResponse.CacheUsage` 交付，并由 Stream 终态事件 `Meta["cache_usage"]` 承载；`TokenUsage` 语义与 RuntimeRecorder schema 保持不变。`model/conformance`、`tool/diagnosticsreplay` 和 RuntimeRecorder 不引入 SDK 依赖，也不新增缓存事实源。

## Goals / Non-Goals

**Goals:**

- 以 provider-neutral、bounded、nullable/default 结构表达 cache read、cache write/create、total 与 availability。
- 在 Run/Stream 的真实 adapter 边界捕获 usage，保证生命周期差异不会造成重复计数。
- 通过版本化 fixture、离线 replay、稳定 drift taxonomy 和 shell/PowerShell gate 固定兼容性。
- 保持历史 fixture、旧 `TokenUsage` 字段和无 cache 来源 provider 的行为兼容。

**Non-Goals:**

- 不实现 cache key、prefix 重排、cache 创建/失效/刷新、价格计算或自动优化策略。
- 不把 provider-specific TTL、region、billing tier 或价格字段提升为通用 runtime contract。
- 不把普通 input token 当作 cached token，不在 CountTokens 缺少来源时推导 cache usage。
- 不新增配置键、诊断事实源、remote catalog、credential store 或 hosted cache service。

## Decisions

### D1. 复用既有 capability 与 projection envelope

在 `provider-request-projection-and-cache-observability` 既有 requirement 上做 delta，不创建新的 cache capability 或第二套 usage schema。保留 `provider_request_projection.v1` 的 canonical envelope；cache 字段作为 additive 子结构演进。

**替代方案：**新建独立 `provider-cache-usage` spec/fixture。放弃原因：会把同一次请求的角色、工具结果和 usage 拆成两个事实源，造成 digest、replay 与 Run/Stream parity 重复治理。

### D2. Adapter-owned extraction，conformance-owned normalization

每个 adapter 在自身 SDK response/stream mapping 内读取 provider-native usage，并构造不含 SDK 类型的 bounded intermediate facts。`model/conformance` 只校验非负、可用性、字段边界、total 关系和 canonical serialization；replay 只处理 fixture。

**替代方案：**在 `model/conformance` 直接解析各 SDK 的 JSON。放弃原因：违反 Provider 细节必须位于 `model/<provider>` 的模块边界，并使 SDK 升级泄漏到通用包。

### D3. 保守的 normalized semantics

规范字段采用 `available`、`read_tokens`、`write_tokens`、`total_tokens`，另带有界的 `source_kind`/`source_version` 标识。`total_tokens` 仅表示 cache read + write/create 的合计，不表示 provider 全部 input tokens。只有 SDK 明确表达的数量才填充；无法区分 read 与 write 的 provider 只填可确认的 total 或保持 unavailable。provider-specific TTL 等字段留在 adapter 内，不能强行映射。

**替代方案：**把所有 provider usage 原样透传。放弃原因：会把 provider-only 字段变成 runtime 依赖，并破坏跨 Provider 可比较性。

### D4. Stream authoritative usage policy

Run 使用终态 response usage。Stream 若存在多个 usage 事件，adapter 只采纳 contract 定义的 authoritative terminal usage；缺失终态但有单个可信 interim usage 时，按 provider-specific adapter 规则标记来源，不把多个事件相加。重复或互相矛盾的 usage 由 replay/gate 分类，不静默修正。

**替代方案：**每个事件增量累加。放弃原因：多数 provider usage 是累计快照而非增量，累加会产生稳定 over-count。

### D5. Compatibility and privacy

新增字段必须为 nullable/default，历史 fixture 缺失时解码为 unavailable/zero。canonical digest 对缺失字段使用默认语义；未知字段不进入 digest。只记录 token counts 和 bounded source labels，不记录 prompt、cache key、credentials、TTL payload 或原始 SDK response。

### D6. Evidence and gates before runtime promotion

先更新 fixture、adapter unit tests、replay 和 contract gate；仅当这些证据证明来源和 parity 稳定时，才允许将字段接入更高层 diagnostics/eval。当前只经独立响应载体返回，不写 RuntimeRecorder，避免在来源和语义未稳定前扩散 schema。

## Risks / Trade-offs

- **Provider 语义不可完全同构** → 只暴露可确认的中立数量；不确定维度保持 unavailable，并保留 bounded source kind。
- **Stream 中间 usage 造成重复计数** → 采用终态优先、非累加策略，增加重复/冲突 fixture。
- **SDK 字段未来变化** → adapter 层做版本化 fixture 和缺失字段默认；未知字段安全忽略，schema drift fail-fast。
- **新增字段误被当作价格/成本事实** → 明确本 change 只表示 token usage，不计算价格；成本由既有 budget/eval owner 另行解释。
- **现有脏工作区干扰验证** → 提案文件精确检查与暂存；不清理用户已有改动和缓存。

## Migration Plan

1. 在 adapter 层增加 SDK usage 到 bounded intermediate facts 的映射和 Run/Stream 单测。
2. 扩展 `provider_request_projection.v1` fixture、conformance 校验与 diagnostics replay；保留历史 fixture 兼容测试。
3. 接入 provider projection contract gate、文档和主线测试索引。
4. 回滚时删除新增映射、fixture/replay/gate 与文档 delta；历史 payload 继续按 unavailable 读取，无配置或持久化迁移。

## Open Questions

无。Provider-specific TTL/region/pricing 不进入本 change，未来若有需求必须另行提出并先证明 owner 与 contract 边界。

## Example Impact Assessment

无需示例变更（附理由）。本 change 只修改 provider usage fixture、replay、adapter 测试与兼容性 contract，不改变 `examples/agent-modes` 的配置、runtime path 或 expected markers。
