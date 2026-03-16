## 1. Rule Contract and Config Schema

- [x] 1.1 在 `core/types` 增加 Action Gate 参数规则 DTO（rule/condition/operator/action）
- [x] 1.2 在 `runtime/config` 增加参数规则配置字段（含 and/or 组合结构）
- [x] 1.3 为参数规则新增 fail-fast 校验（非法 operator、空条件树、缺失 path 等）
- [x] 1.4 补充 env/file/default 优先级覆盖测试（含参数规则配置）

## 2. Runner Evaluation and Priority

- [x] 2.1 在 `core/runner` 接入参数规则判定入口，支持复合条件（AND/OR）
- [x] 2.2 实现操作符语义（eq/ne/contains/regex/in/not_in/gt/gte/lt/lte/exists）
- [x] 2.3 实现规则 action 继承逻辑（规则 action 缺省继承全局 policy）
- [x] 2.4 固化优先级顺序：参数规则 > decision_by_tool/decision_by_keyword > 现有默认路径
- [x] 2.5 保持不破坏现有 H2/H3 语义（deny/timeout/require_confirm 主流程不回归）

## 3. Observability and Diagnostics

- [x] 3.1 在 timeline 增加参数规则命中 reason code（`gate.rule_match`）
- [x] 3.2 在 `runtime/diagnostics` 增加最小字段：`gate_rule_hit_count`、`gate_rule_last_id`
- [x] 3.3 在 `run.finished` payload 与 `RuntimeRecorder` 完成新字段映射
- [x] 3.4 补充 diagnostics/recorder 字段稳定性测试

## 4. Run/Stream Contract Tests

- [x] 4.1 增加操作符语义测试（逐项覆盖）
- [x] 4.2 增加复合条件测试（AND/OR、嵌套、短路）
- [x] 4.3 增加优先级冲突测试（参数规则与 keyword/tool 冲突）
- [x] 4.4 增加规则 action 继承测试（缺省继承全局 policy）
- [x] 4.5 增加 Run/Stream 等价契约测试（命中/不命中/超时路径）

## 5. Example and Docs

- [x] 5.1 增量改造一个示例（推荐 `examples/02`）演示参数规则命中
- [x] 5.2 示例输出结构化事件并体现 `gate.rule_match`
- [x] 5.3 同步更新文档：`README.md`、`docs/runtime-config-diagnostics.md`、`docs/v1-acceptance.md`、`docs/development-roadmap.md`
- [x] 5.4 更新 `docs/mainline-contract-test-index.md`，纳入 H4 主干契约用例

## 6. Validation

- [x] 6.1 执行 `go test ./...` 并修复回归
- [x] 6.2 执行 `go test -race ./...`，保证并发安全基线
- [x] 6.3 执行 `golangci-lint run --config .golangci.yml` 并修复问题
- [x] 6.4 执行 `govulncheck ./...`（strict）并记录结果
