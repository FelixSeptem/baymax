## 1. Skill Budget Engine

- [x] 1.1 在 `skill/loader` 增加 budget mode 分支（`fixed|adaptive`），并将默认行为切换为 `adaptive`。
- [x] 1.2 实现 adaptive 预算决策（`min_k/max_k/min_score_margin`）并保证等价输入下结果确定性。
- [x] 1.3 保持 fixed 模式兼容现有 top-k 语义（复用 `max_semantic_candidates`）。
- [x] 1.4 保持 explicit 命中旁路预算裁剪语义，并保证合并后顺序与去重稳定。

## 2. Runtime Config Integration

- [x] 2.1 在 `runtime/config` 增加 `skill.trigger_scoring.budget.*` 配置结构与默认值（默认 adaptive、`min_score_margin=0.08`、`min_k=1`、`max_k=5`）。
- [x] 2.2 打通 YAML/ENV 加载映射，保持 `env > file > default` 语义。
- [x] 2.3 增加 startup/hot-reload 校验（mode、k 范围、margin 范围）并验证无效更新回滚。

## 3. Diagnostics And Event Contract

- [x] 3.1 为 skill 触发事件新增字段：`budget_mode`、`selected_semantic_count`、`score_margin_top1_top2`、`budget_decision_reason`。
- [x] 3.2 打通 `skill/loader` -> `observability/event` -> `runtime/diagnostics` 的字段映射与持久化。
- [x] 3.3 保证新增字段为 additive 扩展，不改变现有 skill lifecycle 字段语义。

## 4. Contract Tests And Regression

- [x] 4.1 新增/更新 loader 单测覆盖 adaptive 默认路径（clear winner 收缩到 `min_k`）。
- [x] 4.2 新增/更新 loader 单测覆盖 adaptive close-score 扩展路径（不超过 `max_k`）。
- [x] 4.3 新增/更新 loader 单测覆盖 fixed 模式兼容路径与 explicit 旁路语义。
- [x] 4.4 新增/更新 Run/Stream 契约测试，验证 adaptive/fixed 两种预算下语义等价。
- [x] 4.5 新增/更新 runtime config 单测覆盖默认值、env 覆盖、非法预算配置 fail-fast 与 hot-reload rollback。
- [x] 4.6 执行并通过回归门禁：`go test ./...`、`go test -race ./...` 与相关 skill/config/diagnostics 契约测试。

## 5. Docs Alignment

- [x] 5.1 更新 `docs/runtime-config-diagnostics.md` 的预算模式、默认值、校验与新增诊断字段说明。
- [x] 5.2 更新 `docs/v1-acceptance.md` 的 skill trigger scoring 能力说明（adaptive 默认 + fixed 兼容）。
- [x] 5.3 更新 `docs/development-roadmap.md` 对应进展条目，保持与提案口径一致。
