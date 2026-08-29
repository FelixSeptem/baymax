## ADDED Requirements

### Requirement: Observability export SHALL keep evaluation correlation optional and redacted
Observability export and diagnostics bundles MUST include evaluation correlation only when present and permitted by existing redaction/cardinality policy. Export MUST use nullable additive fields and MUST NOT require corpus or feedback content to be embedded in OTel spans or bundles.

#### Scenario: Evaluation correlation is exported
- **WHEN** a run has bounded corpus and experiment references
- **THEN** the export contains those references and digests as additive fields while preserving existing schema compatibility

#### Scenario: Sensitive evaluation input is present
- **WHEN** an evaluation item contains sensitive input or reviewer text
- **THEN** export omits or redacts the content and retains only allowed bounded references/digests

#### Scenario: No evaluation metadata is present
- **WHEN** a run does not participate in corpus evaluation
- **THEN** export remains byte-for-byte compatible with the existing non-eval shape apart from permitted nullable defaults
