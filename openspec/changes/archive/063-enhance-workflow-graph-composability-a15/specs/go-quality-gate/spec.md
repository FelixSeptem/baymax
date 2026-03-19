## ADDED Requirements

### Requirement: Quality gate SHALL include workflow graph composability contract suites
Shared multi-agent quality gate MUST include workflow graph composability contract suites as blocking checks.

#### Scenario: CI executes shared contract gate after A15 rollout
- **WHEN** CI runs `check-multi-agent-shared-contract.*`
- **THEN** workflow graph composability suites run as blocking checks in the same shared gate path

### Requirement: Workflow graph composability gate SHALL block compile-boundary regressions
Workflow graph composability contract suites MUST fail on depth-limit regression, alias/id collision acceptance, template-scope violation acceptance, or forbidden kind-override acceptance.

#### Scenario: Regression accepts kind override in subgraph instance
- **WHEN** test suite detects forbidden `kind` override no longer fails validation
- **THEN** quality gate fails and blocks merge

### Requirement: Mainline contract index SHALL map workflow graph composability coverage
Mainline contract index MUST include traceable mapping for A15 core scenarios: expansion determinism, compile fail-fast, Run/Stream equivalence, and resume consistency.

#### Scenario: Contributor audits A15 coverage
- **WHEN** contributor checks mainline contract index
- **THEN** each required A15 contract row maps to concrete test cases
