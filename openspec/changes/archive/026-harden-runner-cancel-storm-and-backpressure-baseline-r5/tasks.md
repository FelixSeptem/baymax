## 1. Config Baseline and Validation

- [x] 1.1 在 `runtime/config` 增加取消风暴与背压基线字段，并固定默认背压策略为 `block`
- [x] 1.2 保持并验证 `env > file > default` 优先级行为（含新增字段）
- [x] 1.3 为新增枚举/阈值字段补齐 fail-fast 校验（非法模式、非法范围、空值约束）
- [x] 1.4 增加配置加载与校验测试，覆盖默认值、覆盖优先级、非法输入失败路径

## 2. Runner Concurrency and Cancellation Convergence

- [x] 2.1 在 `core/runner` 收敛统一背压入口，达到并发限制时按 `block` 语义处理
- [x] 2.2 在 `core/runner` 收敛取消传播逻辑，确保父上下文取消后不再接收新 dispatch
- [x] 2.3 将 `tool`、`mcp`、`skill` 三条路径纳入统一取消传播行为
- [x] 2.4 保持 Run/Stream 语义等价，覆盖取消、超时、并发受限路径
- [x] 2.5 预留 `drop_low_priority` 扩展 TODO（不在本期实现策略行为）

## 3. Diagnostics and Timeline Mapping

- [x] 3.1 在 `runtime/diagnostics` 增加最小字段：`cancel_propagated_count`、`backpressure_drop_count`、`inflight_peak`
- [x] 3.2 在 `run.finished` payload 与 `observability/event.RuntimeRecorder` 完成字段映射
- [x] 3.3 在 timeline 增加取消传播与背压结果的可观测语义映射，保证与 diagnostics 一致
- [x] 3.4 补齐 recorder/store 稳定性测试，验证字段幂等与零值兼容

## 4. Contract, Integration, and Pressure Testing

- [x] 4.1 增加 Run/Stream 契约测试：取消风暴下语义一致性（含 tool/mcp/skill）
- [x] 4.2 增加高 fanout + 取消风暴集成测试，验证 goroutine 收敛与无新增泄漏
- [x] 4.3 增加背压命中测试，验证默认 `block` 行为与 `backpressure_drop_count=0`
- [x] 4.4 将 `p95 latency` 与 `goroutine peak` 纳入基线回归检查并形成可比输出
- [x] 4.5 保持既有主干能力不回归（CA/HITL/provider 路径 smoke 覆盖）

## 5. Documentation and Contract Index Sync

- [x] 5.1 更新 `README.md`：补充取消风暴与背压基线说明（默认 `block`）
- [x] 5.2 更新 `docs/runtime-config-diagnostics.md`：新增配置字段与诊断字段口径
- [x] 5.3 更新 `docs/v1-acceptance.md`：补充 R5 验收项（Run/Stream 一致、p95、goroutine peak）
- [x] 5.4 更新 `docs/mainline-contract-test-index.md`：登记新增主干契约测试
- [x] 5.5 更新 `docs/development-roadmap.md`：记录提案落点与后续 TODO（`drop_low_priority`）

## 6. Validation

- [x] 6.1 执行 `go test ./...` 并修复回归
- [x] 6.2 执行 `go test -race ./...` 并确认并发安全基线
- [x] 6.3 执行 `golangci-lint run --config .golangci.yml` 并修复问题
- [x] 6.4 运行对应性能/压力检查脚本并记录 `p95 latency` 与 `goroutine peak` 结果
