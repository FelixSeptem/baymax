## MODIFIED Requirements

### Requirement: Request-side conformance SHALL be observable through existing adapter seams

请求侧审计 MUST 复用既有注入缝隙（OpenAI 的 SDK 参数级缝隙、Anthropic/Gemini 的适配器→SDK 边界注入），MUST NOT 新增导出 API、MUST NOT 新增运行时配置键。门禁 MUST 静态校验三家适配器的 SDK 请求构造仍归属各自 `model/<provider>` 包，且 canonical request-facts 归一化与校验不依赖 provider SDK。

三家 supported adapter MUST 使用各自官方 SDK 可表达的原生请求形状保留 `ModelRequest.Messages` 的 system/user/assistant 角色、稳定顺序、最终用户输入和有效 tool-result 的 call/name 关联。Run 与 Stream MUST 产生语义等价的归一化请求投影；CountTokens 若可用，MUST 采用相同的 canonical request facts。门禁与离线 replay MUST 使用版本化 fixture 证明归档 142 中已修复的 declared gap 已被显式迁移，且不得记录 raw prompt、reasoning、credentials 或无界 provider payload。

#### Scenario: SDK-boundary capture proves native projection
- **WHEN** 三家 adapter 接收包含角色消息与有效 tool result 的同一 canonical request facts
- **THEN** 各自注入缝隙捕获到的 SDK-boundary request 都保留可比较的角色、顺序和原生结果关联事实

#### Scenario: Run Stream parity regression is blocked
- **WHEN** 同一 adapter 的 Run 与 Stream 路径在角色、最终输入位置或 tool-result 关联上产生不同归一化投影
- **THEN** conformance suite 以稳定的 Run/Stream parity drift 分类失败

#### Scenario: Provider-owned construction boundary is preserved
- **WHEN** 实现新增或修改请求映射代码
- **THEN** provider SDK 类型仅存在于相应的 `model/<provider>` 包，SDK-neutral canonical facts 不形成共享 provider wire protocol

#### Scenario: 审计不新增运行时 API
- **WHEN** 请求侧 conformance 审计被执行
- **THEN** 审计仅依赖既有 SDK-boundary 注入缝隙完成，未新增导出 API、配置键或平行运行时分支

#### Scenario: 适配器所有权被门禁保护
- **WHEN** SDK 请求构造逻辑被移动到 `model/<provider>` 之外的包
- **THEN** 对等 gate 失败并给出稳定失败分类
