## Why

Baymax already owns an OpenAI Responses adapter with structured role, tool-result, stream and cache-usage projection, but it has no explicit contract describing which OpenAI-compatible endpoints may reuse that projection safely. Because OpenAI-compatible model services are widespread, a bounded profile and offline conformance contract can expand host reach without inventing provider discovery, a shared wire protocol, or unverified compatibility claims.

## What Changes

- Add a versioned OpenAI-compatible endpoint profile that requires explicit host-supplied endpoint identity, model identity, API shape, and capability declarations.
- Define offline conformance fixtures for Responses-shaped structured context, role/part ordering, native tool-result correlation, stream event boundaries, terminal cache usage, and Run/Stream parity.
- Reuse the existing `model/openai` native projection owner and permit only explicitly declared compatible endpoints; unsupported CountTokens, cache, tool, or stream features remain explicit and unavailable.
- Add positive, negative, boundary, privacy, replay-idempotency, and shell/PowerShell gate coverage without live network calls or automatic endpoint probing.
- Document the profile, compatibility limits, rollback path, and example impact. No `examples/agent-modes` behavior changes are required.

## Capabilities

### New Capabilities

- `openai-compatible-endpoint-profile-conformance`: Explicit OpenAI-compatible endpoint profile validation and offline protocol conformance for structured context, tools, streaming, cache usage, and parity.

### Modified Capabilities

无。现有 `provider-request-projection-and-cache-observability` 和 OpenAI adapter behavior remain unchanged; this change adds an opt-in conformance/profile contract around the existing owner.

## Impact

- Affected owners: `model/openai`, `model/conformance`, `tool/diagnosticsreplay`, `tool/contributioncheck`, and provider documentation.
- A profile is host-supplied and bounded; it does not add automatic discovery, credential storage, global routing, hosted state, or provider-neutral SDK types in `context/*`.
- Historical `provider_request_projection.v1` fixtures remain valid. Any new profile fields are additive, nullable, and default-safe where persisted.
- Rollback removes the profile/conformance gate and docs while leaving the existing official OpenAI adapter and historical fixtures unchanged.

## Example Impact Assessment

无需示例变更（附理由）：本 change 只增加显式 endpoint profile、离线 conformance fixture/replay 和 gate，不改变 `examples/agent-modes` 的配置、runtime path、expected markers 或 rollback notes。若后续示例需要展示兼容 endpoint，必须先完成 `MATRIX.md` 与对应 README 的文档基线。
