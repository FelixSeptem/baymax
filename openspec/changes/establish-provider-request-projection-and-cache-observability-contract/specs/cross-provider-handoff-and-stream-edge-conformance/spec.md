## ADDED Requirements

### Requirement: Cross-provider conformance SHALL cover request-side projection

已归档的跨 Provider conformance 覆盖响应侧 handoff 与 stream edge；本能力 MUST 扩展至请求侧投影，使 supported adapter 的 `ModelRequest` → provider SDK request 投影属于同一 conformance 家族，可被同一套 fixture、离线 replay 与 gate 覆盖。请求侧与响应侧 MUST 共享 provider-neutral 归一化与稳定 drift 分类口径，MUST NOT 各自定义平行语义。

#### Scenario: 请求侧与响应侧共享归一化家族
- **WHEN** 同一 provider 同时产生请求侧与响应侧投影证据
- **THEN** 二者使用同一 provider-neutral 归一化层与同一 drift 词表，不存在第二套分类

#### Scenario: 请求侧 provider-only 字段被隔离
- **WHEN** provider SDK 请求形状包含 provider 特有字段
- **THEN** conformance 归一化不把该字段提升为必需 runtime 语义，也不写入 raw payload 诊断

### Requirement: Request-side conformance SHALL be observable through existing adapter seams

请求侧审计 MUST 复用既有注入缝隙（OpenAI 的 SDK 参数级缝隙、Anthropic/Gemini 的适配器→SDK 边界注入），MUST NOT 新增导出 API、MUST NOT 新增运行时配置键，MUST NOT 为审计目的改变适配器运行时投影行为。门禁 MUST 静态校验三家适配器的 SDK 请求构造仍归属各自 `model/<provider>` 包，且 canonical 输入构造仍是唯一入口。

#### Scenario: 审计不新增运行时 API
- **WHEN** 请求侧审计被执行
- **THEN** 审计仅依赖既有缝隙完成，未新增导出 API、配置键或运行时分支

#### Scenario: 适配器所有权被门禁保护
- **WHEN** SDK 请求构造逻辑被移动到 `model/<provider>` 之外的包
- **THEN** 对等 gate 失败并给出稳定失败分类
