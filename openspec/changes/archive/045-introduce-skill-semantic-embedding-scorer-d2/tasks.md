## 1. Skill Trigger Scoring Core

- [x] 1.1 在 `skill/loader` 定义 embedding scorer 扩展接口与注册入口（保持未注册可回退）。
- [x] 1.2 扩展策略枚举，支持 `lexical_plus_embedding` 且默认仍为 `lexical_weighted_keywords`。
- [x] 1.3 实现线性加权融合 `final_score = lexical_weight * lexical + embedding_weight * embedding`。
- [x] 1.4 实现 embedding 失败归一化回退（missing/timeout/error/invalid_score）并保持 compile 主流程 best-effort。

## 2. Runtime Config Integration

- [x] 2.1 在 `runtime/config` 增加 `skill.trigger_scoring.embedding.*` 配置结构与默认值。
- [x] 2.2 打通 YAML/ENV 加载映射，保持 `env > file > default` 语义。
- [x] 2.3 增加 startup/hot-reload 校验（timeout、metric、weights 等）并验证无效更新回滚。

## 3. Diagnostics And Event Mapping

- [x] 3.1 为 skill 触发新增最小观测字段：`strategy`、`final_score`、`embedding_score`、`fallback_reason`。
- [x] 3.2 打通 `skill/loader` -> `observability/event` -> `runtime/diagnostics` 的字段映射与持久化。
- [x] 3.3 保证新增字段为 additive 扩展，不改变现有 skill lifecycle 字段语义。

## 4. Contract Tests And Regression

- [x] 4.1 新增/更新 loader 单测覆盖 lexical+embedding 成功路径与线性融合正确性。
- [x] 4.2 新增/更新 loader 单测覆盖 embedding 回退路径（missing/timeout/error/invalid_score）。
- [x] 4.3 新增/更新 Run/Stream 契约测试，验证 skill 触发与诊断语义等价。
- [x] 4.4 执行并通过回归门禁：`go test ./...`、`go test -race ./...` 与相关 skill/config/diagnostics 契约测试。

## 5. Docs Alignment

- [x] 5.1 更新 `docs/runtime-config-diagnostics.md` 的 skill embedding scoring 配置、默认值、校验与诊断字段说明。
- [x] 5.2 更新 `docs/v1-acceptance.md` 的 skill trigger scoring 能力说明（默认 lexical + 可选 embedding 增强）。
- [x] 5.3 更新 `docs/development-roadmap.md` 对应进展条目，保持与提案口径一致。
