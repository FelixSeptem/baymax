## Why

当前 `openspec` 状态与 roadmap/README 里程碑快照仍可能出现人工维护漂移，同时核心模块 README 仍偏“目录索引化”，缺少可直接用于接入与扩展的深度信息。A25 目标是在 `0.x` 治理口径下补齐“状态一致性 + 模块文档深度”双重可执行约束，并纳入阻断门禁。

## What Changes

- 新增 release status parity 治理能力：统一 `openspec` 与 `docs/development-roadmap.md`、`README.md` 的状态口径。
- 定义核心模块 README 丰富化基线，要求核心模块文档包含可执行接入信息、配置/诊断语义与边界说明。
- 将 status parity 与模块 README rich-check 接入 `check-docs-consistency.*` 和 `check-quality-gate.*` 阻断路径。
- 扩展 `tool/contributioncheck` 覆盖状态一致性与模块 README 必填段落约束。
- 更新主干契约索引，建立“状态对齐 gate + module README gate”的可追溯映射。

## Capabilities

### New Capabilities
- `release-status-parity-governance`: 定义 OpenSpec 实时状态与 roadmap/README 进度快照之间的一致性契约。
- `core-module-readme-richness`: 定义核心模块 README 的最小信息深度、必填段落与可追溯要求。

### Modified Capabilities
- `go-quality-gate`: 增加状态口径一致性检查和核心模块 README 丰富化检查的阻断要求。

## Impact

- 脚本与校验：
  - `scripts/check-docs-consistency.sh`
  - `scripts/check-docs-consistency.ps1`
  - `scripts/check-quality-gate.sh`
  - `scripts/check-quality-gate.ps1`
  - `tool/contributioncheck/*`
- 文档：
  - `docs/development-roadmap.md`
  - `README.md`
  - `docs/mainline-contract-test-index.md`
  - 核心模块 README（`a2a/`, `core/*`, `tool/local`, `mcp`, `model`, `context`, `orchestration`, `runtime/*`, `observability`, `skill/loader`）
