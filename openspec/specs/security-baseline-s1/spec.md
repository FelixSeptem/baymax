# security-baseline-s1 Specification

## Purpose
TBD - created by archiving change harden-security-baseline-s1-govulncheck-and-redaction. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL enforce S1 dependency security scan in strict mode
The project quality gate MUST include dependency vulnerability scanning via `govulncheck` and MUST fail the gate by default when vulnerabilities are detected.

#### Scenario: Vulnerability detected during quality gate
- **WHEN** `govulncheck` reports one or more vulnerabilities in project dependencies
- **THEN** quality gate fails and returns non-zero exit status

#### Scenario: No vulnerability detected during quality gate
- **WHEN** `govulncheck` reports no vulnerabilities
- **THEN** quality gate continues to subsequent checks without security-scan failure

### Requirement: Runtime SHALL apply unified redaction pipeline across diagnostics and observability paths
Sensitive values MUST be redacted through a shared pipeline for diagnostics records, runtime event payloads, and context assembler outputs before they are persisted or emitted.

#### Scenario: Sensitive key appears in diagnostics payload
- **WHEN** payload contains keys matching configured sensitive patterns
- **THEN** persisted diagnostics mask corresponding values

#### Scenario: Sensitive key appears in context assembler recap
- **WHEN** context assembler emits tail recap with sensitive key-like fields
- **THEN** recap output is redacted before being attached to model context and diagnostics payload

### Requirement: Security redaction strategy SHALL be keyword-based and extensible
S1 redaction strategy MUST support configurable keyword matching as baseline and MUST expose extension points for future strategy evolution.

#### Scenario: Baseline keyword match
- **WHEN** field key matches baseline keyword set (`token/password/secret/api_key/apikey`)
- **THEN** redaction pipeline masks the field value

#### Scenario: Extended keyword match
- **WHEN** runtime config defines additional sensitive keywords
- **THEN** redaction pipeline applies masking for those keywords without code changes

