## 1. Public Contract and Projection

- [x] 1.1 Add bounded provider-tagged prompt-cache directive DTOs to `core/types.ModelRequest`; test JSON compatibility, exact-one variant validation, provider mismatch, enum/range limits, and absence defaults (`go test ./core/types -run 'TestModelRequestPromptCacheDirectiveRoundTripAndLegacyDefault|TestPromptCacheDirectiveValidatesProviderVariantAndBounds' -count=1`).
- [x] 1.2 Extend provider-neutral request facts and `provider_request_projection.v1` with safe cache-intent projection; test old fixtures, stable digest, unknown fields, and raw key/resource privacy (`go test ./model/conformance -run 'TestPromptCacheFactsAndProjectionOmitOpaqueValues|TestPromptCacheProjectionDriftAndHistoricalDefault' -count=1`).

## 2. Provider Adapter Mapping

- [x] 2.1 Map OpenAI directive to Responses `prompt_cache_key`; test captured Generate/Stream requests, omitted default, invalid values, and zero SDK calls on failure (`go test ./model/openai -run 'TestPromptCacheKeyMapsToRunAndStreamWithoutChangingDefault|TestInvalidPromptCacheDirectiveDoesNotInvokeOpenAI' -count=1`).
- [x] 2.2 Map Anthropic directive to the selected Messages breakpoint and supported ephemeral TTL; test system/message boundary placement, Generate/Stream shared request construction, invalid boundary/TTL, and zero SDK calls on failure (`go test ./model/anthropic -run 'TestAnthropicPromptCacheBreakpointMapsToCanonicalMessageAndTTL|TestInvalidAnthropicPromptCacheDirectiveDoesNotInvokeProvider' -count=1`).
- [x] 2.3 Map Gemini directive to `GenerateContentConfig.CachedContent`; test captured request configuration, no cache lifecycle API path, empty/mismatched reference rejection, and zero SDK calls on failure (`go test ./model/gemini -run 'TestGeminiPromptCacheDirectiveMapsOnlyExistingCachedContent|TestInvalidGeminiPromptCacheDirectiveDoesNotInvokeProvider' -count=1`).
- [x] 2.4 Verify CountTokens request construction preserves provider cache directives where supported and returns a typed unsupported error where the API cannot represent them; test each provider's declared behavior (`go test ./model/openai ./model/anthropic ./model/gemini -count=1`).

## 3. Replay, Documentation, and Gates

- [x] 3.1 Add positive, negative, boundary, historical-default, and Run/Stream cache-control projection fixtures; verify offline deterministic replay and no raw key/reference persistence.
- [x] 3.2 Add `examples/provider-prompt-cache/README.md` and update public docs in `README.md`, `docs/guides/integration.md`, `docs/development-roadmap.md`, and `docs/mainline-contract-test-index.md`; verify links, consistency, and roadmap status.
- [x] 3.3 Extend the paired `check-provider-request-projection-contract` shell/PowerShell gates to cover prompt-cache validation, SDK invocation suppression, projection parity, and privacy; verify both scripts produce the same classification.
- [x] 3.4 Run affected package tests and dedicated gate, then `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, and `openspec validate --all`.

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
