## Why

当前主干在业务扩展时，审计、限流、缓存、鉴权、提示增强等横切逻辑主要通过调用方重复接线完成，缺少统一的 lifecycle hook 与 tool middleware 合同，导致行为可复用性差、观测口径不一致。A61 在实施中后，下一优先级应按 roadmap 顺序推进 A65，一次性冻结 hooks/middleware 与 skill 预处理接缝，避免后续继续拆分同域提案。

## What Changes

- 新增 A65 主合同：agent lifecycle hooks + tool middleware contract。
- 冻结 lifecycle hooks 语义：`before_reasoning|after_reasoning|before_acting|after_acting|before_reply|after_reply`，并定义执行时机、失败传播与取消语义。
- 冻结 tool middleware onion-chain 语义：顺序确定、上下文透传、错误冒泡、超时隔离、短路返回与可观测输出。
- 新增配置域并保持治理一致：
  - `runtime.hooks.*`
  - `runtime.tool_middleware.*`
- 一次补齐 skill discovery source：
  - `runtime.skill.discovery.mode=agents_md|folder|hybrid`
  - `runtime.skill.discovery.roots`（目录列表）
  - 多来源合并顺序 deterministic、重复技能去重规则固定、非法路径 fail-fast + 热更新原子回滚。
- 一次补齐 Discover/Compile 预处理接缝：将其挂入 Run/Stream 前统一阶段，支持开关与失败策略：
  - `runtime.skill.preprocess.enabled`
  - `runtime.skill.preprocess.phase=before_run_stream`
  - `runtime.skill.preprocess.fail_mode=fail_fast|degrade`
- 一次补齐 SkillBundle 合同映射：
  - `SkillBundle -> prompt augmentation`
  - `SkillBundle -> tool whitelist`
  - 冲突仲裁顺序可配置且 deterministic。
- 新增配置域：
  - `runtime.skill.bundle_mapping.prompt_mode`
  - `runtime.skill.bundle_mapping.whitelist_mode`
  - `runtime.skill.bundle_mapping.conflict_policy`
- 新增 replay fixtures：`hooks_middleware.v1`、`skill_discovery_sources.v1`、`skill_preprocess_and_mapping.v1`。
- 新增 gate：`check-hooks-middleware-contract.sh/.ps1`，并接入 `check-quality-gate.*`。
- 一次性收口约束：hooks/middleware 与 skill discovery/preprocess/mapping 同域需求仅允许在 A65 增量吸收，不再新增平行提案。

## Capabilities

### New Capabilities
- `agent-lifecycle-hooks-and-tool-middleware-contract`: 统一 lifecycle hooks、tool middleware、skill discovery/preprocess/mapping 的运行时合同与治理边界。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 增加 `runtime.hooks.*`、`runtime.tool_middleware.*`、`runtime.skill.discovery.*`、`runtime.skill.preprocess.*`、`runtime.skill.bundle_mapping.*` 配置与 additive 诊断字段。
- `react-loop-and-tool-calling-parity-contract`: 增加 Discover/Compile 预处理在 Run/Stream 下等价执行与失败语义约束。
- `skill-trigger-scoring`: 增加 `agents_md|folder|hybrid` discovery source 下的评分输入一致性与去重顺序约束。
- `diagnostics-replay-tooling`: 增加 A65 fixtures 与 drift 分类断言。
- `go-quality-gate`: 增加 hooks/middleware contract gate 与 required-check 候选。

## Impact

- 代码：
  - `core/runner`（hook 生命周期挂接、Run/Stream 等价语义）
  - `tool/local`、`mcp/*`（tool middleware 接缝）
  - `skill/loader`（discovery source、preprocess、bundle 映射）
  - `runtime/config`（A65 配置域解析/校验/热更新回滚）
  - `runtime/diagnostics`、`observability/event`（A65 additive 字段）
  - `tool/diagnosticsreplay`、`integration/*`（A65 fixtures + drift tests）
  - `scripts/check-hooks-middleware-contract.*` + `scripts/check-quality-gate.*`
- 文档：
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
  - `README.md`、`skill/loader/README.md`
- 兼容性与边界：
  - 对外 API 不做 breaking；新增字段遵循 `additive + nullable + default`。
  - 不绕过 A58 precedence、A57 sandbox/allowlist/egress 与 `RuntimeRecorder` 单写入口。
  - 不引入托管 hooks/middleware 控制面或服务化编排平面，保持 `library-first`。
