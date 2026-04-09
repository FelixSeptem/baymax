# A62 Roadmap Alignment Matrix

更新时间：2026-04-07  
目标：将 `docs/development-roadmap.md` 的 a62 条目映射到本 change 的 `proposal/design/tasks/specs`，用于评审“无重不漏”。

## Mapping Table

| ID | Roadmap 来源 | Proposal 映射 | Design 映射 | Tasks 映射 | Spec 映射 | 覆盖结论 |
| --- | --- | --- | --- | --- | --- | --- |
| A62-R1 | `docs/development-roadmap.md` a62 目标与模式总览（PocketFlow + Baymax 扩展） | `proposal.md` `What Changes`（模式族覆盖） | Decision 1/2 | 1.1~1.3、2.1~2.9 | `delivery-usability-agent-mode-example-pack-contract/spec.md` Requirement `PocketFlow and Baymax Mode Families SHALL Be Fully Covered` | 已同步 |
| A62-R2 | `docs/development-roadmap.md` 双档示例要求（minimal + production-ish） | `proposal.md`（双档示例） | Decision 1 | 1.3、4.1、4.6 | `delivery-usability-agent-mode-example-pack-contract/spec.md` Requirement `Each Mode SHALL Provide Minimal and Production-ish Variants` | 已同步 |
| A62-R3 | `docs/development-roadmap.md` 主干流程覆盖（mailbox/task-board/scheduler/readiness） | `proposal.md`（主干流程覆盖） | Decision 3 | 3.14~3.17 | `delivery-usability-agent-mode-example-pack-contract/spec.md` Requirement `Mainline Flow Examples SHALL Cover Canonical Orchestration Paths` | 已同步 |
| A62-R4 | `docs/development-roadmap.md` 自定义 adapter 覆盖（mcp/model/tool/memory + health-circuit） | `proposal.md`（自定义 adapter 覆盖） | Decision 3 | 3.18~3.19 | `delivery-usability-agent-mode-example-pack-contract/spec.md` Requirement `Custom Adapter Examples SHALL Cover Four Adapter Domains and Health Circuit` | 已同步 |
| A62-R5 | `docs/development-roadmap.md` 示例语义必须复用既有 contract，不定义平行语义 | `proposal.md`（硬约束） | Decision 2 | 4.2、5.3 | `delivery-usability-agent-mode-example-pack-contract/spec.md` Requirement `Example Outputs SHALL Reuse Existing Contract Semantics` | 已同步 |
| A62-R6 | `docs/development-roadmap.md` 迁移手册与 `prod delta` 检查清单增强项 | `proposal.md`（PLAYBOOK + prod delta） | Decision 4 | 4.5~4.7 | `delivery-usability-agent-mode-example-pack-contract/spec.md` Requirement `Migration Playbook SHALL Define Example-to-Production Checklist` + `Production-ish Readmes SHALL Declare Prod Delta Checklist` | 已同步 |
| A62-R7 | `docs/development-roadmap.md` A62 门禁（smoke + pattern coverage + migration-playbook consistency） | `proposal.md`（三类门禁） | Decision 5 | 1.4、1.5、4.7、4.8、5.1 | `specs/go-quality-gate/spec.md` 全部 ADDED Requirements | 已同步 |
| A62-R8 | `docs/development-roadmap.md` A62 同域需求内闭环，不新增平行示例提案 | `proposal.md`（A62 主合同闭环） | Goals/Non-Goals | 5.5（收口验收） | `delivery-usability-agent-mode-example-pack-contract/spec.md` 统一矩阵与覆盖要求 | 已同步 |
| A62-R9 | `docs/development-roadmap.md` 历史示例 TODO 占位清理与回流阻断 | `proposal.md`（历史占位清零 + legacy cleanup gate） | Decision 6 | 1.7、4.10、4.11、5.2 | `delivery-usability-agent-mode-example-pack-contract/spec.md` Requirement `Legacy Example TODO Placeholders SHALL Be Eliminated` + `specs/go-quality-gate/spec.md` Requirement `A62 Legacy Example TODO Cleanup Gate Is Mandatory` | 已同步（本轮新增） |
| A62-R10 | `docs/development-roadmap.md` a69 前置于 a62 context-governed 验收 | `proposal.md`（a62 与 a69 依赖边界） | Decision 7 | 2.6、5.6、5.7 | `delivery-usability-agent-mode-example-pack-contract/spec.md` Requirement `Context-Governed Example Completion SHALL Depend on A69 Production Convergence` + `specs/go-quality-gate/spec.md` Requirement `A62 Context-Governed Validation SHALL Require A69 Context Compression Gates` | 已同步（本轮新增） |
| A62-R11 | A62 治理新增：后续提案需评估是否修改/新增 example | `proposal.md`（提案级示例影响评估约束） | Decision 8 | 5.8、5.9 | `delivery-usability-agent-mode-example-pack-contract/spec.md` Requirement `Future Contract Proposals SHALL Declare Example Impact Assessment` | 已同步（本轮新增） |
| A62-R12 | A62 治理新增：示例回归时延/波动超阈值时，不新开平行提案，直接在 a62 内吸收稳定性治理 | `proposal.md`（示例稳定性/性能治理条件触发约束） | Decision 9 | 5.10、5.11、5.12 | `delivery-usability-agent-mode-example-pack-contract/spec.md` Requirement `Example Stability and Regression Performance Governance SHALL Be Absorbed Within A62` + `specs/go-quality-gate/spec.md` Requirement `A62 Example Stability Governance Gate SHALL Be Triggered by Baseline Breach` | 已同步（本轮新增） |

## Notes

- A62 作为交付易用性收口项，仅复用既有 contract 语义，不承担 runtime 语义重定义。
- `context-governed` 子项完成判定依赖 A69 context compression 生产合同收敛；非 context 子项可并行推进。
- 若 roadmap 新增 A62 同域条目，应优先在本 change 中以增量任务吸收，并同步更新本矩阵映射行。
