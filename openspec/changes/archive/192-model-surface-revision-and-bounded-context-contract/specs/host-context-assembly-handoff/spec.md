## ADDED Requirements

### Requirement: Resolved host materials SHALL have fixed all-or-nothing bounds

Before material reaches context assembly or a caller-owned correlated restore
injection, the host context projection contract SHALL validate every
resolver-returned material as a complete set.  A material role and content MUST
be nonempty, valid UTF-8, and respectively at most 64 UTF-8 bytes and 64 KiB;
the aggregate UTF-8 byte size of every role and content in one resolution MUST
not exceed 256 KiB.  The limits are fixed library-contract limits: they MUST
NOT be exposed through environment, file, hot-update, or per-request
configuration.

An invalid, over-limit, or aggregate-overflow material set MUST return a stable
bounded `host_context.*` classification, discard the entire transient result,
and MUST NOT inject a prefix, truncate a value, persist a body, update a
handoff, resume execution, write diagnostics, or commit a business effect.
The resolver remains host-owned and current-authorization checks remain
unchanged; the bounds do not turn a digest, role, or content into authorization
evidence.

#### Scenario: Exact-boundary materials are admitted as one detached resolution
- **WHEN** every resolver-returned role and content is nonempty, individually
  within the declared limits, and the aggregate equals 256 KiB or less
- **THEN** the contract returns the complete transient material set to the
  existing assembly path without changing its source-owner or budget semantics

#### Scenario: One invalid or oversized material rejects the whole resolution
- **WHEN** a resolver returns a role or content that is invalid UTF-8, a role
  longer than 64 UTF-8 bytes, or content longer than 64 KiB
- **THEN** the contract returns its stable material-bound classification and
  no material from that descriptor is injected or retained

#### Scenario: Aggregate material overflow is never truncated
- **WHEN** individually valid resolver materials exceed 256 KiB in aggregate
- **THEN** the contract rejects the descriptor's complete transient result
  without returning a partial prefix, creating a handoff correlation, or
  starting/resuming execution

#### Scenario: Material-bound failure preserves Run and Stream safety
- **WHEN** equivalent Run and Stream assembly requests encounter the same
  material-bound failure
- **THEN** both paths expose semantically equivalent bounded rejection before a
  provider, tool, terminal owner, or business effect is reached
