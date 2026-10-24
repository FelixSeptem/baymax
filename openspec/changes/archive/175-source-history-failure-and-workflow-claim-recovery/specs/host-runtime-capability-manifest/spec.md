## ADDED Requirements

### Requirement: Workflow continuation capability SHALL require an active owner fence

The workflow continuation capability manifest MUST advertise same-host single-writer sequential reopen only while the workflow source store owns its process-wide writer fence and can validate incomplete-claim recovery semantics. A failed or absent fence MUST report typed unavailable/unsupported before workflow resume admission.

#### Scenario: Capability requested without an acquired fence
- **WHEN** a workflow Engine requests continuation capabilities but its FileStore has no valid writer ownership
- **THEN** capability lookup returns typed unavailable/unsupported
- **AND** the engine does not claim cross-process restore availability

#### Scenario: Capability requested by the fenced owner
- **WHEN** a workflow Engine is bound to a valid fenced FileStore with supported continuation contracts
- **THEN** it reports only same-host single-writer sequential-reopen support
- **AND** it does not advertise multi-writer or cross-machine recovery
