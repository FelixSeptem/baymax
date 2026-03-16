## ADDED Requirements

### Requirement: Default concurrency backpressure policy SHALL remain block after drop-low-priority scope expansion
Expanding `drop_low_priority` applicability MUST NOT change the default runtime backpressure behavior. When backpressure mode is not explicitly configured, the runtime SHALL continue using `block` semantics.

#### Scenario: Runtime starts without explicit backpressure config
- **WHEN** concurrency config omits backpressure mode
- **THEN** runtime behavior remains `block` and no low-priority dropping is applied
