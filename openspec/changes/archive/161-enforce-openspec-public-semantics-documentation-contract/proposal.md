## Why

当前 `AGENTS.md` 和 Documentation Impact gate 已要求提案评估文档影响，但“对外能力发生变化时必须同步维护文档”仍主要依赖文字约定，三份 OpenSpec 工件可以漏写、互相矛盾或只声明 `additive` 而没有维护文档证据。现在需要把这条约束提升为每个 OpenSpec change 都必须接受的、可自动验证的治理 contract。

## What Changes

- 在 `AGENTS.md` 中把公开语义文档同步承诺列为每个 OpenSpec 的强制规则。
- 为 `proposal.md`、`design.md`、`tasks.md` 增加统一的 `Public Semantic Documentation Commitment` 模板。
- 扩展 PowerShell 与 shell Documentation Impact gate，校验三份工件均声明公开语义变化状态、同步状态、维护文档、owner 和验证证据，并确认每个维护文档路径随当前 change 修改。
- 当声明存在对外语义变化时，阻断缺少维护文档路径、当前 change 的 Git 差异证据或同步证据的 change；当声明无变化时，阻断缺少具体理由的 change。
- 增加稳定失败分类码和治理 spec 场景，保证 OpenSpec 在实施和归档前都接受该约束。
- 同步治理文档和 roadmap/README 入口，说明对外语义变化与维护文档必须同 change 提交。

## Capabilities

### New Capabilities

无。此 change 只增强 OpenSpec 治理、文档模板和静态门禁，不新增运行时或 API 能力。

### Modified Capabilities

- `proposal-governance-automation-contract`: 为每个 OpenSpec 工件增加公开语义文档同步承诺及一致性校验。

## Impact

- 影响 `AGENTS.md`、`docs/documentation-style-guide.md`、`README.md`、`docs/development-roadmap.md` 和 OpenSpec 治理 spec。
- 影响 `scripts/check-openspec-documentation-impact.ps1/.sh` 及 `check-docs-consistency.*` 的 active change 校验路径。
- 不改变 Go runtime、公开 API、配置、诊断 schema、Run/Stream 语义或 `examples/agent-modes` 运行行为。
- 新增治理失败码：`missing-public-semantic-documentation-commitment`、`invalid-public-semantic-documentation-commitment`、`public-semantic-doc-sync-missing`、`public-semantic-documentation-drift`。

## Example Impact Assessment

无需示例变更（附理由）：本 change 只增强 OpenSpec 工件、维护文档承诺和静态治理门禁，不改变 `examples/agent-modes` 的配置、runtime path、expected markers 或 rollback notes。

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 无需文档变更（附理由） | — | maintainer | no runtime architecture or dependency boundary changes |
| components | 无需文档变更（附理由） | — | package owners | no component behavior changes |
| configuration | 无需文档变更（附理由） | — | runtime/config | no configuration keys or defaults change |
| contract/API | 修改文档 | `AGENTS.md`, `docs/documentation-style-guide.md`, `openspec/changes/archive/161-enforce-openspec-public-semantics-documentation-contract/specs/proposal-governance-automation-contract/spec.md` | governance maintainers | `openspec validate --strict` |
| diagnostics | 无需文档变更（附理由） | — | observability | no RuntimeRecorder or QueryRuns schema changes |
| examples | 无需文档变更（附理由） | — | example owners | Example Impact gate and no agent-mode source changes |
| CLI/integration | 修改文档 | `README.md`, `docs/development-roadmap.md` | integration/release owners | docs consistency and roadmap status gates |
| best practices | 修改文档 | `docs/documentation-style-guide.md` | maintainer | documentation impact gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | release owner | `scripts/check-openspec-roadmap-status-consistency.ps1` |

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`AGENTS.md`、`README.md`、`docs/documentation-style-guide.md`、`docs/development-roadmap.md`、`docs/guides/openspec-workflow.md`、`openspec/changes/archive/161-enforce-openspec-public-semantics-documentation-contract/specs/proposal-governance-automation-contract/spec.md`
- 维护 owner：governance maintainers
- 验证证据：双平台 documentation impact gate、docs consistency、OpenSpec strict validation
