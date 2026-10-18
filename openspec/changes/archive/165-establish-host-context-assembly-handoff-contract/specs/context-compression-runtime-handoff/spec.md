## ADDED Requirements

### Requirement: Handoff SHALL preserve bounded host context projection correlation

When an explicitly admitted `host_context_projection.v1` participates in context assembly, `handoff.v1` SHALL preserve an additive, nullable, bounded correlation for its schema, projection identity/version/digest, capability digest, optional plan correlation, and opaque source-reference evidence. The handoff MUST NOT copy resolved source bodies, private-memory/knowledge content, credentials, raw business authorization, or effect payloads.

Restore SHALL first retain all existing handoff schema/cut/reference validation and then require matching host projection correlation plus current host source authorization before injection. A missing/changed source reference, projection or capability digest drift, authorization failure, or capability expansion MUST fail closed before context mutation and MUST NOT create a new terminal outcome, business commit, or resolver-independent restore success.

#### Scenario: Host projection correlation is additive and bounded
- **WHEN** a legal handoff is generated for an admitted host projection
- **THEN** existing handoff consumers remain compatible while the record contains only the bounded projection correlation needed for restore revalidation

#### Scenario: Unauthorized correlated restore is rejected
- **WHEN** an otherwise valid handoff carries a projection that no longer has current source authorization
- **THEN** restore rejects before injection and preserves the existing source/terminal ownership semantics
