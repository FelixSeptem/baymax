## 1. Baseline and Contract

- [x] 1.1 Audit the locked OpenAI, Anthropic, and Gemini SDK usage fields and record provider-specific source mappings; verify the audit fixture/documentation contains no live provider dependency.
- [x] 1.2 Extend the provider-neutral cache usage projection with bounded availability, read, write/create, total, and source metadata using additive + nullable + default semantics; verify existing historical JSON still decodes as unavailable.
- [x] 1.3 Add positive, negative, overflow, missing-source, unknown-field, and inconsistent-total conformance tests; verify stable cache drift classifications and no fabricated values.

## 2. Provider Adapter Mapping

- [x] 2.1 Map OpenAI Responses cached input usage at the adapter boundary for Run and Stream; verify non-negative values and unavailable fallback with provider unit tests.
- [x] 2.2 Map Anthropic cache read and cache creation input usage at the adapter boundary for Run and Stream; verify read/write classification and no TTL/region leakage.
- [x] 2.3 Map Gemini cached content token usage at the adapter boundary for Run and Stream; verify the provider-specific field is translated only when explicitly present.
- [x] 2.4 Define and test authoritative Stream usage selection so interim and terminal usage records are not double-counted; verify Run/Stream normalized parity.
- [x] 2.5 Verify CountTokens paths preserve unavailable/default cache semantics when the provider API exposes no cache accounting.

## 3. Fixture, Replay, and Gates

- [x] 3.1 Update `provider_request_projection.v1` fixtures with bounded cache-available cases for each supported source and historical cache-absent cases; verify fixture size and redaction bounds.
- [x] 3.2 Extend `tool/diagnosticsreplay` to validate cache usage normalization, canonical digest, unknown-field compatibility, overflow, and parity; verify replay idempotency with repeated runs.
- [x] 3.3 Add or extend shell and PowerShell provider projection gates for taxonomy parity, provider SDK ownership, fixture bounds, historical compatibility, and Run/Stream cache parity.

## 4. Documentation and Integration

- [x] 4.1 Update `model/README.md`, `docs/mainline-contract-test-index.md`, `docs/runtime-config-diagnostics.md` if required, and `docs/development-roadmap.md` to describe the additive cache usage contract and its non-goals.
- [x] 4.2 Add Example Impact Assessment evidence to proposal, design, and tasks and verify it is one of the allowed values.
- [x] 4.3 Run focused model/conformance/replay tests and both provider projection gates; verify no runtime config, RuntimeRecorder schema, or example-mode changes were introduced.
- [x] 4.4 Run `openspec validate --all`, roadmap/docs consistency checks, `git diff --check`, and the repository quality gates required for code changes; record any environment-blocked command explicitly.

## Example Impact Assessment

**无需示例变更（附理由）**。任务只覆盖 provider usage 映射、离线 fixture/replay、contract gate 与文档，不改变 `examples/agent-modes` 的配置、runtime path 或 expected markers。
