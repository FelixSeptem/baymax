## 1. Action Gate Core Contract

- [x] 1.1 在 `core/types` 定义 Action Gate 判定与确认接口（matcher/resolver）及最小决策模型（allow/require_confirm/deny）
- [x] 1.2 在 `core/runner` 接入工具执行前 gate 评估链路，未命中高风险规则时保持原行为
- [x] 1.3 实现默认 `require_confirm` 策略，未配置 resolver 时按 deny fail-fast

## 2. Timeout and Run/Stream Semantic Convergence

- [x] 2.1 在 runner 中实现 gate resolver 超时控制，超时统一按 deny 处理
- [x] 2.2 收敛 Run/Stream 的 gate allow/deny/timeout 行为与错误分类语义
- [x] 2.3 增加 Run/Stream 契约测试，验证语义等价与回归稳定

## 3. Risk Rule Scope (H2 Minimal)

- [x] 3.1 实现首期风险判定规则：仅 `tool name + keyword`（不包含参数 schema）
- [x] 3.2 增加规则命中/未命中边界测试，避免误拦截与漏拦截

## 4. Config, Timeline, and Diagnostics

- [x] 4.1 在 `runtime/config` 增加 Action Gate 配置字段与校验（默认 `require_confirm`、timeout-deny）
- [x] 4.2 在 timeline 事件中新增 gate reason code（`gate.require_confirm`、`gate.denied`、`gate.timeout`）
- [x] 4.3 在 `runtime/diagnostics` 增加最小字段：`gate_checks`、`gate_denied_count`、`gate_timeout_count`
- [x] 4.4 增加 recorder/diagnostics 对应测试，保证字段映射稳定

## 5. Validation and Quality Gate

- [x] 5.1 执行 `go test ./...`，修复回归直到通过
- [x] 5.2 执行 `go test -race ./...`，确保并发安全基线不回退
- [x] 5.3 执行 `golangci-lint run --config .golangci.yml` 并修复问题
- [x] 5.4 执行 `govulncheck ./...`（strict）并记录结果

## 6. Documentation Consistency

- [x] 6.1 同步更新 `README.md` 的 Action Gate/HITL 能力说明
- [x] 6.2 同步更新 `docs/runtime-config-diagnostics.md` 的配置与诊断字段
- [x] 6.3 同步更新 `docs/v1-acceptance.md` 与 `docs/development-roadmap.md`，确保文档与实现一致
