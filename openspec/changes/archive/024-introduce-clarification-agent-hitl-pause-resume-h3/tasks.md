## 1. H3 Core Contract

- [x] 1.1 在 `core/types` 定义 clarification 相关接口与最小 DTO（request/response/resolver）
- [x] 1.2 在 `core/runner` 增加 `await_user -> resumed -> canceled_by_user` 生命周期
- [x] 1.3 默认超时策略落地为 `cancel_by_user`（fail-fast）

## 2. Run/Stream Semantic Convergence

- [x] 2.1 收敛 Run/Stream 在 await/resume/cancel 下的行为与错误分类
- [x] 2.2 为 Run/Stream 增加等价契约测试（含 timeout 场景）
- [x] 2.3 确保不破坏现有 H2 Action Gate 与 tool dispatch 语义

## 3. Event and Diagnostics

- [x] 3.1 增加结构化 `clarification_request` payload（最小字段）
- [x] 3.2 在 timeline 中增加 H3 reason code（await/resumed/cancel_by_user）
- [x] 3.3 在 `runtime/diagnostics` 增加最小计数：`await_count`、`resume_count`、`cancel_by_user_count`
- [x] 3.4 补齐 recorder/diagnostics 字段映射测试

## 4. Runtime Config

- [x] 4.1 在 `runtime/config` 增加 H3 配置字段与校验（`enabled`、`timeout`、`timeout_policy`）
- [x] 4.2 默认配置为启用 H3 且 `timeout_policy=cancel_by_user`
- [x] 4.3 增加 env/file/default 优先级与非法配置 fail-fast 测试

## 5. Multi-Agent Example

- [x] 5.1 增量改造 `examples/07-multi-agent-async-channel`，加入 clarification-agent 流程
- [x] 5.2 示例输出结构化事件，包含 `clarification_request` 与恢复路径
- [x] 5.3 示例可编译可运行（最小演示即可）

## 6. Validation and Docs

- [x] 6.1 执行 `go test ./...` 并修复回归
- [x] 6.2 执行 `go test -race ./...`，保证并发安全基线
- [x] 6.3 执行 `golangci-lint run --config .golangci.yml` 并修复问题
- [x] 6.4 执行 `govulncheck ./...`（strict）并记录结果
- [x] 6.5 同步更新 `README.md`、`docs/runtime-config-diagnostics.md`、`docs/v1-acceptance.md`、`docs/development-roadmap.md`、`docs/examples-expansion-plan.md`
