# V1 Acceptance And Known Limitations

## Acceptance Checklist

- Multi-turn tool-call loop runs in-process with bounded iteration and timeout policy.
- Local tool dispatch supports schema validation, concurrency controls, and fail-fast/continue policy.
- MCP stdio and HTTP adapters expose normalized `ToolResult` and aligned timeout/retry semantics.
- Runtime emits versioned events with `run_id`, `iteration`, `call_id`, `trace_id`, and `span_id` correlation fields.
- OTel spans are emitted for run/model/tool/mcp/skill paths and can be joined with event/log correlation IDs.
- Streaming path preserves causal event ordering and does not drop model deltas in integration tests.

## Known Limitations (V1 Non-goals)

- No distributed orchestration or cross-process execution coordination.
- No persisted checkpoint/replay for crash recovery between sessions.
- No built-in multi-tenant control-plane, RBAC, or audit pipeline.
- Skill semantic triggering uses lightweight lexical scoring, not embedding-based retrieval.
- OpenAI stream path is currently compatibility-first and can be upgraded to full SDK event mapping.
- MCP HTTP heartbeat/reconnect logic is client-side best effort and not yet tuned by deployment profile.
