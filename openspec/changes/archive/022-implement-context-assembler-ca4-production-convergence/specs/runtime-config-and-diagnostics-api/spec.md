## ADDED Requirements

### Requirement: Diagnostics SHALL expose CA4 token-counting semantics clearly
Runtime diagnostics documentation and fields MUST clarify that CA4 token counts are used for threshold strategy control, with explicit fallback semantics and non-blocking behavior.

#### Scenario: Token counting falls back during execution
- **WHEN** provider or local tokenizer counting fails and fallback is used
- **THEN** diagnostics semantics remain consistent and execution continues without run termination caused solely by counting failure

### Requirement: Configuration docs SHALL define CA4 threshold resolution order
Runtime configuration documentation MUST describe the exact resolution order among global thresholds, stage overrides, and mixed trigger selection.

#### Scenario: Operator reads CA4 config guide
- **WHEN** operator configures global and stage thresholds
- **THEN** operator can determine effective thresholds and conflict resolution deterministically from docs
