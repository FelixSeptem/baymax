## Why

当前仓库功能已较完整，但新接入者在“如何使用核心 API”与“如何快速复盘一次运行”上仍存在明显成本。进入 DX D1 阶段后，需要用最小变更补齐可读文档与可复盘调试闭环，降低维护与协作摩擦。

## What Changes

- 新增 API 参考覆盖基线：优先覆盖 `core/*`、`runtime/*`、`context/*`，并额外覆盖 `skill/*` 的 godoc 与最小示例索引。
- 新增 diagnostics replay 最小能力：支持 JSON 输入，输出精简视图（`phase/status/reason/timestamp` + 最小关联 ID）。
- 增加 replay 契约测试：固定输入输出，保证回放结果稳定可回归。
- CI 增加 replay 质量门禁并作为 required check 候选（建议 job 名 `diagnostics-replay-gate`）。
- README/docs 更新为中文优先口径并接受英文内容，提供最小排障路径说明。

## Capabilities

### New Capabilities

- `api-reference-coverage`: 定义核心包 API 参考覆盖基线与示例索引要求，覆盖 `core/runtime/context/skill`。
- `diagnostics-replay-tooling`: 定义 diagnostics JSON 回放输入契约、精简输出语义与稳定错误码行为。

### Modified Capabilities

- `go-quality-gate`: 增加 diagnostics replay 契约校验作为 CI 阻断检查项，并支持 required status check 收敛。

## Impact

- Affected docs:
  - `README.md`
  - `docs/*`（API 参考、调试路径、回放使用说明）
- Affected code:
  - 新增 replay helper/tooling（库优先，可选最小入口）
  - 相关单元测试与契约测试
- Affected CI:
  - `.github/workflows/ci.yml` 新增或扩展 replay gate job
- Runtime 主流程语义与现有 provider/runner/context 执行链路不变。
