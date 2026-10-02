## Purpose

This capability gives minimal local and test model clients a deterministic, explicit way to participate in provider identity and capability preflight without expanding the core `ModelClient` contract.

## ADDED Requirements

### Requirement: Explicit model capability adapters SHALL declare identity and capabilities
The runtime integration surface SHALL provide an opt-in adapter that delegates the wrapped model client's Generate and Stream operations while exposing a non-empty provider identity and an explicitly configured capability report. The adapter MUST NOT infer capabilities from the wrapped method set.

#### Scenario: Wrapped streaming model declares streaming support
- **WHEN** an embedding application wraps a valid `types.ModelClient` with provider `local` and declares `streaming`
- **THEN** the adapter delegates Generate and Stream and capability discovery reports provider `local` with `streaming` supported

#### Scenario: Wrapped model omits a capability
- **WHEN** a capability is requested that is not present in the adapter declaration
- **THEN** capability discovery reports that capability as unsupported or unknown according to the canonical provider capability contract, and the runner does not invoke the model under strict preflight

### Requirement: Explicit model capability adapters SHALL fail fast on invalid configuration
Adapter construction MUST reject a nil model, blank provider identity, unknown model capability, or a capability declaration that normalizes to an invalid value. Duplicate normalized capabilities MUST be collapsed deterministically.

#### Scenario: Adapter configuration is incomplete
- **WHEN** an embedding application constructs an adapter without a model or provider identity
- **THEN** construction returns a deterministic validation error and no runnable adapter is produced

#### Scenario: Adapter declaration contains duplicate capabilities
- **WHEN** an embedding application declares the same capability more than once with equivalent casing or whitespace
- **THEN** construction succeeds with one normalized capability entry in deterministic order

### Requirement: Adapter delegation SHALL preserve model behavior and errors
The adapter MUST return wrapped model responses, stream callbacks, and underlying errors without changing their semantic values or introducing a second execution path.

#### Scenario: Wrapped Generate returns an error
- **WHEN** the wrapped model returns an error from Generate
- **THEN** the adapter returns the same error to the caller without remapping it

#### Scenario: Wrapped Stream emits events and returns an error
- **WHEN** the wrapped model emits events and then returns an error from Stream
- **THEN** the adapter forwards each callback and returns the underlying error unchanged
