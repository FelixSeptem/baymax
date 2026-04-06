# runtime/security 组件说明

## 功能域

`runtime/security` 当前聚焦脱敏基础能力，提供统一 redaction 组件：

- 结构化 payload 脱敏（map / nested object）
- JSON 文本脱敏
- 关键字匹配与可扩展 matcher
- sandbox egress / adapter allowlist 治理语义的安全域约束说明（配置与执行实现在 `runtime/config`、`core/runner`、`adapter/manifest`）
- policy precedence / decision trace 治理边界说明（precedence evaluator 在 `runtime/config`，执行透传在 `core/runner`）

Canonical 架构入口：`docs/runtime-harness-architecture.md`

## 架构设计

实现位于 `runtime/security/redaction`：

- `Redactor` 基于关键词 token 规则判断敏感 key
- 命中后使用统一掩码值 `***`
- 支持默认关键词和运行时自定义关键词
- 支持 `Matcher` 扩展额外匹配策略

该能力被 `runtime/config.Manager`、`context/assembler`、`observability/event` 复用。

## 关键入口

- `redaction/redactor.go`

子模块文档：

- `redaction/README.md`

## 边界与依赖

- 安全脱敏是横切能力，必须保持纯函数行为和稳定输出语义。
- 不在该域混入调度、模型或传输逻辑，避免安全域职责扩散。
- 新增策略时需保证不破坏现有 key 分词匹配语义。
- sandbox egress/allowlist 治理边界遵循同一原则：`runtime/security` 不直接承载 egress 判定与 allowlist 激活状态机，仅维护安全能力边界说明；实际判定由上层合同模块落地。
- policy precedence 治理边界保持一致：`runtime/security` 不承载 precedence 评估器或 winner 计算，不直接写入 `policy_decision_path`；仅维持安全域能力边界，决策链路由 `runtime/config` + `core/runner` + `observability/event` 统一实现。

## 治理边界说明

sandbox egress 与 `adapter.allowlist.*` 配置域属于“安全治理合同”，但实现分层保持不变：

- 配置解析与 fail-fast/回滚：`runtime/config`
- readiness/admission finding 与 deterministic mapping：`runtime/config/readiness`
- 执行期 egress action 判定与 additive 字段写入：`core/runner` + `observability/event`
- adapter allowlist 激活前校验：`adapter/manifest`
- replay 与 gate：`tool/diagnosticsreplay` + `scripts/check-sandbox-egress-allowlist-contract.*`
- policy precedence replay 与 gate：`tool/diagnosticsreplay` + `scripts/check-policy-precedence-contract.*`

该分层保证 `runtime/security` 不越界依赖 runner/adapter，同时保持安全合同可回放、可门禁、可审计。

## 配置与默认值

- N/A：当前模块未单独暴露 runtime 配置键，主要通过调用方注入关键词与 matcher。
- 默认掩码值为 `***`，默认关键词集合覆盖常见凭证与密钥字段。

## 可观测性与验证

- 关键验证：`go test ./runtime/security/redaction -count=1`。
- 调用侧（config/observability/context）会复用同一脱敏器，确保输出一致性。
- 回归重点：嵌套结构脱敏、JSON 文本脱敏、matcher 扩展兼容。
- 安全治理联调：`pwsh -File scripts/check-sandbox-egress-allowlist-contract.ps1`（或 shell 同名脚本）验证 egress/allowlist 合同阻断语义。
- precedence 治理联调：`pwsh -File scripts/check-policy-precedence-contract.ps1`（或 shell 同名脚本）验证跨层 winner 与 decision trace 合同稳定性。

## 扩展点与常见误用

- 扩展点：自定义 matcher、扩展关键词词典、引入域特定脱敏策略。
- 常见误用：在业务层手工脱敏并与统一 redactor 混用，导致口径不一致。
- 常见误用：把 redaction 逻辑与业务状态机耦合，降低可复用性。
