## ADDED Requirements

### Requirement: Manifest SHALL declare source commit projection recovery separately

For a source with durable history, the capability manifest MUST separately
declare whether source commit/projection recovery is supported, unavailable, or
unsupported. A supported declaration MUST reference the source-owned commit
contract, its recovery evidence, and deployment scope. History replay, a file
reopen, or a continuation DTO MUST NOT imply this support.

#### Scenario: Durable history does not imply transactional recovery
- **WHEN** a source entry has a retained journal but no declared commit and
  projection recovery capability
- **THEN** the manifest reports the recovery dimension unavailable or unsupported
- **AND** the planner cannot admit a transaction-dependent projection path

#### Scenario: Supported recovery shows its boundary
- **WHEN** a source owner provides committed source records and recovery reads
- **THEN** the manifest includes its evidence reference and same-host or
  distributed deployment scope
- **AND** it does not claim an external business database is atomically committed
