## MODIFIED Requirements

### Requirement: Composite graph restore SHALL preserve causal lineage

Workflow checkpoint and restore projections SHALL retain original parent/run/session/causation identities and predecessor/successor checkpoint relationship for each paused graph. A resumed graph SHALL associate later pauses and terminal evidence with that lineage and SHALL NOT create a parallel graph or reinterpret a history cursor as graph state. When a resumed graph pauses again, the workflow owner MUST atomically persist the successor checkpoint and the predecessor's retained resume result/reference. The successor MUST be reachable only through that committed predecessor projection. A failed or interrupted publication MUST leave the prior committed state intact or return a typed recovery-required outcome; it MUST NOT expose an orphan as a resumable checkpoint or restart the graph.

#### Scenario: Successor pause preserves predecessor identity
- **WHEN** a restored workflow reaches another durable pause
- **THEN** the successor checkpoint refers to its consumed predecessor and preserves the original graph lineage

#### Scenario: Repeated pause publishes one atomic successor transition
- **WHEN** a resumed workflow reaches a later pause and the source owner commits the result
- **THEN** one owner-store transition retains both the successor checkpoint and the predecessor's result/reference
- **AND** a fresh owner can follow that reference and resume the original graph generation without rerunning completed steps

#### Scenario: Interrupted successor publication fails closed
- **WHEN** a process exits or persistence fails while publishing a resumed workflow's successor
- **THEN** reopening observes either the last committed predecessor state or the complete predecessor/successor transition
- **AND** it never treats an unreferenced successor as admissible and never reruns the workflow to reconstruct it

## Example Impact Assessment

无需示例变更（附理由）：the requirement concerns source checkpoint publication and has no changed example-visible setup or flow.

## Documentation Impact Assessment

| Area | Outcome | Maintainer | Verification | Evidence |
| --- | --- | --- | --- | --- |
| architecture | 无需文档变更（附理由） | — | — | No module boundary changes. |
| components | 修改文档 | workflow owner | `go test ./orchestration/workflow -count=1` | `orchestration/workflow/README.md` records atomic publication. |
| configuration | 修改文档 | runtime configuration owner | `pwsh -File scripts/check-docs-consistency.ps1` | `docs/runtime-config-diagnostics.md` records that no new key is introduced. |
| contract/API | 修改文档 | contract-test index owner | `pwsh -File scripts/check-quality-gate.ps1` | `docs/mainline-contract-test-index.md` maps the new gate. |
| diagnostics | 修改文档 | runtime diagnostics owner | `pwsh -File scripts/check-docs-consistency.ps1` | `docs/runtime-config-diagnostics.md` records that no diagnostic field is introduced. |
| examples | 修改文档 | native-events example owner | `go test ./orchestration/workflow -count=1` | `examples/native-execution-events/README.md` states the example remains outside owner commit. |
| CLI/integration | 无需文档变更（附理由） | — | — | No CLI or integration surface changes. |
| best practices | 无需文档变更（附理由） | — | — | No usage guidance changes beyond the workflow owner README. |
| roadmap | 修改文档 | roadmap maintainer | `pwsh -File scripts/check-docs-consistency.ps1` | `docs/development-roadmap.md` records the audited implementation direction. |

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`orchestration/workflow/README.md`, `docs/development-roadmap.md`, `docs/mainline-contract-test-index.md`, `docs/runtime-config-diagnostics.md`, `examples/native-execution-events/README.md`
- 维护 owner：workflow owner, roadmap maintainer, contract-test index owner, runtime configuration/diagnostics owner, native-events example owner
- 验证证据：`go test ./orchestration/workflow -count=1`; `pwsh -File scripts/check-docs-consistency.ps1`; `pwsh -File scripts/check-quality-gate.ps1`
