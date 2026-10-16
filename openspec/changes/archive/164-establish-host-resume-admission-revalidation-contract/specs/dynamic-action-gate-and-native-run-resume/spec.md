## MODIFIED Requirements

### Requirement: Dynamic action resume admission SHALL be idempotent and fail closed

Resume admission MUST require matching Run, session, checkpoint version/digest, opaque action token, and a bounded idempotency key. Duplicate equivalence MUST include the full normalized decision reference and Run/Stream mode in addition to these fields. Duplicate equivalent decisions MUST be idempotent; a source-retained completed admission returns its prior result without repeated continuation or events, while an in-flight duplicate reports a stable in-progress classification rather than empty success. Missing, stale, expired, mismatched, conflicting, or terminal checkpoints MUST be rejected without mutating Runner state or invoking a tool, executor, or provider. When host resume revalidation is explicitly enabled, a new decision MUST pass the host contract and a final source checkpoint validation before being recorded as accepted. A revalidation rejection MUST NOT consume the checkpoint or idempotency key.

#### Scenario: Duplicate resume is idempotent
- **WHEN** the same resume identity and normalized decision are submitted more than once
- **THEN** runtime returns a duplicate/idempotent admission and does not execute the continuation twice

#### Scenario: Resume token does not match checkpoint
- **WHEN** a resume request carries a token or checkpoint digest belonging to another action or Run
- **THEN** runtime rejects the request with a deterministic correlation classification and preserves input_required state

#### Scenario: Terminal checkpoint cannot be resumed
- **WHEN** a resume request targets a checkpoint whose source Run is already completed, failed, or canceled and it is not an exact duplicate of its retained admission
- **THEN** runtime rejects the request without synthesizing a new terminal transition

#### Scenario: Reused key does not prove equivalent admission
- **WHEN** a key and decision kind match but another normalized identity field or mode differs
- **THEN** the request is rejected without returning an existing result as duplicate success

#### Scenario: Required host revalidation rejects a new decision
- **WHEN** an enabled host verifier rejects the current binding or authority
- **THEN** the decision is not recorded, no resolution or terminal event is emitted, and the source checkpoint remains available for a later valid request
