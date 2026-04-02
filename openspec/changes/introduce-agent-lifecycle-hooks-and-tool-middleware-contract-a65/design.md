## Context

A61（OTel tracing + agent eval interop）在实施中，主干下一个顺位是 A65（hooks + middleware）。当前运行时虽然已有 Runner、Tool Dispatch、Skill Loader、Security/Sandbox、Diagnostics 等稳定合同，但横切逻辑仍缺少统一扩展面：
- 业务侧需要在推理前后、工具调用前后、回复前后做审计/限流/缓存/提示增强时，通常只能在调用方重复拼接；
- tool 调用链缺少统一 middleware 语义，难以保证顺序、超时、错误冒泡和观测字段一致；
- skill 侧 discovery/preprocess/bundle mapping 的接缝分散，`AGENTS.md` 与目录加载并存但未冻结统一配置合同。

A65 的设计目标是在不引入平台控制面的前提下，一次性冻结 hooks/middleware 与 skill 预处理映射合同，使其与 A57/A58/A59/A60/A61 已有语义保持可组合且可回放。

## Goals / Non-Goals

**Goals:**
- 冻结 lifecycle hooks 合同：`before_reasoning|after_reasoning|before_acting|after_acting|before_reply|after_reply`。
- 冻结 tool middleware onion-chain 合同：顺序、短路、超时、错误冒泡、上下文透传。
- 冻结 skill discovery source 合同：`agents_md|folder|hybrid` + deterministic merge/dedup。
- 将 `Discover/Compile` 统一挂到 Run/Stream 前预处理阶段，保证双入口语义等价。
- 冻结 `SkillBundle -> prompt augmentation / tool whitelist` 映射与冲突仲裁。
- 补齐配置、诊断、回放、门禁：A65 fixtures + `check-hooks-middleware-contract.*`。
- 维持 `library-first + contract-first`，确保同域需求后续仅在 A65 增量吸收。

**Non-Goals:**
- 不引入托管 hooks/middleware 控制面、远程编排服务或平台化 UI/RBAC。
- 不重写 A58 precedence、A57 sandbox/allowlist/egress、A61 tracing/eval 语义。
- 不在 A65 引入与 A64 重叠的性能专项优化（仅保留可观测与合同稳定性）。
- 不在 A65 展开 example pack 收口（A62 负责）。

## Decisions

### Decision 1: Hook 执行采用固定生命周期点 + deterministic 失败策略

- 方案：在 Runner 主循环固定 6 个 hook 点位；每个 hook 输出统一状态（success/failed/skipped）并记录 reason。
- 备选：允许调用方自由注入任意 hook 点。
- 取舍：固定点位更利于 Run/Stream 等价与 replay 稳定，避免 hook 点漂移导致不可回放。

### Decision 2: Tool middleware 采用 onion-chain，单向责任边界清晰

- 方案：中间件按注册顺序进入、逆序退出；支持短路返回；错误按 canonical 分类向上冒泡；超时由统一 budget/timeout 约束。
- 备选：并行 middleware 或事件总线式 middleware。
- 取舍：onion-chain 简化语义、易测试、可与现有 tool dispatch 串接；并行 middleware 难以保证 deterministic 顺序。

### Decision 3: Skill discovery source 一次收口到 `agents_md|folder|hybrid`

- 方案：统一 discovery mode + roots 配置；`hybrid` 下明确 merge 顺序与 dedup key；非法路径 fail-fast + 热更新回滚。
- 备选：保留各入口独立配置，不做统一口径。
- 取舍：统一口径可避免多入口分叉与重复提案，同时保持 `AGENTS.md` 向后兼容。

### Decision 4: Discover/Compile 前置为 Run/Stream 统一预处理阶段

- 方案：在执行前阶段按开关执行 discover-only 或 discover+compile；`fail_fast|degrade` 由配置决定；Run/Stream 共用同一编排函数。
- 备选：仅在 Run 或仅在 Stream 接入。
- 取舍：单入口接入会造成语义分叉，违反既有主线“Run/Stream 等价”治理。

### Decision 5: SkillBundle 映射采用显式模式 + 冲突仲裁策略

- 方案：分别定义 prompt augmentation 与 tool whitelist 的映射模式，并显式配置冲突策略（优先级/并集/拒绝）。
- 备选：隐式映射（运行时自动推断）。
- 取舍：显式策略更可解释、可回放、可门禁，避免隐式推断导致 nondeterministic 行为。

### Decision 6: 观测与门禁走现有主线，不新增平行数据面

- 方案：所有 hooks/middleware/skill 预处理事件仅通过 `RuntimeRecorder` 单写入口进入 diagnostics；新增 A65 fixtures 与独立 gate，并接入 quality gate。
- 备选：单独 hooks telemetry 管线。
- 取舍：平行数据面会破坏现有 contract/replay 稳定性，增加解释冲突风险。

## Risks / Trade-offs

- [Risk] Hook 与 middleware 可扩展性提高后，业务方可能滥用导致链路复杂化。  
  → Mitigation: 固定生命周期点、限制 middleware 责任边界、增加 fail-fast 校验与门禁示例。

- [Risk] `hybrid` discovery 在大规模 skill 目录下可能引入加载开销。  
  → Mitigation: A65 先冻结语义与回放；性能治理统一纳入 A64 子项吸收。

- [Risk] 预处理 `fail_mode=degrade` 可能造成“部分生效”认知偏差。  
  → Mitigation: 输出明确的 preprocess 状态字段与 reason taxonomy，并在 replay/gate 固化断言。

- [Risk] whitelist 映射与 sandbox/allowlist 上界冲突。  
  → Mitigation: 强制执行“白名单映射不得突破 A57 上界”，冲突时 deterministic deny 并记录来源。

- [Risk] 并行实施中出现与 A61 字段解释冲突。  
  → Mitigation: A65 仅新增 additive 字段，禁止重定义 A61/A58 同义字段，并通过 parser compatibility 测试阻断。

## Migration Plan

1. 配置层：在 `runtime/config` 增加 A65 配置域与校验规则，覆盖 default/env/file 与热更新回滚。
2. 运行时层：在 Runner/Tool Dispatch 接入 hooks 与 middleware 编排；在 Skill Loader 接入 discovery source、preprocess、bundle mapping。
3. 观测层：在 `runtime/diagnostics` 与 `observability/event` 增加 A65 additive 字段并保持单写入口。
4. 回放层：在 `tool/diagnosticsreplay` 增加 `hooks_middleware.v1`、`skill_discovery_sources.v1`、`skill_preprocess_and_mapping.v1`。
5. 门禁层：新增 `check-hooks-middleware-contract.sh/.ps1` 并接入 `check-quality-gate.*`。
6. 文档层：同步 `runtime-config-diagnostics`、contract index、roadmap、README 与 `skill/loader/README.md`。

回滚策略：
- 配置回滚：沿用 runtime manager 现有热更新原子回滚；
- 行为回滚：通过 `runtime.hooks.*`、`runtime.tool_middleware.*`、`runtime.skill.preprocess.enabled` 开关回退到 A65 前路径；
- 数据兼容：新增字段保持 `additive + nullable + default`，可在旧解析器下安全忽略。

## Open Questions

- None. 本提案按 roadmap 占位口径一次性收口 hooks/middleware + skill 预处理映射同域需求，不再拆分平行提案。
