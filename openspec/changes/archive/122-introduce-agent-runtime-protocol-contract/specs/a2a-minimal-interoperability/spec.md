## ADDED Requirements

### Requirement: A2A Task lifecycle SHALL map to Agent Runtime Protocol references
A2A Task lifecycle, status, artifact, event, and orchestration-correlation metadata MUST map deterministically to Agent Runtime Protocol Run, Step, Event, and Artifact references. The mapping MUST not redefine A2A Task ownership, transport, or terminal semantics.

#### Scenario: Peer task terminal result maps to protocol lineage
- **WHEN** an A2A peer Task reaches a terminal result with orchestration correlation
- **THEN** its protocol mapping retains A2A source identity, terminal state, Run/Step correlation, and any available artifact reference
