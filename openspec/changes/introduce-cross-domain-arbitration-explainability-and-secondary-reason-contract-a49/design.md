## Context

A47 正在构建 readiness/timeout/adapter-health 的组合回放夹具，A48 正在冻结 primary reason 裁决优先级。当前缺口是解释层：在 primary reason 已确定后，系统缺少统一 secondary reasons 语义与 remediation hint taxonomy，导致同一输入在不同路径或版本下解释文本与辅助字段不稳定。

A49 通过定义 explainability 契约，把“secondary candidate 为什么被降级/未选中”转成可测试、可回放、可阻断的结构化语义。

## Goals / Non-Goals

**Goals:**
- 固化 secondary reasons 的有界输出规则（最大数量、稳定排序、去重规则）。
- 固化 arbitration explainability 字段集与 canonical hint taxonomy。
- 保持 Run/Stream/replay explainability 输出语义一致。
- 将 explainability drift 纳入 quality gate 阻断。

**Non-Goals:**
- 不改变 A48 的 primary precedence 规则与业务终态机。
- 不引入平台控制面、外部状态存储或多租户运维面板。
- 不提供任意自由文本解释（避免高基数与不可比）。

## Decisions

### Decision 1: Secondary reasons 固定有界输出

- 方案：
  - `max_secondary_reasons=3`（固定上限）
  - canonical code 排序
  - 去重后输出
- 原因：平衡可解释性与诊断体积控制。

### Decision 2: Explainability 使用结构化 hint taxonomy

- 方案：新增 `runtime_remediation_hint_code`、`runtime_remediation_hint_domain`，值来自受控集合。
- 原因：利于自动化处理和跨版本稳定对账。

### Decision 3: Rule version 强制输出

- 方案：输出 `runtime_arbitration_rule_version` 并纳入 replay 断言。
- 原因：避免规则升级后 fixture 难以解释差异来源。

### Decision 4: 漂移默认 fail-fast

- 方案：secondary 顺序漂移、hint taxonomy 漂移、rule version 不一致均阻断。
- 原因：A49 是收敛提案，需保证解释层稳定性。

## Risks / Trade-offs

- [Risk] 受控 hint taxonomy 初期覆盖不足
  -> Mitigation: 保留 additive 扩展窗口，但新 code 必须通过 OpenSpec 变更。

- [Risk] secondary reasons 固定上限丢失部分信息
  -> Mitigation: 以 deterministic top-N 输出 + count 字段保留规模信息。

- [Risk] explainability 字段增加 replay fixture 维护成本
  -> Mitigation: 与 A47 夹具复用，新增专用 explainability case 分层管理。

## Migration Plan

1. 在 runtime 层实现 secondary reasons/hint/rule version 组装逻辑。
2. 在 diagnostics 与 recorder 写入 additive explainability 字段。
3. 在 replay tooling 增加 explainability 规范化比较与 drift 分类。
4. 在 integration + gate 接入 explainability contract suites。
5. 更新文档索引与状态快照。

## Open Questions

- 是否在后续阶段支持按 domain 输出多条 remediation hints（本提案先保持单条 primary hint）。
- 是否在 A50+ 开放 profile 级 explainability 策略覆写（本提案不做）。
