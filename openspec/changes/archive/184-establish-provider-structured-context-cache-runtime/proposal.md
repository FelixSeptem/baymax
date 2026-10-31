## Why

The archived request-projection and cache-usage contracts preserve structured context and report provider-observed cache usage, but `types.ModelRequest` has no request-side cache intent and the adapters do not map an explicit host choice to their native prompt-cache controls. Hosts therefore cannot deliberately select a stable OpenAI cache key, an Anthropic cache breakpoint, or an existing Gemini cached-content resource through the supported runtime path. This is a distinct runtime gap from archive 142/144/148 and admission-only evidence in 152.

## What Changes

- Add a bounded, SDK-neutral, additive cache-control intent to model requests, with provider-specific validation at the owning adapter boundary.
- Map only documented native request controls for the supported OpenAI Responses, Anthropic Messages, and Gemini GenerateContent paths, and keep Run/Stream behavior equivalent.
- Extend request projection/conformance evidence to record the presence and bounded classification of cache controls without recording raw prompt content, provider resource bodies, or credentials.
- Preserve default behavior when no cache intent is supplied. Unsupported, malformed, or mismatched intent fails before provider invocation; it is never silently ignored or converted into another provider's cache behavior.
- Do not create, update, delete, persist, or resolve provider cache resources; do not add a gateway, cache service, cross-provider cache equivalence claim, automatic cache policy, or runtime selector.

## Capabilities

### New Capabilities

- `provider-native-prompt-cache-controls`: explicit, opt-in request-side cache controls with provider-owned mapping, validation, and observable projection.

### Modified Capabilities

- `provider-request-projection-and-cache-observability`: record request-side cache intent and observed native control in the bounded projection, preserving additive/default compatibility and Run/Stream parity.

## Impact

- Likely code owners: `core/types.ModelRequest`, `model/openai`, `model/anthropic`, `model/gemini`, `model/conformance`, diagnostics replay, and contribution gates.
- Existing provider SDKs already expose relevant request fields: OpenAI Responses `prompt_cache_key`, Anthropic Messages cache-control fields, and Gemini `GenerateContentConfig.CachedContent`. Mapping must remain inside each `model/<provider>` package.
- Existing cache usage response projection remains unchanged; request intent is not evidence of a cache hit, cost reduction, or latency improvement.
- Test and documentation surfaces: provider request-shape tests, `provider_request_projection.v1` compatibility fixtures/replay, provider cache evidence gate, `docs/development-roadmap.md`, provider integration documentation, and `docs/mainline-contract-test-index.md`.

## Example Impact Assessment

修改示例
说明：本 change 增加模型请求的可选 cache-control intent，并改变 provider request projection；新增 provider adapter 示例，展示省略 intent 时的默认请求及显式 intent 的 provider-native 请求形状，不包含真实 cache key、资源名或 prompt。

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
