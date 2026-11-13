# sandbox-external-executor-backends Specification

## Purpose
TBD - created by archiving change introduce-mainstream-sandbox-executor-backends. Update Purpose after archive.

## Requirements

### Requirement: Runtime SHALL register mainstream sandbox runtimes as canonical backend profile variants
The runtime MUST register mainstream sandbox runtimes (e.g. anthropics/sandbox-runtime, open-sandbox.ai, e2b) as variants of the existing canonical backend profiles (`linux_nsjail` / `linux_bwrap` / `oci_runtime` / `windows_job`) through `runtime/config` external backend registration plus the `adapter/sandboxexternal` implementation, so their sandbox execution remains interchangeable under the existing `SandboxExecutor` contract.

#### Scenario: e2b registers as oci_runtime variant
- **WHEN** `security.sandbox.executor.external_backend=e2b` is configured with `backend=oci_runtime`
- **THEN** configuration validation accepts the pair and `ResolveSandboxExternalBackend` reports the `oci_runtime` canonical profile

#### Scenario: unregistered or mismatched backend is rejected
- **WHEN** the configured external runtime is unregistered, or its canonical mapping disagrees with `executor.backend`
- **THEN** configuration validation and hot reload fail fast with an error naming `security.sandbox.executor.external_backend`

### Requirement: External sandbox adapter SHALL implement the host-provided SandboxExecutor contract
Each mainstream runtime adapter MUST implement `SandboxExecutor` by translating `SandboxExecSpec` (command/env/mount/network/resource/session/timeout) into the target runtime call and normalizing the outcome into `SandboxExecResult` (exit code / timeout / oom / violation codes / resource summary).

#### Scenario: adapter executes a sandboxed call successfully
- **WHEN** the executor returns a successful execution result
- **THEN** the runtime maps output into the normalized tool result without semantic drift from the non-sandbox success path

#### Scenario: adapter executor is nil
- **WHEN** the sandbox action resolves to `sandbox` but no executor is injected
- **THEN** the runtime returns `sandboxReasonLaunchFailed` and does not execute on host

### Requirement: External sandbox adapter SHALL negotiate capabilities before execution
The adapter MUST run a `SandboxCapabilityProbe` during readiness/admission and fail-fast deny when `required_capabilities` are not satisfied.

#### Scenario: required capability missing
- **WHEN** the adapter probe reports a missing required capability
- **THEN** the runtime denies with `capability_mismatch` and records a stable reason code

### Requirement: External sandbox adapter SHALL use frozen failure classification
Failure classification for external adapters MUST be frozen as `launch_failed` / `capability_mismatch` / `timeout` / `oom` / `egress_denied` / `backend_unavailable`, reusing the existing `sandbox-*-matrix`, without introducing a parallel contract.

#### Scenario: egress denied by governance
- **WHEN** the sandbox egress policy denies the outbound attempt
- **THEN** the runtime emits `egress_denied` and does not execute the network call

#### Scenario: backend unavailable
- **WHEN** the target sandbox backend cannot be reached
- **THEN** the runtime emits `backend_unavailable` and falls back per `DefaultAction`

### Requirement: External sandbox adapters SHALL be verified through conformance harness
Each mainstream backend MUST be covered by `integration/sandboxconformance` `sandbox-<backend>-matrix` fixtures (capability probe / normal / timeout / oom / egress deny / backend unavailable / Run-Stream parity), and the gate MUST run shell/PowerShell parity.

#### Scenario: conformance gate blocks on matrix drift
- **WHEN** the external failure matrix misses a runtime/kind pair, or a fixture expects a reason code that differs from the frozen taxonomy
- **THEN** `scripts/check-sandbox-external-executor-conformance` fails the quality gate

#### Scenario: remote call without credentials
- **WHEN** a fixture requires a live remote backend but no credentials are configured
- **THEN** the check reports `skipped-unconfigured` rather than pass or fail
