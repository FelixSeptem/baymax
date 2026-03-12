## 1. Provider 适配实现（M1 非流式）

- [x] 1.1 新增 `model/anthropic` 最小非流式适配，优先使用官方 SDK
- [x] 1.2 新增 `model/gemini` 最小非流式适配，优先使用官方 SDK
- [x] 1.3 保持统一模型接口调用语义，不新增 streaming 占位 API

## 2. 错误语义与 TODO 占位

- [x] 2.1 将 Anthropic/Gemini 错误映射到基础 `types.ErrorClass` 集合
- [x] 2.2 在代码中增加 M2 TODO：后续细化 provider 特有错误分类
- [x] 2.3 在文档中增加 M2 TODO：后续补齐多 provider streaming 与细粒度错误映射

## 3. 契约测试

- [x] 3.1 新增跨 provider 契约测试（OpenAI/Anthropic/Gemini）最小成功路径
- [x] 3.2 新增跨 provider 基础错误分类一致性测试（认证/限流/超时/请求错误/未知）
- [x] 3.3 保持真实 provider 在线 smoke 为可选非阻塞检查

## 4. 文档与质量门禁

- [x] 4.1 更新 README 的 provider 能力矩阵与 M1 边界说明
- [x] 4.2 更新 `docs/development-roadmap.md` 的 R3 M1 状态与 M2 待办说明
- [x] 4.3 更新 `docs/v1-acceptance.md` 的已实现能力与已知限制
- [x] 4.4 执行并通过 `go test ./...`、`golangci-lint`、`scripts/check-docs-consistency.ps1`
