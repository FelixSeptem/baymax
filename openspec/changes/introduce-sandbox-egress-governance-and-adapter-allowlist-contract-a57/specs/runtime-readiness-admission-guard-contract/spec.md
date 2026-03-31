## ADDED Requirements

### Requirement: Admission guard SHALL deny execution on blocking egress findings
Runtime readiness-admission guard MUST deny managed execution when readiness reports blocking sandbox egress findings.

Deny path MUST remain side-effect free.

#### Scenario: Admission receives blocking egress finding
- **WHEN** readiness output contains blocking `sandbox.egress.policy_invalid`
- **THEN** admission decision is `deny` and runtime performs no scheduler or mailbox side effects

#### Scenario: Equivalent Run and Stream egress deny
- **WHEN** equivalent managed Run and Stream requests consume the same blocking egress finding
- **THEN** both paths return semantically equivalent deny classification

### Requirement: Admission guard SHALL deny activation on blocking allowlist findings
Admission MUST deny execution when readiness reports blocking adapter allowlist findings for required runtime adapters.

#### Scenario: Required adapter missing allowlist entry
- **WHEN** readiness output includes blocking `adapter.allowlist.missing_entry`
- **THEN** admission decision is `deny` with deterministic allowlist reason taxonomy

#### Scenario: Signature invalid under enforce mode
- **WHEN** readiness output includes blocking `adapter.allowlist.signature_invalid`
- **THEN** admission decision is `deny` and managed execution does not start

### Requirement: Admission explainability SHALL preserve egress and allowlist primary reason fields
Admission outputs MUST preserve canonical arbitration explainability fields when deny is driven by egress or allowlist findings.

#### Scenario: Egress-driven deny includes explainability payload
- **WHEN** admission denies due to egress blocking finding
- **THEN** output includes canonical primary domain code source and bounded secondary reasons

#### Scenario: Allowlist-driven deny includes explainability payload
- **WHEN** admission denies due to allowlist blocking finding
- **THEN** output includes canonical allowlist primary reason fields without remapping drift
