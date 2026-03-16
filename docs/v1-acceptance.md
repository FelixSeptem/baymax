# V1 Acceptance And Known Limitations

## Acceptance Checklist

- Multi-turn tool-call loop runs in-process with bounded iteration and timeout policy.
- Local tool dispatch supports schema validation, concurrency controls, and fail-fast/continue policy.
- MCP stdio and HTTP adapters expose normalized `ToolResult` and aligned timeout/retry semantics.
- Runtime emits versioned events with `run_id`, `iteration`, `call_id`, `trace_id`, and `span_id` correlation fields.
- OTel spans are emitted for run/model/tool/mcp/skill paths and can be joined with event/log correlation IDs.
- Streaming path preserves causal event ordering and does not drop model deltas in integration tests.
- OpenAI streaming path uses official SDK native events with fail-fast termination and complete-tool-call-only emission.
- Model layer supports minimal multi-provider non-streaming adapters (OpenAI/Anthropic/Gemini) through the same runner contract.
- Model streaming path supports OpenAI/Anthropic/Gemini with aligned external event semantics and fail-fast behavior.
- Provider capability discovery uses official SDK model metadata APIs when available and returns controlled `unknown` when capability cannot be inferred.
- Runner performs model-step capability preflight and deterministic provider fallback by configured order; exhausted candidates fail fast with normalized model error.
- Runtime config supports YAML + env + default precedence (`env > file > default`) with startup fail-fast validation.
- Runtime diagnostics expose library APIs for recent run/MCP summaries and sanitized effective config snapshots.
- Runtime diagnostics use single-writer event ingestion with idempotent run/skill dedup semantics.
- Context Assembler CA1 runs as pre-model hook on Run/Stream, enforces immutable prefix drift fail-fast, and writes append-only file journal.
- Run diagnostics include Context Assembler CA1 baseline fields: `prefix_hash`, `assemble_latency_ms`, `assemble_status`, `guard_violation`.
- Context Assembler CA2 supports staged routing (Stage1 -> conditional Stage2), configurable stage failure policy, and tail recap append semantics.
- Run diagnostics include Context Assembler CA2 fields: `assemble_stage_status`, `stage2_skip_reason`, `stage1_latency_ms`, `stage2_latency_ms`, `stage2_provider`, `recap_status`.
- Context Assembler CA3 memory pressure control is enabled with tiered zones, dual thresholds, squash/prune/spill-swap behaviors, and Run/Stream semantic consistency checks.
- Action Gate HITL H2 is enabled with default `require_confirm`, timeout-deny semantics, and Run/Stream equivalent deny/timeout behavior.
- Clarification HITL H3 is enabled with native `await_user -> resumed -> canceled_by_user` lifecycle, structured `clarification_request` payload, and Run/Stream equivalent timeout-cancel behavior.
- Action Gate H4 parameter-schema rules are enabled with operator + composite conditions (`AND/OR`), deterministic priority over keyword/tool decisions, and Run/Stream semantic equivalence.
- R3 advanced tutorial examples (`05` to `08`) are present, runnable, and aligned with README/docs pattern navigation.
- Quality gate includes repository hygiene checks (reject temp backup artifacts), and mainline contract test coverage is indexed for traceability.

## Known Limitations (V1 Non-goals)

- No distributed orchestration or cross-process execution coordination.
- No persisted checkpoint/replay for crash recovery between sessions.
- No built-in multi-tenant control-plane, RBAC, or audit pipeline.
- Skill semantic triggering uses lightweight lexical scoring, not embedding-based retrieval.
- MCP HTTP/stdio reliability profile is available, but tuning thresholds may still require environment-specific adjustment.
- Hot reload updates runtime config atomically; invalid updates are rejected and rolled back to previous snapshot.
- `mcp/stdio` pool sizes are fixed at initialization; hot reload does not dynamically resize existing pools in-place.
- `golangci-lint` policy is baseline-first and may be tightened in later iterations.
- Concurrency safety gate in CI is baseline and will be tightened with benchmark percentage thresholds in next phases.
- Tool-call argument fragments are buffered internally and not exposed externally (complete-only contract).
- Provider fallback is scoped to model-step boundary and does not support mid-stream provider switching.
- Context Assembler CA2 Stage2 supports `file/http/rag/db/elasticsearch` via unified retriever SPI + HTTP adapter; provider-specific SDK adapters are deferred.
- Context Assembler agentic routing mode is reserved as TODO hook and currently returns explicit not-ready classification.
- Action Gate H2 当前仍仅覆盖执行前确认（tool name + keyword）；参数 schema 风险规则留作后续迭代。
- Action Gate 参数规则当前为本地配置引擎（library-first）；未接入外部策略引擎（如 OPA），未提供 schema 自动推断。
