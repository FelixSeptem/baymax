## Why

随着多轮迭代叠加（multi-provider、CA1/CA2/CA3、timeline、diagnostics、安全门禁），当前实现需要一次按“模块职责 + 端到端主干流程”收敛的代码评审与修复，避免语义漂移、边界回退和测试盲区继续累积。现在做这件事可以在功能扩展前先稳住基线，降低后续变更成本。

## What Changes

- 新增一次性收敛的模块化代码评审与修复流程：按 `core`、`context`、`model`、`runtime`、`observability` 五个主模块开展 review，并输出可执行问题清单（P0/P1/P2）。
- 新增端到端主干流程串联评审矩阵，覆盖 `Run`、`Stream`、`tool-loop`、`CA2 external retriever`、`CA3 pressure/recovery` 等关键路径。
- 收敛修复策略升级为同一提案内一次完成 `P0 + P1 + P2`，不拆分后续批次。
- 强化契约测试要求：主干流程必须具备契约测试覆盖，确保 Run/Stream 语义一致、错误分类一致、诊断与 timeline 一致。
- 纳入仓库卫生治理：清理临时/备份产物（如 `*.go.<random>`）并防止回流。
- 同步更新 README 与 `docs/` 全量受影响文档，保证实现与文档无漂移。

## Capabilities

### New Capabilities
- `modular-e2e-review-alignment`: 定义“模块化评审 + 端到端串联验证 + 一次性修复收敛”的统一执行标准与验收口径。

### Modified Capabilities
- `go-quality-gate`: 质量门禁新增“主干流程契约测试覆盖”和“仓库卫生检查”作为必选基线。
- `runtime-module-boundaries`: 强化职责边界核验要求，明确评审过程需覆盖跨模块责任泄漏与依赖方向回归。

## Impact

- 代码范围：`core/*`、`context/*`、`model/*`、`runtime/*`、`observability/*` 及其对应测试目录。
- 测试范围：新增/补强主干流程契约测试与回归测试，持续执行 `go test ./...`、`go test -race ./...`、`golangci-lint`。
- 文档范围：`README.md` + `docs/` 下所有受影响页面（含流程、边界、质量门禁、诊断说明）。
- 工程治理：清理无效脚本/临时文件，降低噪音与误导风险。
