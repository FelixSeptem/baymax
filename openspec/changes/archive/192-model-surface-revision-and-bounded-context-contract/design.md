## Context

See `proposal.md` for motivation.  `context/hostprojection` currently admits a
bounded opaque descriptor, then calls a host resolver and returns transient
`Material` values without a material-size envelope.  `model/conformance`
already canonicalizes and replays provider-neutral request facts, but it has no
explicit carrier for host-supplied configuration, prompt, and tool-schema
revision evidence.  Existing adapter request audits, replay fixtures, and dual
platform gates provide the intended verification seams.

## Goals / Non-Goals

**Goals:**

- Fail closed before assembly when host-resolver material is structurally or
  cumulatively over the fixed public bounds.
- Add a small, opt-in, canonical model-surface carrier that represents only
  safe revision/digest/length/availability facts for configuration, prompt,
  and tool schema.
- Keep Run/Stream and replay behavior deterministic, privacy-preserving, and
  provider-neutral.
- Extend the existing request-projection and host-context gates instead of
  creating another diagnostics, configuration, or control-plane pathway.

**Non-Goals:**

- No prompt/configuration/tool-schema registry, persistence, raw-body hashing
  service, model router, authorization decision, terminal owner, or effect
  owner.
- No provider SDK in `context/*`, no environment/file/hot-update key, no
  RuntimeRecorder write, no QueryRuns change, and no automatic truncation.
- No inference from `ToolOrder`, message roles, capability strings, cache
  controls, or provider-native request shapes.

## Decisions

1. **Use fixed all-or-nothing host material limits.**
   `context/hostprojection` will validate role (64 bytes), content (64 KiB),
   and total role-plus-content bytes (256 KiB) after each host callback and
   before returning `Resolution`.  A material error clears the accumulated
   transient result.  Fixed limits preserve `env > file > default` semantics
   and prevent callers from widening the public boundary.  Alternative:
   truncate material or create a runtime config key.  Rejected because either
   silently changes host meaning or creates configuration/diagnostics scope
   outside this request.

2. **Represent model surface as explicit safe facts in both source and
   observed projections.**
   Add a three-slot value carrier, with `available`, revision, digest, and byte
   length per slot.  Source facts receive only explicit host-supplied evidence;
   native projection copies the safe evidence only when it is supplied.  The
   default zero value means unavailable, preserving historical fixtures.
   Alternative: derive tool schema from tool ordering or add raw prompt/schema
   bodies to fixtures.  Rejected because it creates false provenance or leaks
   model input.

3. **Validate, canonicalize, and compare the carrier in the existing
   `model/conformance` path.**
   The value must be validated before canonical JSON, included in expected and
   observed comparison, and contribute a dedicated stable drift reason.  This
   keeps replay, fixture regeneration, and all provider audits in one contract
   family.  Alternative: a parallel replay format.  Rejected because a second
   truth source would allow request facts and revision evidence to drift.

4. **Treat adapter integration as provider-neutral pass-through evidence.**
   Request construction keeps provider SDK mapping in each `model/<provider>`
   package.  `context` never imports model code; `model/conformance` validates
   facts only.  Run/Stream parity test coverage goes through existing fake
   provider seams and fixtures.

5. **Document the public contract before task completion.**
   Update the committed public entry points, module boundary and no-config/no-
   diagnostics statements, contract-test index, and roadmap.  Existing
   `examples/agent-modes` paths do not change, so no example baseline is
   altered.

## Risks / Trade-offs

- A valid host source can now be rejected for size even though its descriptor
  was admitted -> publish the exact bounds and stable classification; hosts can
  supply a smaller authorized source set.
- Summary evidence can be stale or dishonest -> it is explicitly host-supplied
  reference evidence and never grants authorization or selects a model.
- The additional canonical field can break committed fixtures -> default absent
  values, update fixtures explicitly, and verify historical replay.
- Provider implementations might copy a summary inconsistently -> assert
  conformance at Run/Stream boundaries and use one provider-neutral builder.
- A safe digest might still become a correlation vector -> cap every field,
  forbid raw bodies, and retain no material outside the immediate call.

## Migration Plan

The change is additive and opt-in.  Existing host projections and request
fixtures retain their former behavior: resolver material within the new fixed
bounds is unchanged and omitted model-surface fields default unavailable.
Roll back by removing the opt-in model-surface input and reverting only the
bounded validation/carrier, fixture, and gate changes.  No data migration,
configuration rollback, diagnostic migration, provider behavior switch, or
business-effect compensation is required.

## Example Impact Assessment

无需示例变更（附理由）：本设计仅扩展 host-resolver admission 和
provider-neutral request evidence。现有 `examples/agent-modes` 的配置、
runtime path、semantic anchor、expected markers 与 rollback notes 均不变；
离线 fixture/replay/gate 是契约验证，不构成新的示例模式。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`context/hostprojection/README.md`、`model/README.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`
- 维护 owner：`context/hostprojection`、`model/conformance` 和 documentation maintainers
- 验证证据：`go test ./context/hostprojection ./model/conformance ./tool/diagnosticsreplay`、dedicated shell/PowerShell gate、`pwsh -File scripts/check-openspec-documentation-impact.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`openspec validate --all`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | architecture maintainers | package boundary and gate checks |
| host context API | 修改文档 | `context/hostprojection/README.md` | context maintainers | exact-bound resolver tests |
| model request evidence | 修改文档 | `model/README.md` | model maintainers | canonical/replay/parity tests |
| configuration and diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | no new key/writer assertions |
| contract index | 修改文档 | `docs/mainline-contract-test-index.md` | quality maintainers | fixture and dedicated gate |
| public entry point | 修改文档 | `README.md` | documentation maintainers | public capability summary |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainers | status consistency check |
| components | 修改文档 | `context/hostprojection/README.md`、`model/README.md` | context/model maintainers | focused package tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | no-config assertion and docs consistency gate |
| contract/API | 修改文档 | `README.md`、`docs/runtime-module-boundaries.md` | API/architecture maintainers | OpenSpec validation and focused contract gates |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | RuntimeRecorder/QueryRuns exclusion assertion |
| examples | 修改文档 | `context/hostprojection/README.md`、`model/README.md` | context/model maintainers | Contract documentation records the behavior; existing examples/agent-modes paths and markers remain unchanged. |
| CLI/integration | 修改文档 | `docs/mainline-contract-test-index.md` | quality maintainers | dual-platform dedicated gates |
| best practices | 修改文档 | `README.md`、`model/README.md` | documentation maintainers | privacy/default behavior tests |
