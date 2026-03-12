## 1. Streaming 适配实现（Anthropic/Gemini）

- [x] 1.1 为 `model/anthropic` 增加官方 SDK streaming 实现并接入现有模型接口
- [x] 1.2 为 `model/gemini` 增加官方 SDK streaming 实现并接入现有模型接口
- [x] 1.3 保持 fail-fast 终止语义与超时分类一致

## 2. 统一事件语义与错误分类

- [x] 2.1 对齐 OpenAI/Anthropic/Gemini 的公共流式事件语义集合
- [x] 2.2 允许新增必要 `ModelEvent.Type` 枚举并保持向后兼容
- [x] 2.3 保持“tool_call 仅完整态对外发射”，不暴露参数增量片段
- [x] 2.4 细化错误分类 reason（auth/rate_limit/timeout/request/server/unknown）

## 3. 契约测试与回归

- [x] 3.1 新增跨 provider streaming 契约测试（事件顺序/终态一致性）
- [x] 3.2 新增跨 provider fail-fast 与超时分类测试
- [x] 3.3 新增跨 provider tool-call complete-only 行为测试

## 4. 文档统一与门禁

- [x] 4.1 更新 README 中多 provider 能力说明与 M2 状态
- [x] 4.2 更新 `docs/development-roadmap.md`（M2 进展与 M3 待办）
- [x] 4.3 更新 `docs/v1-acceptance.md` 已知限制条目
- [x] 4.4 执行并通过 `go test ./...`、`golangci-lint`、`scripts/check-docs-consistency.ps1`
