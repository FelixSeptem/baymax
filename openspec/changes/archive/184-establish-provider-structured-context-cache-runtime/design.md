## Context

See `proposal.md` and the two spec deltas. The archived projection contract already preserves native message roles, ordering, tool-result correlation, and cache-usage output. The missing request-side path is explicit cache intent: `ModelRequest` currently has no such field and the three native request builders do not attach cache controls. SDK controls are not semantically interchangeable.

## Goals / Non-Goals

**Goals:**

- Add a typed, bounded, additive directive union in `core/types` with exactly one provider-specific variant matching the selected adapter.
- Map OpenAI's opaque Responses `PromptCacheKey`, Anthropic's Messages cache breakpoint and TTL at a caller-selected structured-message boundary, and Gemini's `GenerateContentConfig.CachedContent` reference.
- Reject invalid, conflicting, unsupported, or mismatched directives before SDK invocation.
- Extend provider-neutral request facts/projections, replay, tests, and documentation without storing raw keys or references.

**Non-Goals:**

- Create or manage cached-content resources, infer cache identity, generate cache keys, or promise cache hits/cost savings.
- Add provider SDK dependencies outside existing `model/<provider>` packages.
- Change response cache-usage normalization, retries, model routing, context assembly policy, or terminal behavior.

## Decisions

1. **Use a provider-tagged typed union.** `ModelRequest` gets an optional `PromptCacheDirective{Provider, OpenAI, Anthropic, Gemini}` with exactly one provider-specific subrecord. OpenAI carries an opaque key (1-64 UTF-8 bytes); Anthropic carries a zero-based interpreted-message boundary and `5m|1h` TTL; Gemini carries a caller-owned resource reference (1-512 UTF-8 bytes). Provider discriminator and variant must match. This makes cross-provider semantic differences explicit while keeping SDK request structs private to their adapters. A single generic `enabled` boolean was rejected because it cannot represent the three native controls or safely validate unsupported behavior.
2. **Keep cache references caller-owned.** OpenAI receives an opaque bounded key; Anthropic receives a message boundary index and TTL enum; Gemini receives a bounded existing cached-content reference. Runtime does not call cache lifecycle APIs. This avoids silently adding persistence, network calls, and resource ownership.
3. **Make absence preserve today's behavior.** No directive means no new SDK fields. Every provider validates a present directive before invocation, including `Generate`, `Stream`, and `CountTokens` request construction where applicable. Anthropic's boundary indexes the exact interpreted canonical sequence (including the synthetic final input when present); the adapter attaches cache control to the corresponding native text block without reordering content.
4. **Project structure, not opaque values.** Request projection records provider/kind and safe structural metadata. Raw key/resource reference never enters conformance, diagnostics, or fixtures. Any reference digest is only a parity aid, not an identity or cache key.
5. **Version fixture semantics additively.** Existing `provider_request_projection.v1` fixtures remain valid when the field is omitted; new observations use optional fields and stable drift classification. Existing response cache usage remains independent.

## Risks / Trade-offs

- [Provider SDK field behavior changes across releases] -> Pin request-shape tests to the currently supported SDK versions and fail closed when the required field is unavailable.
- [A provider discriminator or union variant may be malformed] -> Validate exact-one variant, adapter match, bounds, enums, and Anthropic boundary range before any SDK call.
- [Opaque keys/resource names may leak through errors] -> Never include values in error messages, logs, fixture data, or projection; tests use sentinel strings and assert their absence.
- [Cache controls do not guarantee a hit] -> Keep output accounting independent and explicitly report unavailable usage when the provider does not report it.
- [Run/Stream implementations may diverge] -> Share provider-owned request builders and test captured native params on both paths.

## Migration Plan

Additive rollout: existing serialized requests omit the optional directive and retain current behavior. Hosts can opt in by sending a provider-matching directive after upgrading. Rollback removes the directive field handling and projection additions; no stored resources or data migration exist. Documentation changes are included in this change: `README.md`, `docs/guides/integration.md`, `docs/development-roadmap.md`, and `docs/mainline-contract-test-index.md`; owners and verification commands are declared in `proposal.md`.

## Example Impact Assessment
修改示例
说明：新增 provider adapter 示例，展示省略 intent 时的默认请求及显式 intent 的 provider-native 请求形状，不包含真实 cache key、资源名或 prompt。

## Public Semantic Documentation Commitment
- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/guides/integration.md`、`docs/development-roadmap.md`、`docs/mainline-contract-test-index.md`、`examples/provider-prompt-cache/README.md`
- 维护 owner：runtime/provider maintainers
- 验证证据：provider adapter tests、projection replay tests、dedicated provider request projection gate、`pwsh -File scripts/check-docs-consistency.ps1`、`pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1`

## Documentation Impact Assessment
| 表面 | 结果 | 文档路径 | Owner | 验证证据 |
| --- | --- | --- | --- | --- |
| architecture | 无需文档变更（附理由） | — | — | 不改变 runtime module boundaries 或 provider SDK ownership。 |
| components | 修改文档 | `model/README.md` | provider adapter maintainers | adapter package tests and docs consistency gate |
| configuration | 修改文档 | `docs/guides/integration.md` | provider adapter maintainers | provider directive validation tests and docs consistency gate |
| contract/API | 修改文档 | `README.md`, `docs/guides/integration.md` | runtime/provider maintainers | model request and provider adapter tests |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | test infrastructure maintainers | projection replay tests and dedicated gate |
| examples | 新增文档 | `examples/provider-prompt-cache/README.md` | provider adapter maintainers | example review and docs consistency gate |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | integration maintainers | integration docs consistency gate |
| best practices | 无需文档变更（附理由） | — | — | No new practice beyond explicit opt-in and secret omission documented in integration guidance. |
| roadmap | 修改文档 | `docs/development-roadmap.md` | runtime maintainers | `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1` |
