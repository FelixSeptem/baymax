## Context

A49 已把 cross-domain arbitration 的 explainability 字段（含 `rule_version`）纳入主线，但当前只解决“输出了哪个版本”，未解决“为什么选这个版本、哪些版本允许被选、版本不兼容时如何收敛”。  
在 0.x 持续演进阶段，后续 arbitration 规则升级会成为高频行为；若缺少版本治理契约，Run/Stream/replay 语义会出现跨版本漂移且无法稳定回归。

## Goals / Non-Goals

**Goals:**
- 固化 arbitration 规则版本解析与兼容窗口语义（requested/default/effective/source）。
- 固化不兼容处理策略（unsupported/mismatch）并默认 fail-fast。
- 将版本治理信号接入 readiness preflight/admission、diagnostics、replay、quality gate。
- 保持 Run/Stream、memory/file、recovery/replay 的版本解释一致性。

**Non-Goals:**
- 不改变 A49 已冻结的 primary/secondary 裁决业务终态语义。
- 不引入平台化控制面（UI/RBAC/多租户运维）。
- 不承诺无限历史版本兼容（仅支持受控 compatibility window）。

## Decisions

### Decision 1: 新增 arbitration version resolver，并固定解析链路

- 方案：
  - 新增 `runtime.arbitration.version.*` 配置域：
    - `enabled`（默认 `true`）
    - `default`（默认 `a49.v1`）
    - `compat_window`（默认 `1`）
    - `on_unsupported`（默认 `fail_fast`）
    - `on_mismatch`（默认 `fail_fast`）
  - 版本解析固定输出：`requested/effective/source/policy_action`。
- 取舍：
  - 相比“仅输出 effective version”方案，增加了治理字段，但可显著降低 drift 排障成本。

### Decision 2: 兼容判定使用“规则注册表 + 窗口”双约束

- 方案：
  - arbitration rule 版本必须在运行时注册表中存在；
  - requested version 需满足 compatibility window，否则判定为 unsupported/mismatch。
- 取舍：
  - 纯 allowlist 方案更简单，但升级时运维成本更高；
  - “注册表 + 窗口”可平衡可控升级与配置复杂度。

### Decision 3: 版本不兼容默认 fail-fast，并将策略显式可观测

- 方案：
  - `unsupported_version` 与 `compatibility_mismatch` 默认阻断；
  - 诊断字段写入 `runtime_arbitration_rule_policy_action`，避免“行为变了但诊断无证据”。
- 取舍：
  - fail-fast 短期会更严格，但能避免 silent downgrade 导致语义漂移。

### Decision 4: replay 增加 cross-version fixture 及 drift 分类

- 方案：
  - 新增 A50 fixture schema（`a50.v1`），覆盖 default/requested/unsupported/mismatch 路径；
  - 漂移分类固定为：`version_mismatch`、`unsupported_version`、`cross_version_semantic_drift`。
- 取舍：
  - fixture 维护成本上升，但可把“版本治理回归”前移到 gate 阻断。

## Risks / Trade-offs

- [Risk] 兼容窗口配置与注册版本不一致，导致误阻断。  
  -> Mitigation: 启动与热更新阶段统一 fail-fast 校验，非法更新原子回滚。

- [Risk] 新增字段提高 diagnostics 体积。  
  -> Mitigation: 采用 additive + bounded 字段，并复用 A45 截断治理策略。

- [Risk] 旧 fixture 无版本治理字段时对账成本增加。  
  -> Mitigation: 保持 A49 fixture 兼容窗口；A50 新增字段采用 nullable/default，增量引入 cross-version suites。

## Migration Plan

1. 在 `runtime/config` 增加 arbitration version 配置域与解析/校验逻辑（启动 + 热更新 + 回滚）。
2. 在 readiness preflight/admission 路径接入 resolver 输出与 canonical finding 映射。
3. 在 `runtime/diagnostics` 与 `RuntimeRecorder` 增加版本治理 additive 字段并保持 replay idempotency。
4. 在 `tool/diagnosticsreplay` 增加 `a50.v1` fixture 与 drift 分类断言。
5. 在 `integration` 与 `check-quality-gate.*` 接入 A50 contract suites（Run/Stream parity、memory/file parity、cross-version drift guard）。
6. 更新 `docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`、`README.md`。

## Open Questions

- 是否在 A50 内支持 `on_unsupported=fallback_default` 的非阻断策略，还是先仅保留 `fail_fast`（当前建议保留枚举，默认 fail-fast）？
- compatibility window 是否允许按 domain 粒度覆写（当前建议仅全局窗口，避免策略分裂）？
