## ADDED Requirements

### Requirement: Plan admission SHALL require live coordinated owner evidence

When a host plan requests multi-writer, multi-instance, or cross-machine source
history/continuation behavior, compilation and admission SHALL validate the
bound owner coordinator profile, current fence, transaction scope and evidence
reference before publish, enqueue, claim, restore, Run or Stream. The bridge
MUST delegate all source mutation to the native owner and MUST NOT acquire a
lease, substitute a local store or fall back to direct execution.

#### Scenario: Coordinated request reaches a matching owner
- **WHEN** a plan requests a profile evidenced by its bound active owner
- **THEN** the bridge preserves that request for native owner admission without
  creating a second coordinator or source owner

#### Scenario: Coordinator profile mismatches before source activation
- **WHEN** the plan requests distributed behavior but the owner is local-only,
  unavailable, stale or evidence-mismatched
- **THEN** the bridge returns typed unsupported/unavailable before any source
  side effect or fallback execution
