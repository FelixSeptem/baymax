## Why

A22 正在补齐外部 adapter conformance harness，但当前接入仍依赖人工拷贝样板与手工拼装测试骨架，导致首接入成本高、样板与契约易漂移。A23 目标是把“样板生成 + conformance 启动”标准化为可重复命令，并纳入质量门禁。

## What Changes

- 新增 adapter scaffold 生成能力，提供统一命令入口，支持 `mcp`、`model`、`tool` 三类外部适配脚手架。
- 默认输出目录为 `examples/adapters/<type>-<name>`，并保证离线、确定性生成。
- 生成内容至少包含：adapter 代码骨架、README、最小单测、conformance bootstrap 测试骨架。
- 生成器默认不覆盖已有文件；存在冲突时 fail-fast，只有显式 `--force` 才允许覆盖。
- conformance bootstrap 默认开启，生成后可直接接入 A22 harness 路径执行契约验证。
- 新增 scaffold drift 检查并接入 `check-quality-gate.sh/.ps1`，作为阻断项执行。
- 更新 README、roadmap 与契约索引，建立 “A21 样板文档 -> A23 脚手架生成 -> A22 conformance gate” 闭环。

## Capabilities

### New Capabilities
- `adapter-scaffold-generator-and-conformance-bootstrap`: 定义外部 adapter 脚手架生成契约、默认输出策略、冲突处理和 conformance bootstrap 行为。

### Modified Capabilities
- `go-quality-gate`: 增加 adapter scaffold drift 检查为跨平台阻断门禁，并要求失败即非零退出。

## Impact

- 代码与测试：
  - `cmd/adapter-scaffold/*`
  - `adapter/scaffold/*`（如采用库化实现）
  - `integration/*`（scaffold 生成与 drift 回归）
  - `scripts/check-adapter-scaffold-drift.sh`
  - `scripts/check-adapter-scaffold-drift.ps1`
  - `scripts/check-quality-gate.sh`
  - `scripts/check-quality-gate.ps1`
- 文档：
  - `README.md`
  - `docs/development-roadmap.md`
  - `docs/mainline-contract-test-index.md`
  - `docs/external-adapter-template-migration.md`（若存在导航同步项）
