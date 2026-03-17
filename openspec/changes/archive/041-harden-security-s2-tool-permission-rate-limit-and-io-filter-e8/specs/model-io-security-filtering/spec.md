## ADDED Requirements

### Requirement: Runtime SHALL provide pluggable model input and output security filter interfaces
The runtime MUST expose extension interfaces for model input filtering and model output filtering so host applications can register custom security filters.

#### Scenario: Host registers custom input and output filters
- **WHEN** application provides valid filter implementations during runtime setup
- **THEN** runtime accepts registration and executes filters in configured pipeline order

#### Scenario: Host provides incompatible filter implementation
- **WHEN** application registers a filter that does not satisfy interface contract
- **THEN** runtime rejects registration with explicit validation error

### Requirement: Runtime SHALL execute input filter before model invocation and output filter before emission
Model input filters MUST run before provider invocation, and model output filters MUST run before final response is returned or streamed to consumers.

#### Scenario: Input filter transforms prompt payload
- **WHEN** input filter marks content as allowed with transformed payload
- **THEN** runtime invokes model provider with transformed input

#### Scenario: Output filter transforms model response
- **WHEN** output filter marks generated response as allowed with transformed output
- **THEN** runtime returns transformed output to downstream consumer

### Requirement: Runtime SHALL deny execution on blocking security-filter outcome
If input or output filter returns blocking outcome, runtime MUST fail fast with deny semantics and MUST NOT continue the blocked path.

#### Scenario: Input filter blocks request
- **WHEN** input filter returns blocking decision for current request
- **THEN** runtime terminates model invocation path with deny result and does not call provider

#### Scenario: Output filter blocks generated content
- **WHEN** output filter returns blocking decision for generated content
- **THEN** runtime terminates response emission with deny result

### Requirement: Model I/O filtering SHALL emit normalized observability fields
Filter execution MUST emit additive diagnostics fields including filter stage (`input|output`), match result, and normalized reason code.

#### Scenario: Input filter hit is recorded
- **WHEN** input filter evaluates content and produces a hit or block outcome
- **THEN** diagnostics record stage=`input` with normalized filter result and reason code

#### Scenario: Output filter hit is recorded
- **WHEN** output filter evaluates generated content and produces a hit or block outcome
- **THEN** diagnostics record stage=`output` with normalized filter result and reason code

### Requirement: Run and Stream SHALL keep model I/O filter semantic equivalence
For equivalent inputs and effective configuration, Run and Stream MUST produce semantically equivalent filtering decisions and deny semantics.

#### Scenario: Equivalent input filter block in Run and Stream
- **WHEN** equivalent requests trigger the same blocking input filter rule in both Run and Stream
- **THEN** both paths terminate before provider invocation with semantically equivalent deny semantics

#### Scenario: Equivalent output filter block in Run and Stream
- **WHEN** equivalent generated content triggers the same blocking output filter rule in both Run and Stream
- **THEN** both paths terminate response emission with semantically equivalent deny semantics