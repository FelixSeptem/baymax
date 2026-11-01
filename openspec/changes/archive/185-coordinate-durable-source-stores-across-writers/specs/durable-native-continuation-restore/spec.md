## MODIFIED Requirements

### Requirement: Resume SHALL use an atomic source claim and preserve one logical Run

The source SHALL provide an atomic resume claim operation that validates digest, schema, ownership, expiry, pause state, causal predecessor, and full idempotency identity. Equivalent duplicate requests SHALL return the retained claim/result projection; conflicting or already-consumed requests SHALL fail closed. Successful restore MUST preserve the original logical parent/run/session and MUST NOT create a replacement Run or rerun completed tool work.

When a store advertises multi-writer, multi-instance, or cross-machine restore,
the atomic claim MUST be published through its source-owned fenced coordinator
and current CAS revision. A local sequential-reopen store MUST reject that
request rather than interpret it as a local claim.

#### Scenario: Exact duplicate resume is idempotent
- **WHEN** the same checkpoint and normalized resume identity are claimed twice
- **THEN** both calls return the same admitted/result projection and only one continuation can execute

#### Scenario: Conflicting claim is rejected
- **WHEN** a second request changes the checkpoint digest, mode, pause reason, causal predecessor, or resume identity
- **THEN** the request receives a typed conflict/in-progress result without consuming or replacing the source checkpoint

#### Scenario: Old checkpoint cannot consume a successor
- **WHEN** a resumed Run pauses at a new checkpoint and an old resume request is repeated
- **THEN** it returns only its retained prior result/projection or a typed stale/expired result and never consumes the successor checkpoint

#### Scenario: Coordinated writer fence becomes stale
- **WHEN** a coordinator-backed writer loses or is superseded by its fence before resume claim publication
- **THEN** the owner returns a typed stale/unavailable result before continuation activation and does not create a replacement Run
