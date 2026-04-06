# Context Assembler Evolution Index

更新时间：2026-04-05

> 说明：本文件已从“分期计划”收敛为“现状索引”。
> 当前主线事实以 `docs/development-roadmap.md` 与 `docs/runtime-config-diagnostics.md` 为准。

## 当前能力地图

- Prefix baseline assembly：在 Run/Stream pre-model hook 进行前缀一致性校验与 append-only journal 写入。
- Stage routing：采用 Stage1 + 条件 Stage2 路由，支持 `best_effort` 与 `fail_fast` 策略。
- Context pressure and recovery：提供分区压力治理、压缩/裁剪/落盘回填、阈值治理与诊断输出。
- Production hardening：通过契约测试、回放校验与基准回归门禁维持语义稳定。

## 历史阶段映射（仅检索）

| 历史阶段编号 | 当前语义名称 | Canonical 入口 |
| --- | --- | --- |
| stage-1 | Prefix baseline assembly | `docs/runtime-config-diagnostics.md` |
| stage-2 | Stage routing + external retriever | `docs/context-stage2-external-retriever-evolution.md` |
| stage-3 | Context pressure and recovery | `docs/runtime-config-diagnostics.md` |
| stage-4 | Production hardening and gates | `docs/development-roadmap.md` |

## Canonical 文档入口

- `docs/development-roadmap.md`
- `docs/runtime-config-diagnostics.md`
- `docs/runtime-harness-architecture.md`
- `docs/context-stage2-external-retriever-evolution.md`
- `docs/mainline-contract-test-index.md`

## 维护规则

- 本文件仅保留索引与导航，不再维护阶段性实施细节。
- 新行为、字段或门禁变化必须落到对应 canonical 文档，并在 roadmap 更新状态口径。
- 历史编号与变更编号映射仅保留在 `openspec/**`。
