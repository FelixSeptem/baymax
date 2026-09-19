## ADDED Requirements

### Requirement: Replay SHALL support provider request projection fixtures

Diagnostics replay MUST 解析版本化 `provider_request_projection.v1` fixture，覆盖三家 supported provider × Run/Stream、role 投影、tool-result 归属形式、part 顺序、稳定前缀、工具顺序、能力投影、cache usage 可用性、overflow 与 declared gap 一致性。Replay MUST 离线、只读、确定性，MUST 与历史 fixture 版本兼容，MUST NOT 调用 provider/工具或修改运行时状态。

#### Scenario: 合法请求投影 fixture 可回放
- **WHEN** replay 收到合法且 digest 自洽的 `provider_request_projection.v1` fixture
- **THEN** replay 成功完成，不触发网络调用、不修改运行时状态、不产生副作用

#### Scenario: 非法 fixture 快速失败
- **WHEN** provider、mode、source/observed、correlation、边界或 declared gap 字段缺失或非法
- **THEN** replay 以确定性 schema 校验失败返回，且不产生部分成功结果

#### Scenario: 重复回放幂等
- **WHEN** 同一 fixture 被重复回放
- **THEN** 归一化结果与 drift/gap 分类完全一致，计数器与源状态不增长

### Requirement: Replay SHALL classify request-side drift canonically

Replay MUST 至少分类以下稳定码：`provider_request_schema_drift`、`provider_request_role_projection_drift`、`provider_request_tool_result_native_drift`、`provider_request_part_ordering_drift`、`provider_request_stable_prefix_drift`、`provider_request_tool_order_drift`、`provider_request_capability_projection_drift`、`provider_request_run_stream_parity_drift`、`provider_cache_usage_projection_drift`、`provider_request_overflow_drift`、`provider_request_contract_drift`。码集合 MUST 与 `model/conformance` 常量、spec 与本 gate 文档一致。

#### Scenario: 请求侧漂移被分类
- **WHEN** 归一化投影与 fixture 期望在 role、tool-result 归属、part 顺序、稳定前缀、工具顺序、能力、cache usage 或 parity 上不一致
- **THEN** replay 返回对应的稳定 drift 分类

#### Scenario: 分类词表漂移被阻断
- **WHEN** drift 码集合与契约常量或文档不再一致
- **THEN** 对等 gate 失败并给出 taxonomy drift 分类

### Requirement: Replay SHALL pin declared request projection gaps

Replay MUST 校验 `declared_gap` 与计算出的 gap 分类严格一致，MUST 在未申报缺口时失败，MUST 在已申报缺口不再成立时失败。Replay MUST NOT 为已申报缺口输出任何「通过即合规」或「等价语义」的表达。

#### Scenario: 缺口缺失申报
- **WHEN** 计算出的 gap 分类非空而用例未声明
- **THEN** replay 返回对应 drift 分类并失败

#### Scenario: 缺口被静默修复
- **WHEN** 用例声明了缺口但当前投影已不再产生该缺口
- **THEN** replay 返回 `provider_request_contract_drift` 并失败，要求显式更新契约
