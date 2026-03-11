# V1 Acceptance And Known Limitations

## Acceptance Checklist

- Multi-turn tool-call loop runs in-process with bounded iteration and timeout policy.
- Local tool dispatch supports schema validation, concurrency controls, and fail-fast/continue policy.
- MCP stdio and HTTP adapters expose normalized `ToolResult` and aligned timeout/retry semantics.
- Runtime emits versioned events with `run_id`, `iteration`, `call_id`, `trace_id`, and `span_id` correlation fields.
- OTel spans are emitted for run/model/tool/mcp/skill paths and can be joined with event/log correlation IDs.
- Streaming path preserves causal event ordering and does not drop model deltas in integration tests.
- OpenAI streaming path uses official SDK native events with fail-fast termination and complete-tool-call-only emission.

## Known Limitations (V1 Non-goals)

- No distributed orchestration or cross-process execution coordination.
- No persisted checkpoint/replay for crash recovery between sessions.
- No built-in multi-tenant control-plane, RBAC, or audit pipeline.
- Skill semantic triggering uses lightweight lexical scoring, not embedding-based retrieval.
- MCP HTTP/stdio reliability profile is available, but tuning thresholds may still require environment-specific adjustment.
- `golangci-lint` policy is baseline-first and may be tightened in later iterations.
- Concurrency safety gate in CI is baseline and will be tightened with benchmark percentage thresholds in next phases.
