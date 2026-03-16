## Why

CA3 semantic compaction in F2 already has rule-based quality gating and an embedding SPI hook, but quality evaluation still lacks real similarity signals from provider embeddings. We should complete the next incremental step now so quality decisions become more robust while preserving current fail-fast and best-effort safety semantics.

## What Changes

- Implement provider-specific embedding adapters for OpenAI, Gemini, and Anthropic in CA3 semantic compaction quality scoring (behind config switch).
- Extend CA3 quality scorer to support hybrid scoring: existing rule-based score plus optional embedding similarity component.
- Use cosine similarity as the initial and only E3 similarity metric.
- Keep deterministic fallback semantics: adapter unavailable/timeout/error falls back to rule-only path under `best_effort`, and fails assembly under `fail_fast`.
- Add runtime config for embedding scorer enablement, provider/model selection, independent embedding credentials, timeout, and weighting validation.
- Extend diagnostics with embedding component visibility (adapter path used, similarity contribution, fallback reasons).
- Add contract tests and benchmark coverage for Run/Stream semantic equivalence and regression safety.

## Capabilities

### New Capabilities
- `ca3-semantic-embedding-adapter`: Provider-backed embedding similarity component for CA3 semantic quality scoring.

### Modified Capabilities
- `context-assembler-memory-pressure-control`: Extend CA3 semantic compaction requirements from SPI hook placeholder to adapter-backed scoring behavior and fallback semantics.
- `runtime-config-and-diagnostics-api`: Extend CA3 quality-related config and diagnostics requirements to include embedding scorer controls and fields.

## Impact

- Affected code:
  - `context/assembler` (embedding adapter implementation, hybrid quality scorer, failure handling)
  - `model/*` or adjacent adapter module (provider-specific embedding client integration)
  - `runtime/config` (new scorer config surface + validation)
  - `runtime/diagnostics`, `observability/event`, `core/runner` (diagnostic propagation)
  - `integration` / contract tests / benchmarks
- Affected docs:
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/context-assembler-phased-plan.md`
  - `docs/development-roadmap.md`
  - `docs/v1-acceptance.md`
- Compatibility:
  - Default behavior remains rule-only scoring unless embedding scorer is explicitly enabled.
  - Default hybrid settings: cosine similarity, `rule_weight=0.7`, `embedding_weight=0.3`, and shared existing quality threshold.
  - Existing APIs remain backward compatible; new fields are additive.
