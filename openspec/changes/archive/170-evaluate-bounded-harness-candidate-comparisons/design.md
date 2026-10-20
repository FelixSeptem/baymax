## Context

See `proposal.md` for motivation. The existing evaluation contract already owns versioned corpus/Badcase data, deterministic experiment comparison, continuity/attribution references, and review-only feedback. This design adds a bounded evaluation protocol on those owners; it does not add a runtime execution owner.

## Goals / Non-Goals

**Goals:**
- Make baseline, repeat-execution selection, and Propose–Probe–Compose candidates comparable under one immutable experiment identity.
- Ensure candidate scope and evaluation isolation are validated before probes, and keep final scoring separate from selection.
- Produce independently inspectable quality, reliability, contract, latency, cost, and evidence-availability outcomes.

**Non-Goals:**
- A general harness optimizer, hosted experiment platform, candidate registry, production executor, scheduler, or long-lived raw trajectory store.
- Automatic candidate publication, approval, repository mutation, or durable learning/experience writeback.
- Replacing runtime budget, mailbox, checkpoint, event-history, retry, or recovery owners.

## Decisions

### Reuse the evaluation contract and existing replay owner

Represent the protocol as additive, versioned evaluation inputs and outputs in `runtime/evalcontract`; fixtures and deterministic replay stay with `tool/diagnosticsreplay` and existing contract gates. Do not add a new capability because this is a behavioral extension of the current corpus/Badcase/experiment contract. A new owner would fragment existing normalization, drift, and evidence semantics.

### Freeze identity and candidate surface before execution

Normalize an experiment manifest before probe execution. Its digest binds corpus item/Badcase, rubric declaration, baseline and candidate asset identities/digests, model, authority, input-data reference, budget, network/egress policy, and execution mode. Candidate paths, maximum size, naming, and static validators are declared before any probe; scope failure is admission failure, not a post-hoc warning.

Use the same task and declared resource envelope for baseline, same-budget repeated execution, and Propose–Probe–Compose candidates. Any variation requires a different experiment identity. Repeat runs and cache hits remain explicit cost-ledger entries rather than being collapsed into one opaque score.

### Separate selection from final scoring

Selection receives only allowed candidate artifacts and bounded behavioral evidence. Rubric answers and scoring-only data remain unavailable to selection. Record the selection result and rationale before invoking an independent final scorer. If scoring cannot run or evidence is insufficient, preserve `unavailable` or `insufficient-evidence`; never coerce missing evidence to zero. Baseline is always selectable, including when no candidate is valid or supported by evidence.

### Treat isolation and provenance as admission evidence

Require explicit egress policy, scoring-data visibility, fixture/data references, authority, write scope, budget, and asset digests in the normalized manifest. If the selected evaluator cannot demonstrate a required boundary, fail closed and produce no valid comparison. Do not infer isolation from filenames, prompt wording, or the absence of observed network access.

### Keep outputs review-only

The normalized result may contain evidence references, outcome axes, cost ledger, selection, and a recommendation for human review. It must not write candidate assets or auto-populate a persistent experience library. A repeatable issue owned by another subsystem is routed as a follow-up OpenSpec with evidence, not implemented by this experiment.

### Replay evaluates recorded facts only

Fixture replay validates schema, digest bindings, isolation declarations, outcome axes, cost accounting, selection/scoring ordering, and idempotency. It does not run candidate code, call models/tools/network, regenerate answers, retry execution, or restore a production continuation. Experiment resumption is only resuming evaluation bookkeeping and is not a claim about native source checkpoint support.

### Preserve admission and attempt evidence in reduction

The reducer requires the admitted experiment digest and retained network/scoring-isolation evidence references. An external evaluator owns actual isolation enforcement; the reducer does not open references or create a sandbox. Probe quality is nullable so that a measured zero remains distinct from missing evidence. A committed selection has a positive sequence and final scoring follows that sequence; conservative baseline selection remains valid even when another candidate has higher probe quality.

Run and cost evidence use `(candidate_id, attempt, retry)` keys; omitted run attempt means attempt 1. Duplicate keys fail closed. Retry totals count retry records rather than summing retry ordinals. Missing cost coverage preserves partial totals while marking the cost axis insufficient, and results retain the original run/ledger records and isolation evidence references for review.

Outcome-axis sums reject integer overflow/underflow rather than emitting wrapped values. Run evidence and cost-ledger entries sharing an attempt key must agree on status; ledger-only attempts remain retained accounting without inferred probe evidence.

## Risks / Trade-offs

- [Isolation is declared but not enforced by an evaluator] → Require enforcement evidence from the selected offline runner and reject unverifiable isolation; do not treat manifest text as proof.
- [Candidate variants are not actually comparable] → Bind all comparison-critical inputs and policies into the experiment digest and classify same-identity differences as drift.
- [Scoring availability biases selection] → Keep scoring unavailable/insufficient as a first-class state and do not turn it into a numeric penalty.
- [Cost accounting misses retries or caching] → Require per-attempt accounting and contract fixtures for repeated execution, retries, cache hits, model calls, and incomplete runs.
- [Review-only result becomes an implicit release path] → Keep outputs reference-only and gate any implementation/publication through a separate reviewed change.

## Migration Plan

Additive contract version and fixture namespace; preserve legacy corpus and experiment payload behavior through nullable/additive normalization defaults. Rollback removes the new candidate-comparison fixture and gate path without changing existing corpus, Badcase, experiment aggregation, continuity, or attribution behavior. No data migration or runtime deployment is required.

## Open Questions

None. Evaluator-specific mechanisms may vary, but they cannot weaken the declared isolation, fixed identity, evidence, or review-only requirements.

## Example Impact Assessment

无需示例变更（附理由）：离线评测契约不改变 `examples/agent-modes` 的配置、runtime path、expected markers 或公开运行时行为。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/mainline-contract-test-index.md`、`docs/evaluation-contract.md`
- 维护 owner：evaluation contract maintainers、documentation and contract-test index maintainers
- 验证证据：`openspec validate evaluate-bounded-harness-candidate-comparisons --strict`、`pwsh -File scripts/check-openspec-documentation-impact.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 无需文档变更（附理由） | — | evaluation contract maintainers | 不增加 runtime owner、执行循环或模块依赖 |
| components | 修改文档 | `docs/development-roadmap.md` | roadmap owner | 研究结论与 active proposal 范围已同步 |
| configuration | 修改文档 | `docs/evaluation-contract.md` | evaluation contract maintainers | 记录离线实验配置绑定；不增加生产配置键、默认值或热更新语义 |
| contract/API | 修改文档 | `docs/mainline-contract-test-index.md` | contract-test index maintainers | 增加评测契约与 replay/gate 验证映射 |
| best practices | 修改文档 | `docs/evaluation-contract.md` | evaluation contract maintainers | 明确评测隔离、证据状态与 review-only 边界 |
| diagnostics | 修改文档 | `docs/evaluation-contract.md` | evaluation contract maintainers | 记录 replay-only 证据与 review-only 输出；不改变 RuntimeRecorder 写入 |
| examples | 修改文档 | `docs/evaluation-contract.md` | examples maintainers | 明确无示例变更及不应推断运行时恢复的边界 |
| CLI/integration | 无需文档变更（附理由） | — | integration maintainers | 不新增 CLI 或生产集成入口 |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap owner | Harness² 校准和 proposal 路由已记录 |
| release status | 修改文档 | `README.md` | release documentation maintainers | 同步 active milestone snapshot 与 OpenSpec 状态 |
