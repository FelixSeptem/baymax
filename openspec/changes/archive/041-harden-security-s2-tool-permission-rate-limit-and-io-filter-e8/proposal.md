## Why

当前安全基线（S1）已经覆盖依赖扫描与脱敏，但 R3 仍缺少“工具调用可治理”与“模型 I/O 可过滤”的统一策略闭环：不同工具缺乏可审计权限边界，频率限制未形成可配置门禁，模型输入输出缺少标准化过滤扩展口。进入 Security S2 阶段，需要在不破坏现有 library-first 架构的前提下，补齐可配置、可热更新、可回滚、可观测、可门禁的安全治理能力。

## What Changes

- 新增 S2 工具安全治理能力：支持 `namespace+tool` 级别权限策略（allow/deny）与调用频率限制策略。
- 新增 S2 频率限制行为约束：默认进程级（`process`）计数作用域；超限行为采用 `deny`（fail-fast）。
- 新增模型输入/输出安全过滤扩展口：统一 input/output filter 接口与策略装配点，先支持可插拔实现与默认最小策略。
- 统一运行时配置治理：安全策略默认治理模式为 `enforce`，支持热更新立即生效；无效配置更新必须回滚到上一有效快照。
- 增强诊断可观测：新增工具权限拒绝、限流拒绝、I/O 过滤命中等标准化字段与 reason code。
- CI 增加独立安全契约门禁（required check 候选）：验证权限/限流/过滤策略语义与热更新回滚语义。

## Capabilities

### New Capabilities

- `tool-security-governance-s2`: 定义 `namespace+tool` 粒度的工具权限策略与进程级频率限制策略，包括超限 `deny` 语义与最小可观测字段。
- `model-io-security-filtering`: 定义模型输入输出过滤扩展接口、策略执行顺序、命中行为与错误语义。

### Modified Capabilities

- `runtime-config-and-diagnostics-api`: 扩展 S2 安全配置字段与诊断字段，明确热更新“成功原子切换 / 失败回滚”在安全治理配置上的适用语义。
- `go-quality-gate`: 增加 S2 安全契约测试门禁，并以独立 CI job 暴露为 required check 候选。

## Impact

- Affected code:
  - `runtime/config/*`（安全治理与 I/O 过滤配置模型、校验、热更新装配）
  - `core/runner/*`（工具调用前权限与限流决策、拒绝路径）
  - `guardrails/*` 或等价安全过滤模块（输入/输出过滤接口与执行）
  - `observability/*`（新增安全治理诊断字段与 reason code）
- Affected tests:
  - 工具权限策略契约测试、限流契约测试、I/O 过滤契约测试
  - 配置热更新回滚语义测试（无效更新不污染生效快照）
- Affected CI:
  - `.github/workflows/*` 增加独立 `security-policy-gate`（命名可在实施阶段落定）
- Compatibility:
  - 该变更为 pre-1.x 阶段新增安全能力，采用增量字段与可配置策略，不引入既有公共 API 的破坏性更改。