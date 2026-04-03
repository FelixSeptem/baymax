## ADDED Requirements

### Requirement: Memory Lifecycle Alignment with Unified Snapshot
Unified snapshot import/export MUST align with A59 memory lifecycle semantics and MUST NOT redefine memory source-of-truth behavior.

#### Scenario: Lifecycle policy preserved on export/import
- **WHEN** memory records with A59 lifecycle metadata are exported and re-imported through unified snapshot
- **THEN** lifecycle actions (retention/ttl/forget) MUST remain semantically equivalent

#### Scenario: No shadow source introduction
- **WHEN** unified snapshot restore is performed for memory segment
- **THEN** restore path MUST consume existing memory SPI/filesystem semantics without introducing an alternate authoritative store

### Requirement: Memory Restore Idempotency and Quality Stability
Repeated memory segment restore from identical snapshot payload MUST remain idempotent and MUST preserve retrieval quality baseline behavior.

#### Scenario: Memory restore idempotent
- **WHEN** identical memory segment snapshot is imported multiple times
- **THEN** record counts and lifecycle counters MUST NOT inflate

#### Scenario: Retrieval behavior remains within baseline
- **WHEN** query/rerank is executed before and after compatible restore
- **THEN** retrieval output ordering and quality thresholds MUST remain within deterministic contract tolerance
