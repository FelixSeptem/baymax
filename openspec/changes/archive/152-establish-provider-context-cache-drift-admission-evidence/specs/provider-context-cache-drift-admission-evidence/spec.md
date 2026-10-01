## Purpose

本能力以离线、引用式、可复放的准入证据，将新的 Provider SDK 边界或宿主性能观察与既有请求投影契约关联，并确定性地决定是否存在足以启动最小增量修复的漂移。

## ADDED Requirements

### Requirement: Provider context/cache evidence SHALL be versioned, bounded, and reference-only

系统 MUST 支持版本化 fixture `provider_context_cache_evidence.v1`。每条 evidence MUST 使用受支持的 Provider 身份、有限的 SDK 版本标识、既有 `provider_request_projection.v1` case identity 与 canonical digest 引用，以及有限的结构化上下文、cache parity 或宿主性能维度。fixture MUST 以稳定 canonical serialization 与 digest 归一化，并 MUST 拒绝未知版本、重复冲突 identity、无效引用、超界集合、无界字符串、raw prompt/reasoning/credential/full SDK payload 或原始性能日志。

#### Scenario: Reference-only evidence is accepted deterministically
- **WHEN** evidence 引用存在的 projection case 和相同 canonical digest，且所有字段在边界内
- **THEN** replay 生成稳定的 canonical evidence 与 digest，且不调用 Provider、工具、网络或 runtime

#### Scenario: Unsafe or unresolved evidence is rejected
- **WHEN** evidence 缺少 projection reference、digest 不匹配、含原始 payload，或任一集合/字段超出边界
- **THEN** replay 以稳定 schema、reference 或 privacy drift 分类 fail-fast，且不产生部分 verdict

### Requirement: Evidence verdict SHALL distinguish no-drift, confirmed drift, and insufficient evidence

系统 MUST 仅从已验证的 evidence dimension 导出 `no-drift`、`drift-confirmed` 或 `insufficient-evidence`。当任一已验证维度与其既有 projection reference、Run/Stream cache parity expectation 或有界宿主基线阈值矛盾时，verdict MUST 为 `drift-confirmed` 并列出稳定 reason code；当声明为必需的维度缺失、不可验证或没有可信成本样本时，verdict MUST 为 `insufficient-evidence`；只有所有声明为必需的维度都被验证且未发现矛盾时，verdict MUST 为 `no-drift`。系统 MUST NOT 将 evidence 缺失、未知 SDK 字段或 unavailable cache usage 推导为 `no-drift`。

#### Scenario: Missing required cache-cost evidence remains insufficient
- **WHEN** fixture 声明宿主 cache 成本/P95 为必需维度，但没有满足样本数、单位和阈值边界的摘要
- **THEN** replay 输出 `insufficient-evidence`，且不合成成本、P95 或 cache 命中数据

#### Scenario: Verified projection contradiction confirms drift
- **WHEN** 一个已验证的 role、Skill/tail、tool-result association、stable ordering 或 Run/Stream cache parity 维度与其 projection reference 矛盾
- **THEN** replay 输出 `drift-confirmed` 与稳定维度 reason code，并保留导致结论的 reference identity

#### Scenario: Fully verified evidence reports no drift
- **WHEN** 每个声明为必需的维度均可由受信 reference 或有界宿主摘要验证，且没有矛盾
- **THEN** replay 输出 `no-drift`，并在重复运行和两种受支持 shell 中得到同一结论

### Requirement: Host performance evidence SHALL remain bounded and unit-explicit

可选宿主性能证据 MUST 仅包含有限的样本数、显式单位、baseline/observed P50 与 P95 摘要、阈值和关联的 projection digest；它 MUST NOT 包含价格表、逐请求成本、原始时序、用户标识或 provider response。replay MUST 验证非负性、P50/P95 顺序、样本数下限、unit 一致性和阈值计算；不一致或超界的数据 MUST 以稳定 cost-evidence drift 分类失败，而不是静默校正。

#### Scenario: Stable P95 regression confirms drift
- **WHEN** 完整的宿主摘要满足样本和单位要求，且 observed P95 超过声明阈值
- **THEN** replay 输出 `drift-confirmed` 的稳定 cost/P95 reason code，而不修改 cache 策略、价格模型或 runtime 配置

#### Scenario: Mixed units cannot be compared
- **WHEN** baseline 与 observed 摘要使用不同单位，或阈值无法按 fixture 定义计算
- **THEN** replay 拒绝该 evidence 并输出稳定 cost-evidence drift 分类

### Requirement: Admission replay and gate SHALL be offline, deterministic, and non-remediating

replay 与 contribution gate MUST 离线、只读、确定性地验证 `provider_context_cache_evidence.v1`，并与既有 `provider_request_projection.v1` historical fixture 行为兼容。shell 和 PowerShell gate MUST 对相同输入给出相同 verdict、canonical digest 和 reason codes。`drift-confirmed` 只能产生指向既有 `model/<provider>` owner 的 review-only 修复建议；系统 MUST NOT 自动改写 adapter、Prompt、Skill、Tool、Policy、配置、fixture expectation、gate 或代码。

#### Scenario: Gate parity preserves a confirmed-drift outcome
- **WHEN** 同一有效 evidence fixture 在 shell 与 PowerShell gate 中运行
- **THEN** 两者产生相同的 `drift-confirmed` verdict、digest 和稳定 reason codes，且不发生任何 runtime 或文件状态修改

#### Scenario: Historical request projection fixtures remain valid
- **WHEN** replay 同时处理历史 `provider_request_projection.v1` fixture 和新的 evidence fixture
- **THEN** 缺少 evidence 的历史 fixture 保持其既有解析、digest 和 drift 分类，不被解释为 admission verdict

### Requirement: Verification baseline exception renewal SHALL be narrow and time-bounded

为完成本 change 的验证，仓库 MAY 仅对既有 A63 staged-split line-budget exception rows 做一次明确续期。续期 MUST 只更新已存在路径的 expiry，保留原 owner、reason、baseline_lines 与 allow_growth；不得新增例外、提高 warn/hard 阈值、改变原本禁止增长的文件，或将任何质量门禁失败静默解释为通过。续期 MUST 具有不晚于 `2026-12-31` 的复审日期，并在 change tasks 中记录后续拆分或重新审查责任。

#### Scenario: Existing line-budget exceptions are renewed without widening scope

- **WHEN** 当前 change 的 quality gate 因既有 A63 staged-split exception expiry 到期而阻断
- **THEN** 仅清单中既有的 11 条超长文件路径可被续期至 `2026-12-31`，其 owner、reason、baseline 和 allow_growth 保持不变，新增路径或阈值变化均被视为越界

#### Scenario: Line-budget governance remains fail-fast after the renewal window

- **WHEN** 续期日期到达或任何文件超出原有 allow_growth/baseline 约束
- **THEN** `check-go-file-line-budget` 与完整 quality gate MUST 继续失败并要求拆分或重新审查，而不是自动放行

## Example Impact Assessment

无需示例变更（附理由）：该 capability 仅定义离线 evidence/replay/gate 的可观察行为，不改变 agent-mode 示例的配置、runtime path、expected markers 或 rollback notes。
