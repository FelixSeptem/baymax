## ADDED Requirements

### Requirement: CA2 Stage2 external retriever observability SHALL preserve existing stage policy behavior
CA2 Stage2 external retriever observability enhancements MUST NOT change existing stage policy semantics (`fail_fast` and `best_effort`).

Threshold-hit evaluation and provider trend aggregation MUST be observational only in this milestone.

#### Scenario: fail_fast policy with threshold hit
- **WHEN** Stage2 runs under `fail_fast` policy and threshold-hit signal is produced
- **THEN** Stage2 execution behavior remains governed by existing fail_fast semantics without additional automatic actions

#### Scenario: best_effort policy with threshold hit
- **WHEN** Stage2 runs under `best_effort` policy and threshold-hit signal is produced
- **THEN** Stage2 execution behavior remains governed by existing best_effort semantics without additional automatic actions

### Requirement: CA2 Stage2 error-layer trend semantics SHALL allow enum extension
CA2 Stage2 diagnostics trend aggregation MUST support baseline error layers (`transport`, `protocol`, `semantic`) and MUST allow forward-compatible enum extension.

#### Scenario: Baseline error layers are aggregated
- **WHEN** Stage2 retrieval failures occur across baseline layers
- **THEN** trend diagnostics aggregate and expose layer distribution without schema conflict

#### Scenario: Extended error layer value is emitted
- **WHEN** an implementation emits a new layer enum value in a backward-compatible extension
- **THEN** diagnostics trend aggregation accepts and preserves the value without failing parsing
