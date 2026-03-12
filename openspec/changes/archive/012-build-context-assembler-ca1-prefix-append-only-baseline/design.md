## Context

The roadmap has split Context Assembler into CA1-CA4 milestones to control risk and preserve current runner semantics. CA1 must deliver a minimal but strict baseline: immutable prefix, append-only journal, rule-based guardrails, and minimal diagnostics. The existing runtime already supports fail-fast semantics, hot-reload config management, and diagnostics APIs; CA1 extends those mechanisms instead of introducing a parallel subsystem.

The user confirmed constraints for CA1: scope must stay narrow, default enablement is required, guard strategy is fail-fast, persistence uses local files first, and DB integration is interface-only placeholder.

## Goals / Non-Goals

**Goals:**
- Add a pre-model `context/assembler` hook into runner with stable input/output contract.
- Enforce immutable prefix for each session/version and record `prefix_hash`.
- Enforce append-only journal writes with local JSONL storage.
- Apply guardrails outside LLM (hash/schema/sanitization) with fail-fast behavior.
- Extend runtime config/diagnostics with CA1 minimal fields.
- Keep run/stream semantics backward-compatible.

**Non-Goals:**
- No Stage2 retrieval orchestration (RAG/long-term-memory) in CA1.
- No memory-pressure control (Goldilocks/squash/prune/spill-swap) in CA1.
- No production DB backend implementation; only adapter interface placeholder.
- No changes to tool-call complete-only contract or streaming event taxonomy.

## Decisions

### Decision 1: Introduce Context Assembler as pre-model hook, not separate runtime
- Choice: runner executes assembler right before each model call (Run and Stream) and receives a normalized context bundle.
- Rationale: minimizes architecture churn and preserves current state machine boundaries.
- Alternative: independent orchestration service. Rejected due to excessive scope for CA1.

### Decision 2: Immutable prefix with explicit version and hash verification
- Choice: compose prefix from stable blocks (system prompt core, tool schema index, skill index summary, key runtime policy snapshot), compute `prefix_hash`, and verify on each assemble cycle.
- Rationale: enforces P1 consistency and provides deterministic audits.
- Alternative: semantic-equality check without hash. Rejected because byte-level invariance is required.

### Decision 3: Local file append-only journal as primary storage
- Choice: write JSONL intent/commit events to file path configured by runtime config; expose storage backend interface with file implementation and db placeholder.
- Rationale: satisfies P2/P10 immediately while preserving future extensibility.
- Alternative: directly introduce DB. Rejected for CA1 scope control.

### Decision 4: Guardrail fail-fast by default and enabled by default
- Choice: `context_assembler.enabled=true` and `guard.fail_fast=true` by default.
- Rationale: user preference and consistency with existing strict runtime semantics.
- Alternative: opt-in best-effort default. Rejected for this milestone.

### Decision 5: Minimal diagnostics first
- Choice: emit only baseline fields (`prefix_hash`, `assemble_latency_ms`, `assemble_status`, `guard_violation`) in CA1.
- Rationale: keeps instrumentation lightweight while enabling quick debugging and KPI baselining.
- Alternative: full CA3/CA4 observability set now. Rejected as premature.

## Risks / Trade-offs

- [Risk] Default enablement may surface strict failures in existing flows. -> Mitigation: provide explicit config toggle, clear diagnostics, and migration notes.
- [Risk] File-based journal I/O overhead under high QPS. -> Mitigation: buffered writer + bounded flush strategy, benchmark in CA1 tests.
- [Risk] Prefix block evolution can break hash stability unexpectedly. -> Mitigation: explicit `prefix_version` and migration guidance.
- [Risk] New hook may impact stream/run parity. -> Mitigation: contract tests enforcing semantic equivalence.

## Migration Plan

1. Add CA1 config schema and defaults (`enabled=true`, file journal path, prefix version, guard fail-fast).
2. Introduce assembler interfaces and file-journal implementation.
3. Integrate pre-model hook into runner Run/Stream paths.
4. Add diagnostics mapping and minimal fields.
5. Add tests (unit, integration parity, race safety).
6. Update docs (README + roadmap + runtime-config + v1 acceptance + phased plan).

Rollback strategy:
- Set `context_assembler.enabled=false` to bypass assembler while preserving binary compatibility.

## Open Questions

- Whether `prefix_version` should auto-bump from config hash or remain manual control can be finalized in CA2.
- Whether journal rotation strategy should be part of CA1 or deferred to CA3 operations hardening.
