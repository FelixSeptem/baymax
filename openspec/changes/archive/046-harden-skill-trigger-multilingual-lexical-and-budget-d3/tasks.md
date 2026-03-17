## 1. Skill Trigger Lexical And Budget Core

- [x] 1.1 在 `skill/loader` 增加 `mixed_cjk_en` 分词实现，覆盖中文与中英混合输入。
- [x] 1.2 将 lexical scorer 接入可配置分词模式，并保持现有 weighted-keyword 评分语义不变。
- [x] 1.3 在 semantic candidate 排序后增加 `max_semantic_candidates` 的 top-k 裁剪逻辑。
- [x] 1.4 保证 explicit 命中不受 semantic budget 裁剪影响，并保持去重与排序确定性。

## 2. Runtime Config Integration

- [x] 2.1 在 `runtime/config` 增加 `skill.trigger_scoring.lexical.tokenizer_mode` 与 `skill.trigger_scoring.max_semantic_candidates` 配置结构和默认值。
- [x] 2.2 打通 YAML/ENV 加载映射，保持 `env > file > default` 语义。
- [x] 2.3 增加 startup/hot-reload 校验（tokenizer_mode 枚举、max_semantic_candidates > 0）并验证无效更新回滚。

## 3. Diagnostics And Event Contract

- [x] 3.1 为 skill 触发事件新增最小字段：`tokenizer_mode`、`candidate_pruned_count`。
- [x] 3.2 打通 `skill/loader` -> `observability/event` -> `runtime/diagnostics` 的字段映射与持久化。
- [x] 3.3 保证新增字段为 additive 扩展，不改变现有 skill lifecycle 字段语义。

## 4. Contract Tests And Regression

- [x] 4.1 新增/更新 loader 单测覆盖中文与中英混合 lexical 命中路径。
- [x] 4.2 新增/更新 loader 单测覆盖 top-k 预算裁剪与 explicit 命中旁路语义。
- [x] 4.3 新增/更新 Run/Stream 契约测试，验证多语言触发与预算裁剪语义等价。
- [x] 4.4 新增/更新 runtime config 单测覆盖默认值、env 覆盖、非法配置 fail-fast 与 hot-reload rollback。
- [x] 4.5 执行并通过回归门禁：`go test ./...`、`go test -race ./...` 与相关 skill/config/diagnostics 契约测试。

## 5. Docs Alignment

- [x] 5.1 更新 `docs/runtime-config-diagnostics.md` 的 skill trigger scoring 配置、默认值、校验与诊断字段说明。
- [x] 5.2 更新 `docs/v1-acceptance.md` 的 skill trigger scoring 能力说明（multilingual lexical + semantic top-k budget）。
- [x] 5.3 更新 `docs/development-roadmap.md` 对应进展条目，保持与提案口径一致。
