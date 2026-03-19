## Why

在 A11-A18 主链路能力基本收敛后，仓库缺少“多代理主链路”统一性能回归门禁，当前仅有 CA4 专项 benchmark gate，无法系统覆盖同步/异步/延后/恢复路径。A19 目标是在 lib-first 前提下补齐主链路性能基线治理，避免后续迭代引入隐性延迟和放大回归。

## What Changes

- 新增多代理主链路 benchmark 能力，覆盖同步调用、异步回报、延后调度、恢复重放四类关键路径。
- 新增多代理性能基线文件与回归脚本（Shell + PowerShell）并纳入仓库标准门禁。
- 定义统一回归阈值与判定口径：`ns/op`、`p95-ns/op`、`allocs/op` 相对百分比比较。
- 固化默认执行参数：`benchtime=200ms`、`count=5`，支持环境变量覆盖。
- 固化基线缺失与参数非法行为：fail-fast，阻断质量门禁。
- 更新性能策略与主干契约索引文档，建立“主链路 -> benchmark/gate”可追溯映射。

## Capabilities

### New Capabilities
- `multi-agent-mainline-performance-baseline`: 定义多代理主链路 benchmark 矩阵、基线文件、阈值比较与失败语义。

### Modified Capabilities
- `go-quality-gate`: 将多代理主链路性能回归 gate 纳入标准质量门禁与 CI 阻断路径。

## Impact

- 代码：
  - `integration/benchmark_test.go`（新增主链路基准）
  - `scripts/check-multi-agent-performance-regression.sh`
  - `scripts/check-multi-agent-performance-regression.ps1`
  - `scripts/multi-agent-benchmark-baseline.env`
  - `scripts/check-quality-gate.sh`
  - `scripts/check-quality-gate.ps1`
- 测试与验证：
  - benchmark regression 脚本本地/CI 一致执行
  - `go test ./integration -run ^$ -bench ...` 扩展覆盖
- 文档：
  - `docs/performance-policy.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/development-roadmap.md`
  - `README.md`
