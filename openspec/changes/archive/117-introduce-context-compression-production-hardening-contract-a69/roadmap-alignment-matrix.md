# A69 Roadmap Alignment Matrix

更新时间：2026-04-07  
目标：将 `docs/development-roadmap.md` 的 a69 条目映射到本 change 的 `proposal/design/tasks/specs`，用于评审“无重不漏”。

## Mapping Table

| ID | Roadmap 来源 | Proposal 映射 | Design 映射 | Tasks 映射 | Spec 映射 | 覆盖结论 |
| --- | --- | --- | --- | --- | --- | --- |
| A69-R1 | a69 目标：在不改写 A67-CTX 语义前提下，收敛 context compression production-ready | `proposal.md` Why / What Changes | Goals/Non-Goals、Decision 1 | 1.1~1.3 | `context-assembler-memory-pressure-control/spec.md` Requirement `A69 SHALL Preserve A67-CTX Semantic Invariants While Hardening Production Behavior` | 已同步 |
| A69-R2 | a69-S1：语义压缩稳定性治理 | `proposal.md` S1 条目 | Decision 2 | 2.1~2.4 | `context-assembler-memory-pressure-control/spec.md` Requirement `A69 SHALL Harden Semantic Compaction Quality and Fallback Determinism` | 已同步 |
| A69-R3 | a69-S2：冷热分层 + swap-back 相关性治理 | `proposal.md` S2 条目 | Decision 3 | 3.1~3.3 | `context-assembler-memory-pressure-control/spec.md` Requirement `A69 SHALL Govern Lifecycle Tiering and Swap-Back Retrieval Deterministically` | 已同步 |
| A69-R4 | a69-S3：冷存 retention/quota/cleanup/compact | `proposal.md` S3 条目 | Decision 4 | 4.1~4.3 | `context-assembler-memory-pressure-control/spec.md` Requirement `A69 SHALL Enforce File Cold-Store Lifecycle Governance` | 已同步 |
| A69-R5 | a69-S4：一致性与恢复治理 | `proposal.md` S4 条目 | Decision 5 | 5.1~5.3 | `diagnostics-replay-tooling/spec.md` Requirement `A69 Replay SHALL Validate Recovery and Replay Idempotency` | 已同步 |
| A69-R6 | a69-S5：配置与观测 additive 治理 | `proposal.md` S5 条目 | Decision 6 | 6.1~6.4 | `runtime-config-and-diagnostics-api/spec.md` 两项 ADDED Requirements | 已同步 |
| A69-R7 | a69-S6：强门禁治理 | `proposal.md` S6 条目 | Decision 7 | 7.1~7.4 | `go-quality-gate/spec.md` 两项 ADDED Requirements | 已同步 |
| A69-R8 | 与 a62 顺序关系：A69 前置 context-governed 验收 | `proposal.md` 与 a62 关系条目 | Decision 8 | 8.4 | a62 侧增量 requirement（引用 A69 gate/replay 输出） | 已同步 |

## Notes

- A69 聚焦“生产可用合同治理”，不承担语义扩展主责。
- 若 roadmap 对 a69 新增同域项，应优先在本 change 增量吸收，避免并行 context 压缩提案分叉。
