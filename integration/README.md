# integration 测试说明

## 功能域

`integration` 负责跨模块合同测试与主干流程回归，覆盖：

- Run/Stream 主链路语义等价与终态收敛
- 编排链路（composer/workflow/teams/scheduler/mailbox）跨域协作行为
- 配置/准入/诊断回放等运行时治理合同
- 适配与沙箱一致性子套件（`adapterconformance`、`adaptercontractreplay`、`sandboxconformance`）

## 架构设计

- 根目录 `integration/*.go` 承载主干合同与组合回归测试。
- 子目录按能力域拆分专用 harness：
  - `integration/adapterconformance`：适配能力矩阵与漂移分类
  - `integration/adaptercontractreplay`：回放兼容与幂等收敛
  - `integration/sandboxconformance`：离线 deterministic 沙箱执行合同
- `integration/fakes` 提供跨测试复用的假实现，避免重复构造测试桩。

子套件文档索引：

- `integration/adapterconformance/README.md`
- `integration/adaptercontractreplay/README.md`
- `integration/sandboxconformance/README.md`
- `integration/fakes/README.md`

## 关键入口

- `integration/composer_contract_test.go`
- `integration/runtime_readiness_admission_contract_test.go`
- `integration/unified_snapshot_contract_test.go`
- `integration/adapterconformance/harness_test.go`
- `integration/adaptercontractreplay/replay_test.go`
- `integration/sandboxconformance/harness_test.go`
- `integration/fakes/fakes.go`

## 边界与依赖

- 优先通过公开模块入口验证行为，不把 integration 套件变成内部实现细节测试。
- integration 只承载回归与合同断言，不承载生产逻辑。
- 断言口径以稳定语义字段、reason taxonomy、幂等行为为主，避免依赖易漂移的内部细节。

## 配置与默认值

- 默认走离线、可重复执行路径，保证本地与 CI 回归可复现。
- 与治理相关的开关、阈值、字段默认值统一来源于 `runtime/config`，integration 侧只消费生效结果。
- 新增/修改 contract 时，必须同步补充对应 integration 用例与回放断言。

## 可观测性与验证

- 全量回归：`go test ./integration -count=1`
- 子套件回归：
  - `go test ./integration/adapterconformance -count=1`
  - `go test ./integration/adaptercontractreplay -count=1`
  - `go test ./integration/sandboxconformance -count=1`
- 质量门禁由 `scripts/check-quality-gate.sh` / `scripts/check-quality-gate.ps1` 聚合执行。

## 扩展点与常见误用

- 扩展点：新增主链路 contract 测试、补充能力矩阵维度、增加 replay fixture 漂移防护。
- 常见误用：只补单测不补 integration 合同，导致跨模块语义回归漏检。
- 常见误用：断言瞬时时间戳或无界统计值，造成 flaky 测试。
