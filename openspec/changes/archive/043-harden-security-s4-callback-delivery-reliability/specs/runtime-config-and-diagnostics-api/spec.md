## ADDED Requirements

### Requirement: Runtime config SHALL expose S4 delivery controls with deterministic precedence and fail-fast validation
Runtime configuration MUST expose S4 delivery controls under `security.security_event.delivery` with precedence `env > file > default`.
At minimum, configuration MUST include delivery mode, queue bounds/overflow policy, timeout, retry settings, and circuit breaker controls.
Invalid delivery enum or malformed numeric threshold values MUST fail fast during startup and hot reload.

#### Scenario: Startup resolves S4 delivery defaults
- **WHEN** runtime starts without explicit delivery overrides
- **THEN** effective config resolves valid defaults including `mode=async`, bounded queue, retry budget, and circuit breaker baseline

#### Scenario: Invalid S4 delivery hot-reload update is rejected
- **WHEN** runtime receives malformed delivery config during hot reload
- **THEN** runtime rejects update, records reload failure diagnostics, and keeps previous valid snapshot

### Requirement: Runtime diagnostics SHALL expose additive S4 delivery observability fields
Runtime diagnostics MUST expose additive delivery fields for security alerts, including at minimum delivery mode, retry count, queue-drop marker/count, circuit state, and delivery failure reason.
These fields MUST remain backward-compatible with existing diagnostics consumers.

#### Scenario: Consumer inspects retry and circuit diagnostics
- **WHEN** deny alert delivery experiences retries or circuit transitions
- **THEN** run diagnostics include normalized retry and circuit state markers

#### Scenario: Consumer inspects queue overflow diagnostics
- **WHEN** deny alerts exceed bounded queue capacity under async mode
- **THEN** diagnostics include queue overflow/drop markers with configured overflow-policy semantics

### Requirement: Run and Stream SHALL preserve S4 diagnostics semantic equivalence
For equivalent requests and effective configuration, Run and Stream MUST emit semantically equivalent S4 delivery diagnostics fields.

#### Scenario: Equivalent S4 diagnostics in Run and Stream
- **WHEN** equivalent deny alerts are produced in Run and Stream
- **THEN** delivery-mode, retry, queue-drop, and circuit-state diagnostics are semantically equivalent
