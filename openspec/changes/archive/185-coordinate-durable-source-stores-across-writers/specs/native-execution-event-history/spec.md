## ADDED Requirements

### Requirement: Coordinated source history SHALL publish append and cursor facts atomically

An owner that declares coordinated multi-writer source history SHALL bind event
append, sequence/floor evolution and durable snapshot replacement to one fenced
source-owner CAS revision. It MUST expose only committed source facts, preserve
source-local order, and fail closed on stale writer, CAS conflict or coordinator
unavailability. It MUST NOT claim global ordering across independent sources.

#### Scenario: Retention floor evolves with a committed append
- **WHEN** a coordinated append crosses the journal retention bound
- **THEN** the new event, next sequence and retention floor become visible in
  one committed source revision or none become visible

#### Scenario: Cursor reader cannot mutate coordination state
- **WHEN** a consumer calls ReadAfter with a valid retained cursor
- **THEN** the read observes committed source facts only and makes no lease, CAS
  or execution call
