## MODIFIED Requirements

### Requirement: Runtime SHALL support task-level delayed dispatch via not_before
The runtime MUST support task-level delayed dispatch through mailbox envelope timing fields.

`not_before` MUST control earliest consume/claim eligibility, and optional `expire_at` MUST define terminal expiration boundary.

#### Scenario: Delayed message is published with future not_before
- **WHEN** mailbox command envelope is published with `not_before` greater than current runtime time
- **THEN** the message remains non-consumable until runtime time reaches `not_before`

#### Scenario: Delayed message expires before becoming eligible
- **WHEN** envelope defines `expire_at` and runtime time passes `expire_at` before successful consume
- **THEN** message is handled by configured expiration policy (drop or dlq) with deterministic reason metadata
