# A63 Roadmap Alignment Matrix

更新时间：2026-04-05  
目标：将 `docs/development-roadmap.md` 的 A63 范围条目映射到本 change 的 `proposal/design/tasks/specs`，用于评审“无重复、无遗漏”。

## Mapping Table

| ID | Roadmap 来源 | Proposal 映射 | Design 映射 | Tasks 映射 | Spec 映射 | 覆盖结论 |
| --- | --- | --- | --- | --- | --- | --- |
| A63-R1 | `docs/development-roadmap.md:771` Context Assembler 统一命名（不再保留 `ca1/ca2/ca3/ca4` 活动口径） | `proposal.md`「强制范围 1：Context Assembler 统一命名」 | `design.md` Goals（命名统一）+ Decision 2（语义主名收敛） | `tasks.md` 2.1/2.2/2.3/2.4 | `specs/codebase-consolidation-and-semantic-labeling-contract/spec.md` Requirement `Active Repository Surface SHALL Use Semantic Context Assembler Naming`；`specs/context-assembler-production-convergence/spec.md` | 已同步（需求已固化，实施按 tasks 执行） |
| A63-R2 | `docs/development-roadmap.md:772` Axx 字眼消除（活动目录语义化） | `proposal.md`「强制范围 2：消除 Axx」 | `design.md` Goals + Decision 3（目录分层清理） | `tasks.md` 3.1/3.2/3.3/3.4 | `specs/codebase-consolidation-and-semantic-labeling-contract/spec.md` Requirement `Active Repository Surface SHALL Replace Axx Wording with Semantic Labels` | 已同步 |
| A63-R3 | `docs/development-roadmap.md:783` 编号化保留边界（仅索引层保留编号） | `proposal.md`「Axx 仅允许存在于 openspec/**」 | `design.md` Constraints + Decision 3（`openspec/**` 允许，其他禁止） | `tasks.md` 1.2/3.2/3.3/7.2 | `specs/codebase-consolidation-and-semantic-labeling-contract/spec.md`（Axx 仅 `openspec/**`）+ `specs/go-quality-gate/spec.md`（路径/内容阻断） | 已同步 |
| A63-R4 | `docs/development-roadmap.md:776-777` 回流阻断 + 语义词表集中化 | `proposal.md`「回流阻断」「语义词表集中化」 | `design.md` Decision 1（单一映射源）+ Decision 4（门禁阻断） | `tasks.md` 1.3/7.1/7.2/7.3/7.4 | `specs/codebase-consolidation-and-semantic-labeling-contract/spec.md` Requirement `Repository SHALL Maintain a Single Semantic Mapping Source` + `Repository SHALL Block Legacy Naming Regression`；`specs/go-quality-gate/spec.md` | 已同步 |
| A63-R5 | `docs/development-roadmap.md:778` 运行时 Harness 架构总览文档收口 | `proposal.md`「运行时 Harness 架构文档收口」 | `design.md` Goal（单一总览入口）+ Decision 9（canonical 文档） | `tasks.md` 6.4/6.5 | `specs/core-module-readme-richness/spec.md` Requirement `Runtime Harness Architecture SHALL Have One Canonical Documentation Entry` | 已同步（新增对齐项） |
| A63-R6 | `docs/development-roadmap.md:769-770/774` 临时文档、离线生成物、临时占位清理 | `proposal.md`「临时文档/目录治理」「离线生成物治理」「临时注释与占位清理」 | `design.md` Context/Goals（清理临时与过时资产） | `tasks.md` 5.1/5.2/5.3/6.1/12.1/12.2/12.3 | `specs/core-module-readme-richness/spec.md`（README 对齐与路径收敛）+ `specs/go-quality-gate/spec.md`（门禁约束） | 已同步 |
| A63-R7 | 用户补充范围（仅 `*.go` 大文件拆分，语义/逻辑不变强校验） | `proposal.md`「过大文件拆分治理」「拆分后语义不变」 | `design.md` Goal（仅治理 `*.go`）+ Decision 8（行数预算/超限拆分） | `tasks.md` 13.1~13.8 | `specs/codebase-consolidation-and-semantic-labeling-contract/spec.md` Requirement `Repository SHALL Govern Single-File Code Size...` + `specs/go-quality-gate/spec.md`（强语义等价门禁） | 已同步 |

## Notes

- 本矩阵仅用于 A63 的 roadmap 对齐审计，不替代 `proposal/design/tasks/specs` 本体。  
- 若 roadmap 再增补 A63 同域需求，必须在本文件追加一行映射，并同步更新对应 `proposal/design/tasks/specs`。
