## Purpose

本能力定义一套版本化、有界、可离线回放的**请求侧**投影契约：把 `ModelRequest`（`Input`、`Messages`、`ToolResult`、`Capabilities`）到 provider SDK request 的投影归一化为可比较的 provider-neutral 事实，从而让 role 语义、part 顺序、tool-result 归属、工具顺序、稳定前缀与 usage/cache 可观测性的丢失或漂移成为可被 fixture 捕获、可被 gate 阻断的显式证据，而不是适配器内部的隐式行为。

## ADDED Requirements

### Requirement: 请求侧投影 SHALL 由版本化有界契约表达

系统 MUST 提供版本化、provider-neutral 的请求投影契约 `provider_request_projection.v1`，用于表达一次模型调用的请求侧事实。契约 MUST 同时包含 `source`（runtime 送入 `ModelRequest` 的 provider-neutral 事实）与 `observed`（实际到达 provider SDK 边界的归一化事实），并 MUST 为二者提供确定性 canonical 序列化与 digest。

契约 MUST NOT 引入 provider SDK 依赖，MUST NOT 记录 raw prompt、raw reasoning、完整消息正文、credentials 或任何无界 payload；文本类事实 MUST 只以 digest 与长度表达。

#### Scenario: 契约对同一输入产生稳定投影
- **WHEN** 同一 `ModelRequest` 被同一适配器投影两次
- **THEN** canonical 序列化与 digest 完全一致，且不产生任何副作用

#### Scenario: 契约超界时确定性失败
- **WHEN** cases、parts、roles、tools 或单个 digest 超出契约边界
- **THEN** 契约以稳定的 overflow 分类 fail-fast，且不返回截断后的部分成功结果

#### Scenario: 未知版本被拒绝
- **WHEN** 解析到的 fixture 版本不是 `provider_request_projection.v1`
- **THEN** 解析失败并返回 schema drift 分类，不尝试兼容未知版本

### Requirement: 投影缺口 SHALL 被显式申报且不可静默漂移

契约 MUST 允许每个用例声明 `declared_gap`，并 MUST 依据 `source` 与 `observed` 计算出确定性 gap 分类。系统 MUST 在下列任一情况下失败：

- 未声明缺口但计算出缺口（未申报的语义漂移）；
- 声明缺口但计算不出缺口（投影行为已改变，需显式更新契约）。

`declared_gap` MUST 被当作待修复项的证据锚点；契约 MUST NOT 为它引入任何「容忍」「允许降级」或「等价语义」的解释。

#### Scenario: 未申报的角色投影丢失被发现
- **WHEN** `source` 声明了 system/assistant 角色而 `observed` 只包含 user 角色，且用例未声明对应缺口
- **THEN** 校验失败并返回 role projection drift 分类

#### Scenario: 投影行为改变必须显式更新契约
- **WHEN** 某次改动使此前被申报的缺口不再成立
- **THEN** 校验失败并返回 contract drift 分类，要求显式更新契约与 fixture

### Requirement: Adapter 请求投影 SHALL 可审计且不得泄漏 provider-only 语义

supported adapter（OpenAI、Anthropic、Gemini）的请求投影 MUST 可被同一 conformance 家族通过既有注入缝隙观测，且 MUST NOT 需要新增运行时 API 才能审计。观测到的投影 MUST 归一化为 provider-neutral 事实，provider-only 字段 MUST NOT 成为必需语义，MUST NOT 进入 runtime 契约或诊断输出。

同一 `ModelRequest` 的 Run 与 Stream 投影 MUST 语义等价，仅允许由调用形状决定的差异。

#### Scenario: 三家适配器投影可比较
- **WHEN** 三家适配器接收等价的 `ModelRequest`
- **THEN** 归一化后的投影结构可直接比较，且差异只以声明过的缺口或稳定 drift 分类表达

#### Scenario: provider-only 字段不进入契约
- **WHEN** 适配器观测到 provider 特有字段
- **THEN** 归一化投影忽略该字段，不新增必需 runtime 语义，也不写入 raw payload 诊断

#### Scenario: Run 与 Stream 投影等价
- **WHEN** 同一 `ModelRequest` 分别经 Run 与 Stream 路径投影
- **THEN** 归一化投影语义等价，不等价时返回 Run/Stream parity drift 分类

### Requirement: Tool-result 投影 SHALL 保持关联且可判定归属

请求投影 MUST 保留 tool result 与 tool call 的关联标识与工具身份，并 MUST 显式表达该结果是以 provider 原生 tool 归属形式还是以文本信封形式进入请求。缺失关联的 tool result MUST 在 provider 调用前以既有 `feedback_invalid` 分类失败。

#### Scenario: 关联标识在投影中可判定
- **WHEN** `ModelRequest` 携带带关联的 tool result
- **THEN** 归一化投影保留 call/name 关联并标注其归属形式（原生或文本信封）

#### Scenario: 缺失关联在 provider 调用前失败
- **WHEN** tool result 缺少 `call_id` 或工具名
- **THEN** 投影在 provider 调用前以 `feedback_invalid` 失败，且不发出部分请求

### Requirement: Usage 与 cache 可观测性 SHALL 遵守 additive + nullable + default

usage 投影 MUST 显式表达 cache 计量（cached input / cache read / cache write）是否可用。当适配器不提供 cache 计量时，投影 MUST 表达 `available=false`，MUST NOT 伪造 read/write 数值。任何新增 usage/cache 字段 MUST 遵守 `additive + nullable + default`，历史 fixture 缺失该字段 MUST 按缺省处理且解析不得失败，未知字段 MUST 被安全忽略。

#### Scenario: 当前适配器表达 cache 不可用
- **WHEN** 适配器不返回 cache 计量
- **THEN** 投影表达 `available=false` 且不填充 read/write 值

#### Scenario: 历史 fixture 缺少 cache 字段仍可解析
- **WHEN** 一个历史 fixture 未包含任何 cache 字段
- **THEN** 解析成功并按缺省语义处理，不产生 drift 分类

#### Scenario: 未知字段被安全忽略
- **WHEN** fixture 包含契约未定义的字段
- **THEN** 解析成功且该字段不影响投影、digest 或 drift 分类
