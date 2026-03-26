## ADDED Requirements

### Requirement: Readiness preflight SHALL expose arbitration-aligned primary reason output
Readiness preflight output MUST include primary reason fields aligned with cross-domain arbitration semantics and MUST preserve canonical readiness taxonomy.

Readiness primary reason output MUST remain consistent with:
- preflight status classification,
- canonical finding codes,
- cross-domain precedence and tie-break rules.

#### Scenario: Preflight returns blocked with concurrent timeout finding
- **WHEN** preflight context includes timeout reject and readiness blocked findings
- **THEN** primary reason output follows cross-domain arbitration precedence and remains deterministic

#### Scenario: Preflight returns degraded with optional adapter unavailable
- **WHEN** preflight context includes degraded readiness and optional adapter unavailable
- **THEN** primary reason output uses canonical degraded-level arbitration and remains machine-assertable
