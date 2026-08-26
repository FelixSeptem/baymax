## ADDED Requirements

### Requirement: Realtime events SHALL map to canonical protocol lifecycle and event references
The existing realtime envelope and resume cursor MUST map deterministically to the Agent Runtime Protocol Event and Run lifecycle references. Realtime MUST remain the authority for event taxonomy, sequence, idempotency, interrupt freeze, and resume cursor validation.

#### Scenario: Valid resume maps without transport rewrite
- **WHEN** a valid realtime resume event is accepted
- **THEN** its canonical protocol mapping exposes the existing run/session/event correlation and resume causal relationship without adding a hosted transport dependency
