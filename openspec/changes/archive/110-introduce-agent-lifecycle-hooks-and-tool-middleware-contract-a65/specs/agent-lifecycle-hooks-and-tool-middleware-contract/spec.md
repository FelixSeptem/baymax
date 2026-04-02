## ADDED Requirements

### Requirement: Lifecycle Hooks Execution Contract
The runtime MUST expose lifecycle hooks at `before_reasoning`, `after_reasoning`, `before_acting`, `after_acting`, `before_reply`, and `after_reply`, and MUST execute them in deterministic phase order for both `Run` and `Stream`.

#### Scenario: Deterministic phase order
- **WHEN** the same request is executed once via `Run` and once via `Stream` with identical hook registration
- **THEN** hook invocation phase sequence MUST be identical across both executions

#### Scenario: Hook fail-fast policy
- **WHEN** a hook returns an error under `fail_fast` policy
- **THEN** the runtime MUST stop subsequent hook execution and return a deterministic classified failure

#### Scenario: Hook degrade policy
- **WHEN** a hook returns an error under `degrade` policy
- **THEN** the runtime MUST continue execution with degraded marker and MUST record the hook failure reason

### Requirement: Tool Middleware Onion-Chain Contract
The runtime MUST execute tool middleware as onion-chain semantics: inbound in registration order, outbound in reverse order, with deterministic short-circuit and error propagation behavior.

#### Scenario: Onion-chain order
- **WHEN** three middleware entries are registered in order `m1,m2,m3`
- **THEN** inbound execution MUST be `m1->m2->m3` and outbound execution MUST be `m3->m2->m1`

#### Scenario: Middleware short-circuit
- **WHEN** an inbound middleware returns a short-circuit result without calling next
- **THEN** downstream middleware and tool invocation MUST be skipped and response MUST remain deterministic

#### Scenario: Middleware timeout isolation
- **WHEN** middleware execution exceeds configured timeout budget
- **THEN** the runtime MUST classify timeout deterministically and MUST NOT leave hanging middleware execution paths

### Requirement: Security and Policy Boundary Preservation
Hooks and middleware MUST NOT bypass existing policy precedence, sandbox governance, adapter allowlist, or egress restrictions.

`control_plane_absent`: A65 runtime hooks/middleware and skill preprocess/mapping MUST NOT require hosted control-plane services, managed orchestrators, or remote scheduler dependencies; all semantics remain embedded library behavior.

#### Scenario: Deny decision cannot be overridden
- **WHEN** upstream policy precedence resolves final decision to deny
- **THEN** hook or middleware logic MUST NOT override the deny result to allow

#### Scenario: Whitelist upper-bound enforcement
- **WHEN** `SkillBundle` mapping proposes tool whitelist entries outside sandbox/allowlist boundary
- **THEN** the runtime MUST reject overflow entries and record deterministic reason code

### Requirement: Single-Writer Observability for Hook/Middleware Events
Hook and middleware observability output MUST be emitted through `RuntimeRecorder` single-writer path and MUST preserve additive diagnostics compatibility.

#### Scenario: RuntimeRecorder single-writer path
- **WHEN** hook and middleware events are emitted during execution
- **THEN** diagnostics records MUST be written only through runtime recorder managed path

#### Scenario: Additive diagnostics compatibility
- **WHEN** new hook/middleware fields are introduced
- **THEN** existing diagnostics parsers MUST remain compatible via additive nullable default semantics
