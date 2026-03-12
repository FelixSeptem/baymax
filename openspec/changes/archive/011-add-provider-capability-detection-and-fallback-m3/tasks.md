## 1. Capability Discovery Contract

- [x] 1.1 在 `core/types` 增加 provider capability 数据结构与能力匹配输入模型（覆盖 Run/Stream 共享语义）
- [x] 1.2 在 `model/openai`、`model/anthropic`、`model/gemini` 增加基于官方 SDK 的能力发现实现与统一返回映射
- [x] 1.3 增加能力发现降级规则（SDK 无法发现时返回受控 `unknown` 能力状态，不使用静态硬编码作为主来源）

## 2. Runner Preflight And Fallback

- [x] 2.1 在 runner model-step 前增加 capability preflight 判定流程
- [x] 2.2 实现 provider 候选链按序尝试逻辑（仅 model-step 级，不支持 mid-stream 切换）
- [x] 2.3 在候选链耗尽时实现统一 fail-fast 错误映射与终止语义

## 3. Runtime Config And Diagnostics

- [x] 3.1 在 `runtime/config` 增加 fallback policy 与 discovery 控制字段（含默认值、YAML/ENV 映射）
- [x] 3.2 增加 fallback/discovery 配置校验（启动与热更新路径）并保持原子回滚语义
- [x] 3.3 在 `runtime/diagnostics` 增加 capability 判定与 fallback 路径摘要字段（保持 single-writer + idempotency）

## 4. Tests And Quality Gates

- [x] 4.1 新增 capability/fallback 契约测试（成功降级、无候选 fail-fast、unknown 能力路径）
- [x] 4.2 新增 Run/Stream 一致性测试（确保 fallback 不破坏 complete-only 与事件语义）
- [x] 4.3 执行并通过 `go test ./...`、`go test -race ./...` 与 `golangci-lint run --config .golangci.yml`

## 5. Documentation Alignment

- [x] 5.1 更新 README 的多 provider 状态与 M3 说明，去除与实现不一致内容
- [x] 5.2 更新 `docs/development-roadmap.md`、`docs/v1-acceptance.md` 的 M3 状态与限制描述
- [x] 5.3 补充 capability discovery/fallback 配置与诊断字段文档（含官方 SDK 动态发现约束）
