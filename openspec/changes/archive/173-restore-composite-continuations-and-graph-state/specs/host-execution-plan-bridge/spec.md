## ADDED Requirements

### Requirement: Resumed plans SHALL delegate only to an evidenced source owner

The host plan bridge MAY admit a resumed trigger only when a compatible
source-owned continuation binding, manifest cell, checkpoint reference, and
atomic restore entry are supplied. Compilation and admission MUST reject before
enqueue/publish/Run when any binding is missing or incompatible, and MUST NOT
fall back to direct or inline execution.

#### Scenario: Resumed child plan delegates once
- **WHEN** a resumed plan supplies a compatible child continuation binding and
  current host admission facts
- **THEN** the bridge invokes the child owner restore entry once and exposes its
  native projection without creating a replacement parent

#### Scenario: Resumed plan without source binding is rejected
- **WHEN** a resumed plan supplies only a checkpoint reference or reused run ID
- **THEN** compilation returns `unsupported_capability` before source activation
