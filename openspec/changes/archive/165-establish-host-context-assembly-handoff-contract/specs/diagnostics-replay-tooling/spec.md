## ADDED Requirements

### Requirement: Replay tooling SHALL verify host context projection conformance offline

Diagnostics replay tooling MUST support deterministic, read-only `host_context_projection.v1` fixtures that contain only synthetic opaque identities, bounded references, canonical digests, capability declarations, budget/handoff correlation, expected normalized Run/Stream outcomes, and expected drift classifications. It MUST reject unsupported schema, malformed/oversized descriptor, reference loss, projection or plan correlation drift, capability expansion, unauthorized restore, body/privacy leakage, Run/Stream mismatch, and source side effect before rejection with stable fixture-local classifications.

The replay implementation MUST NOT resolve a real host source, expose fixture body content, invoke a provider/tool/network/business store/RuntimeRecorder, mutate runtime state, or restore a live continuation. Repeating a fixture MUST produce identical normalized output and classifications.

#### Scenario: Valid synthetic projection fixture is replayed deterministically
- **WHEN** replay receives a schema-valid fixture with matching bounded descriptors and expected normalized outcomes
- **THEN** it succeeds twice with identical results and without contacting a host resolver or runtime owner

#### Scenario: Privacy or authorization drift is detected offline
- **WHEN** a fixture declares leaked body content, missing current authorization, or projection-reference drift
- **THEN** replay reports the declared stable drift classification without reading a real source or mutating runtime state
