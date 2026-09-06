## ADDED Requirements

### Requirement: Handoff restore SHALL use reference-first organization
Handoff content SHALL be injected through the existing reference-first and isolate-handoff flow; swap-back and lifecycle tiering SHALL resolve referenced bodies only when relevant and SHALL preserve source ownership.

#### Scenario: Relevant reference swap-back
- **WHEN** a restored next action requires a cold referenced artifact
- **THEN** the assembler swaps back only the relevant body and retains the canonical reference in the handoff
