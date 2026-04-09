# A70 Roadmap Alignment Matrix

更新时间：2026-04-09  
目标：将治理自动化需求映射到 A70 的 `proposal/design/tasks/specs`，用于评审“无重不漏”。

## Mapping Table

| ID | 需求来源 | Proposal 映射 | Design 映射 | Tasks 映射 | Spec 映射 | 覆盖结论 |
| --- | --- | --- | --- | --- | --- | --- |
| A70-R1 | roadmap 与 OpenSpec 状态需一致，避免 active/archive/candidate 漂移 | `proposal.md` 状态对账单一事实源 | Decision 1 | 2.1~2.4 | `proposal-governance-automation-contract/spec.md` Requirement `Governance Status Source of Truth SHALL Be Deterministic` | 已同步 |
| A70-R2 | 后续提案需强制评估 example 影响并可审计 | `proposal.md` example impact 声明校验 | Decision 2 | 3.1~3.4 | `proposal-governance-automation-contract/spec.md` Requirement `Proposal Example Impact Declaration SHALL Be Mandatory` | 已同步 |
| A70-R3 | 治理校验需接入阻断门禁并保持 shell/PowerShell parity | `proposal.md` gate 接线条目 | Decision 3/4 | 4.1~4.3、6.1 | `specs/go-quality-gate/spec.md` 全部 ADDED Requirements | 已同步 |
| A70-R4 | 治理结果需可追踪到文档与索引 | `proposal.md` 文档映射条目 | Migration Plan | 5.1~5.3 | `proposal-governance-automation-contract/spec.md` Requirement `Governance Checks SHALL Emit Auditable Failure Taxonomy` | 已同步 |

## Notes

- A70 是治理自动化提案，不引入 runtime 新能力。
- 后续若新增治理自动化诉求，优先在 A70 内增量吸收，避免同域平行提案。
