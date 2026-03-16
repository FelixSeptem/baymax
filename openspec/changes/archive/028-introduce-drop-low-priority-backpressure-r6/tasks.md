## 1. Config and Validation

- [x] 1.1 在 `runtime/config` 扩展 `concurrency.backpressure` 枚举，新增 `drop_low_priority`
- [x] 1.2 新增 `drop_low_priority` 规则配置结构（仅配置规则：tool/keyword）与可被 drop 的优先级集合字段
- [x] 1.3 接入 `env > file > default` 解析链路与热更新快照
- [x] 1.4 为新增枚举/规则/集合字段补齐 fail-fast 校验（启动与热更新一致）
- [x] 1.5 增加配置测试：默认值、file/env 覆盖、非法配置失败路径

## 2. Local Dispatcher and Runner Semantics

- [x] 2.1 在 `tool/local` 实现 `drop_low_priority` 背压逻辑（仅 local 路径）
- [x] 2.2 按配置规则计算 call 优先级并执行 drop 决策
- [x] 2.3 加入“可被 drop 优先级集合”过滤逻辑
- [x] 2.4 在 `core/runner` 收敛全量 drop fail-fast 终止语义
- [x] 2.5 保持 Run/Stream 语义一致（drop 命中、全量 drop、非 drop 场景）

## 3. Diagnostics, Timeline, and Contract Tests

- [x] 3.1 增加 timeline reason：`backpressure.drop_low_priority`
- [x] 3.2 对齐 diagnostics 字段语义与计数（含 dropped count 对应关系）
- [x] 3.3 新增/更新契约测试：Run/Stream drop 语义一致、全量 drop fail-fast、一致错误分类
- [x] 3.4 更新 `docs/mainline-contract-test-index.md` 登记新增用例

## 4. Benchmark and Performance Gate

- [x] 4.1 增加 benchmark 对比 `block` vs `drop_low_priority` 场景
- [x] 4.2 输出并记录 `p95-ns/op` 与 `goroutine-peak`
- [x] 4.3 按相对提升百分比口径补充回归判定说明

## 5. Documentation Sync

- [x] 5.1 更新 `README.md`（新增模式、范围限制、fail-fast 语义）
- [x] 5.2 更新 `docs/runtime-config-diagnostics.md`（新增配置字段、校验语义、timeline/diagnostics 映射）
- [x] 5.3 更新 `docs/v1-acceptance.md`（新增验收条目与限制说明）
- [x] 5.4 更新 `docs/development-roadmap.md`（标记提案落点与后续 TODO）

## 6. Validation

- [x] 6.1 执行 `go test ./...` 并修复回归
- [x] 6.2 执行 `go test -race ./...` 并确认并发安全基线
- [x] 6.3 执行 `golangci-lint run --config .golangci.yml` 并修复问题
