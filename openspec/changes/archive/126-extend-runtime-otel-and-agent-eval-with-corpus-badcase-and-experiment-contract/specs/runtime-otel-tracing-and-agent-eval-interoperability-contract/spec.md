## ADDED Requirements

### Requirement: Tracing and eval SHALL expose additive corpus and experiment correlation
Tracing and evaluation outputs MUST optionally expose bounded corpus version/item, Badcase, experiment, rubric, comparison, and feedback correlation fields through the existing RuntimeRecorder single-writer path. Missing fields MUST retain documented nullable defaults and existing v1 payloads MUST remain readable.

#### Scenario: Correlated eval result is recorded
- **WHEN** an evaluation result includes valid corpus, experiment, rubric, and Badcase references
- **THEN** RuntimeRecorder records the additive fields without changing existing span topology or metric names

#### Scenario: Legacy payload omits new fields
- **WHEN** a historical tracing/eval payload has none of the new correlation fields
- **THEN** parsing succeeds and all new fields resolve to nullable/default values

#### Scenario: Correlation exceeds cardinality bounds
- **WHEN** a payload contains oversized or high-cardinality corpus/feedback values
- **THEN** the recorder applies existing truncation/redaction policy and reports the bounded outcome without writing unbounded OTel attributes
