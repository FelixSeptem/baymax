## Purpose

为宿主显式模型路由意图提供有界、确定性、可回放的 admission evidence，判断现有 catalog 与 admission 事实是否满足意图，并在证据不足或确有表达缺口时给出稳定结论，而不改变运行时模型选择。

## ADDED Requirements

### Requirement: Route intent evidence SHALL be versioned, bounded, and provider-neutral

系统 MUST 接受版本化的 `model_route_intent_admission.v1` 输入，包含宿主路由意图、catalog generation、允许候选、能力需求、credential/readiness/admission 事实及期望比较范围。归一化结果 MUST 仅包含稳定身份、枚举、digest/length、ordinals 与有界 reason codes，不得包含 provider SDK 类型、endpoint、credential material、raw provider payload、prompt、reasoning 或无界 body。

#### Scenario: Equivalent evidence normalizes identically
- **WHEN** 同一意图、候选、generation 与 admission facts 以等价顺序提交
- **THEN** 归一化意图、事实 digest、reason 顺序与 evidence identity 完全一致

#### Scenario: Evidence exceeds a declared bound
- **WHEN** 意图字段、候选数、能力数、reason 数或序列化大小超出上界
- **THEN** 系统快速失败并返回稳定 overflow 分类，且不返回部分 evidence

### Requirement: Route intent comparison SHALL produce stable verdicts

比较 MUST 只使用宿主提供的意图与既有 catalog/admission facts，输出且仅输出 `satisfied`、`route-gap-confirmed` 或 `insufficient-evidence` 顶层 verdict，并附有界、可排序的字段级 reason。比较不得按 caller order 隐式选模，不得执行 discovery、网络、provider、credential probe、时钟、文件或 runtime mutation。

#### Scenario: Existing facts satisfy explicit intent
- **WHEN** 目标身份、允许候选、能力、credential、readiness、generation 与 Run/Stream facts 均满足意图
- **THEN** evidence 返回 `satisfied`，并记录匹配的 normalized identity 与 generation

#### Scenario: Existing facts prove an expressiveness gap
- **WHEN** catalog/admission facts 可证明宿主意图要求的明确路由选择、约束或 parity 事实无法由现有合同表达
- **THEN** evidence 返回 `route-gap-confirmed`，仅列出缺失的稳定 reason，不执行路由或选择

#### Scenario: Facts are incomplete or contradictory
- **WHEN** 必需事实缺失、generation 不一致、credential/readiness 状态矛盾或期望范围无法判定
- **THEN** evidence 返回 `insufficient-evidence`，不得推断为满足或缺口已证实

### Requirement: Evidence SHALL preserve admission and Run/Stream semantics

Evidence MUST 复用现有 capability、credential、fallback、readiness、catalog generation 与 resolver 事实；不得定义第二套 admission、readiness 或 terminal-state machine。等价 Run/Stream 输入 MUST 产生语义等价的 verdict、selected identity、fallback identity、generation 与 reason sequence。

#### Scenario: Run and Stream facts are equivalent
- **WHEN** Run 与 Stream 对同一意图提供等价 catalog/admission facts
- **THEN** 两者 evidence verdict、identity、generation 与有序 reasons 等价

#### Scenario: Generation changes after an admitted fact
- **WHEN** 后续 catalog generation 发布于已记录 admission fact 之后
- **THEN** evidence 保留原 generation，不把后续快照混入既有比较

### Requirement: Replay SHALL detect semantic drift and preserve privacy

Replay MUST 支持 success、gap、insufficient-evidence、overflow、privacy、generation mismatch 与 Run/Stream parity fixture。重复回放同一 fixture MUST 产生相同 canonical digest 与分类；历史 fixture 缺少可选字段时 MUST 使用文档化默认值，未知字段 MUST 安全忽略。

#### Scenario: Replay detects verdict or reason drift
- **WHEN** 实际 verdict、identity、generation、reason order 或 parity 与 fixture expectation 不同
- **THEN** replay 失败并返回稳定的 route-intent drift 分类，且不产生副作用

#### Scenario: Privacy violation is rejected
- **WHEN** fixture 含 endpoint、credential material、raw response、prompt、reasoning 或无界 body
- **THEN** replay 快速失败并返回 privacy 分类，不输出部分成功结果
