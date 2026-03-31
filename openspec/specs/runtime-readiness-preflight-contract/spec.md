# runtime-readiness-preflight-contract Specification

## Purpose
TBD - created by archiving change introduce-runtime-readiness-preflight-and-degradation-contract-a40. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL expose library-level readiness preflight API
Runtime MUST expose a library-level readiness preflight API that returns deterministic readiness result without requiring platform control-plane dependencies.

The API MUST return:
- overall status (`ready|degraded|blocked`),
- structured findings list,
- evaluation timestamp.

#### Scenario: Host invokes readiness preflight before startup
- **WHEN** application calls readiness preflight on runtime manager with valid config snapshot
- **THEN** runtime returns deterministic readiness result with status and structured findings

#### Scenario: Host invokes readiness preflight repeatedly with unchanged snapshot
- **WHEN** application calls readiness preflight multiple times without configuration or dependency changes
- **THEN** runtime returns semantically equivalent readiness status and finding classifications

### Requirement: Readiness classification SHALL support strict degradation policy
Readiness classification MUST support `strict` policy:
- when `strict=false`, degraded conditions MUST map to `degraded`,
- when `strict=true`, degraded conditions MUST escalate to `blocked`.

#### Scenario: Degraded condition under non-strict policy
- **WHEN** runtime detects recoverable degraded finding and `strict=false`
- **THEN** readiness status is `degraded` and findings remain observable

#### Scenario: Degraded condition under strict policy
- **WHEN** runtime detects same degraded finding and `strict=true`
- **THEN** readiness status escalates to `blocked`

### Requirement: Readiness findings SHALL use canonical structured schema
Each readiness finding MUST use canonical fields:
- `code`
- `domain`
- `severity`
- `message`
- `metadata`

Finding `code` values MUST be stable and machine-assertable for contract tests.

#### Scenario: Consumer inspects readiness findings
- **WHEN** readiness preflight returns one or more findings
- **THEN** each finding includes canonical fields and stable machine-readable code

#### Scenario: Regression introduces non-canonical finding shape
- **WHEN** implementation omits required finding field or renames canonical key
- **THEN** contract validation fails and blocks merge

### Requirement: Preflight SHALL include fallback and backend-activation visibility
Readiness preflight MUST include checks for scheduler/mailbox/recovery activation and fallback outcomes in effective runtime snapshot.

Fallback-aware findings MUST classify deterministic degraded conditions when configured persistent backend falls back to memory.

#### Scenario: Persistent backend falls back to memory
- **WHEN** effective runtime path uses fallback-to-memory for scheduler or mailbox
- **THEN** readiness returns degraded finding with deterministic fallback reason metadata

#### Scenario: Invalid mandatory backend activation
- **WHEN** effective runtime configuration requires backend activation and activation fails without valid fallback
- **THEN** readiness returns blocked finding and overall status is `blocked`

### Requirement: Readiness preflight SHALL incorporate adapter-health findings
Runtime readiness preflight MUST evaluate adapter health findings as part of readiness classification when adapter-health feature is enabled.

Readiness output MUST preserve canonical finding schema while adding adapter-domain findings.

#### Scenario: Required adapter is unavailable under strict policy
- **WHEN** readiness preflight detects required adapter health status `unavailable` and `runtime.readiness.strict=true`
- **THEN** readiness overall status is `blocked` and includes canonical adapter-health blocking finding

#### Scenario: Optional adapter is unavailable under non-strict policy
- **WHEN** readiness preflight detects optional adapter health status `unavailable` and `runtime.readiness.strict=false`
- **THEN** readiness overall status is `degraded` and includes canonical adapter-health degraded finding

### Requirement: Adapter-health readiness mapping SHALL remain deterministic
For equivalent adapter inventory state and effective configuration, readiness classification and primary finding code MUST remain deterministic.

#### Scenario: Repeated preflight with unchanged adapter state
- **WHEN** host runs readiness preflight repeatedly without adapter or config changes
- **THEN** readiness status and adapter-health finding codes remain semantically equivalent

#### Scenario: Regression introduces non-canonical adapter finding code
- **WHEN** implementation emits adapter-health finding code outside canonical taxonomy
- **THEN** contract validation fails and blocks merge

### Requirement: Readiness result SHALL be consumable by admission guard with deterministic mapping
Runtime readiness preflight output MUST remain a deterministic input to readiness-admission guard mapping.

For equivalent readiness status and findings, admission mapping inputs MUST remain semantically stable across repeated evaluations.

#### Scenario: Equivalent readiness outputs feed identical admission input semantics
- **WHEN** host triggers repeated readiness preflight calls under unchanged runtime snapshot
- **THEN** resulting status/finding semantics consumed by admission guard remain equivalent

#### Scenario: Readiness primary code is preserved for admission reasoning
- **WHEN** readiness preflight produces blocking or degraded primary code
- **THEN** admission path can consume the same canonical primary code without reclassification drift

### Requirement: Runtime readiness preflight SHALL classify adapter-governance outcomes deterministically
Runtime readiness preflight MUST classify adapter-health governance outcomes using canonical findings and MUST preserve strict/non-strict escalation semantics.

At minimum, readiness findings MUST cover:
- circuit-open sustained unavailable path
- half-open degraded probe path
- governance recovery path after successful half-open probes

Canonical finding codes for governance paths MUST remain in `adapter.health.*` namespace.

#### Scenario: Strict mode escalates sustained circuit-open required adapter
- **WHEN** required adapter remains unavailable with circuit held in `open` under strict mode
- **THEN** preflight returns `blocked` with canonical `adapter.health.*` finding code

#### Scenario: Non-strict mode degrades optional adapter during circuit-open window
- **WHEN** optional adapter is unavailable while governance keeps circuit `open`
- **THEN** preflight returns `degraded` and preserves runnable runtime behavior with recorded findings

### Requirement: Readiness findings SHALL align with composite replay fixtures
Runtime readiness preflight findings MUST remain alignable with A47 composite replay fixtures through canonical fields and stable finding-code taxonomy.

Readiness fixture assertions MUST cover:
- strict/non-strict classification path,
- primary finding code stability,
- degraded-to-blocked escalation semantics.

#### Scenario: Strict escalation is validated through composite fixture
- **WHEN** composite fixture models degraded finding under strict readiness policy
- **THEN** replay assertion confirms blocked classification with canonical readiness code mapping

#### Scenario: Readiness taxonomy drifts from canonical mapping
- **WHEN** composite fixture detects non-canonical readiness finding code
- **THEN** replay fixture validation fails and blocks gate

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

### Requirement: Readiness preflight SHALL include arbitration explainability alignment
Readiness preflight output MUST preserve alignment between primary reason and explainability metadata, including bounded secondary reasons and remediation hint taxonomy.

#### Scenario: Preflight returns blocked with explainability payload
- **WHEN** readiness preflight produces blocked status and arbitration metadata
- **THEN** output includes canonical primary reason plus bounded secondary reasons and remediation hint fields

#### Scenario: Equivalent preflight inputs are evaluated repeatedly
- **WHEN** runtime runs repeated preflight with unchanged inputs
- **THEN** explainability output remains semantically equivalent and deterministically ordered

### Requirement: Readiness preflight SHALL classify arbitration-version compatibility deterministically
Readiness preflight MUST evaluate arbitration version-governance compatibility and emit canonical findings for unsupported-version and compatibility-mismatch paths.

Readiness findings for version governance MUST remain machine-assertable and deterministic under equivalent inputs.

#### Scenario: Preflight detects unsupported arbitration rule version
- **WHEN** runtime preflight receives requested arbitration version that is not supported
- **THEN** readiness output includes canonical unsupported-version finding and deterministic blocking classification

#### Scenario: Preflight detects compatibility-window mismatch
- **WHEN** requested arbitration version is registered but outside configured compatibility window
- **THEN** readiness output includes canonical mismatch finding and deterministic classification aligned with policy

### Requirement: Readiness preflight SHALL expose arbitration-version explainability fields
Readiness preflight output MUST expose arbitration-version explainability fields that align with arbitration diagnostics:
- requested version,
- effective version,
- version source,
- policy action.

#### Scenario: Preflight uses default arbitration version
- **WHEN** preflight runs without per-request version override
- **THEN** readiness output includes effective default version and deterministic source metadata

#### Scenario: Preflight uses requested arbitration version
- **WHEN** preflight runs with supported requested version override
- **THEN** readiness output includes requested/effective version alignment without reclassification drift

### Requirement: Readiness preflight SHALL evaluate sandbox-required availability deterministically
When sandbox governance is enabled with `required=true`, readiness preflight MUST evaluate sandbox executor availability and profile validity as blocking preconditions.

Unavailable or invalid required sandbox dependency MUST produce blocking readiness finding with canonical machine-readable code.

#### Scenario: Required sandbox executor is unavailable
- **WHEN** sandbox is enabled with `required=true` and executor probe fails
- **THEN** readiness preflight returns `blocked` with canonical sandbox-unavailable finding

#### Scenario: Required sandbox profile is invalid
- **WHEN** sandbox is enabled with `required=true` and selected profile validation fails
- **THEN** readiness preflight returns `blocked` with canonical sandbox-profile-invalid finding

#### Scenario: Required sandbox capability is not supported by backend
- **WHEN** sandbox is enabled with `required=true` and executor probe does not satisfy required capabilities
- **THEN** readiness preflight returns `blocked` with canonical sandbox-capability-mismatch finding

#### Scenario: Required sandbox session mode is unsupported
- **WHEN** sandbox is enabled with `required=true` and configured session mode is unsupported by executor/backend
- **THEN** readiness preflight returns `blocked` with canonical sandbox-session-mode-unsupported finding

### Requirement: Non-required sandbox degradation SHALL remain observable without forced blocking
When sandbox governance is enabled with `required=false`, sandbox dependency issues MUST remain observable and MUST follow readiness strict/non-strict classification semantics.

#### Scenario: Non-required sandbox executor unavailable under non-strict policy
- **WHEN** sandbox is enabled with `required=false`, executor probe fails, and readiness strict mode is disabled
- **THEN** readiness preflight returns degraded-class finding and keeps runtime runnable

#### Scenario: Non-required sandbox issue under strict policy
- **WHEN** sandbox is enabled with `required=false`, sandbox finding is degraded-class, and readiness strict mode is enabled
- **THEN** readiness classification escalates to blocked according to strict policy contract

### Requirement: Readiness preflight SHALL evaluate sandbox rollout-governance readiness findings
Runtime readiness preflight MUST evaluate rollout-governance state and emit canonical findings when rollout configuration or state is unsafe for managed execution.

Canonical finding codes for this milestone MUST include:
- `sandbox.rollout.phase_invalid`
- `sandbox.rollout.health_budget_breached`
- `sandbox.rollout.frozen`
- `sandbox.rollout.capacity_unavailable`

#### Scenario: Preflight detects frozen rollout state
- **WHEN** effective rollout phase is `frozen`
- **THEN** readiness returns canonical `sandbox.rollout.frozen` finding and deterministic status classification

#### Scenario: Preflight detects health budget breach
- **WHEN** rollout health budget classification is `breached`
- **THEN** readiness returns canonical `sandbox.rollout.health_budget_breached` finding with machine-readable metadata

### Requirement: Rollout-governance findings SHALL preserve strict/non-strict mapping semantics
Readiness preflight MUST map rollout-governance findings using existing strict/non-strict policy semantics:
- `strict=false` may classify recoverable rollout risk as `degraded`
- `strict=true` MUST escalate equivalent blocking-class finding to `blocked`

#### Scenario: Non-strict policy with capacity pressure
- **WHEN** rollout finding indicates capacity pressure under non-strict readiness policy
- **THEN** readiness status is `degraded` with canonical rollout-capacity finding

#### Scenario: Strict policy with equivalent capacity pressure
- **WHEN** the same rollout-capacity finding is evaluated with strict readiness policy
- **THEN** readiness status escalates to `blocked`

### Requirement: Readiness preflight SHALL evaluate sandbox adapter profile-pack availability and compatibility
Runtime readiness preflight MUST evaluate sandbox adapter profile-pack dependencies when sandbox mode is enabled.

Canonical finding codes for this milestone MUST include:
- `sandbox.adapter.profile_missing`
- `sandbox.adapter.backend_not_supported`
- `sandbox.adapter.host_mismatch`
- `sandbox.adapter.session_mode_unsupported`

#### Scenario: Required sandbox adapter profile is missing
- **WHEN** effective sandbox configuration references non-existent adapter profile
- **THEN** readiness preflight returns canonical `sandbox.adapter.profile_missing` finding

#### Scenario: Host backend support is unavailable
- **WHEN** referenced adapter profile backend is unsupported on current host/runtime
- **THEN** readiness preflight returns canonical `sandbox.adapter.backend_not_supported` finding

### Requirement: Sandbox adapter profile findings SHALL preserve strict/non-strict mapping semantics
Readiness preflight MUST map sandbox adapter profile findings using existing strict/non-strict rules:
- non-strict mode MAY classify recoverable adapter-profile issues as `degraded`,
- strict mode MUST escalate equivalent blocking-class findings to `blocked`.

#### Scenario: Non-strict mode with non-required adapter profile issue
- **WHEN** preflight evaluates recoverable sandbox adapter profile issue under non-strict policy
- **THEN** readiness status is `degraded` with canonical sandbox adapter finding

#### Scenario: Strict mode with equivalent adapter profile issue
- **WHEN** same issue is evaluated under strict policy
- **THEN** readiness status escalates to `blocked`

### Requirement: Readiness preflight SHALL evaluate memory backend profile and contract compatibility
Runtime readiness preflight MUST evaluate memory backend dependencies when memory capability is enabled.

Canonical finding codes for this milestone MUST include:
- `memory.mode_invalid`
- `memory.profile_missing`
- `memory.provider_not_supported`
- `memory.spi_unavailable`
- `memory.filesystem_path_invalid`
- `memory.contract_version_mismatch`

#### Scenario: External memory profile is missing
- **WHEN** effective memory config uses `external_spi` and referenced profile cannot be resolved
- **THEN** readiness returns canonical `memory.profile_missing` finding

#### Scenario: Builtin filesystem path is invalid
- **WHEN** effective memory config uses `builtin_filesystem` and configured root path is not usable
- **THEN** readiness returns canonical `memory.filesystem_path_invalid` finding

### Requirement: Memory readiness findings SHALL preserve strict and non-strict mapping semantics
Readiness preflight MUST apply existing strict/non-strict mapping to memory findings:
- non-strict mode MAY classify recoverable memory findings as `degraded`,
- strict mode MUST escalate equivalent blocking-class findings to `blocked`.

#### Scenario: Non-strict mode with recoverable SPI unavailable finding
- **WHEN** preflight detects recoverable external SPI unavailability under non-strict policy
- **THEN** readiness status is `degraded` with canonical memory finding

#### Scenario: Strict mode with equivalent SPI unavailable finding
- **WHEN** the same finding is evaluated under strict policy
- **THEN** readiness status escalates to `blocked`

### Requirement: Readiness preflight SHALL include memory fallback safety findings
Readiness preflight MUST validate whether configured memory fallback policy is executable for the active mode and environment.

Canonical safety findings MUST include:
- `memory.fallback_policy_conflict`
- `memory.fallback_target_unavailable`

#### Scenario: Fallback policy conflicts with active mode
- **WHEN** config sets fallback target inconsistent with active memory mode constraints
- **THEN** readiness returns canonical `memory.fallback_policy_conflict` finding

#### Scenario: Degrade-to-builtin policy but builtin target unavailable
- **WHEN** fallback policy is `degrade_to_builtin` but builtin backend fails readiness checks
- **THEN** readiness returns canonical `memory.fallback_target_unavailable` finding with deterministic status classification

### Requirement: Readiness preflight SHALL evaluate observability export and diagnostics bundle dependencies
Runtime readiness preflight MUST evaluate observability export sink and diagnostics bundle storage dependencies when corresponding features are enabled.

Canonical finding codes for this milestone MUST include:
- `observability.export.profile_invalid`
- `observability.export.sink_unavailable`
- `observability.export.auth_invalid`
- `diagnostics.bundle.output_unavailable`
- `diagnostics.bundle.policy_invalid`

Findings MUST use canonical structured schema and remain machine-assertable.

#### Scenario: Preflight detects unavailable export sink
- **WHEN** export profile is enabled and sink probe fails
- **THEN** readiness preflight returns canonical `observability.export.sink_unavailable` finding

#### Scenario: Preflight detects unwritable bundle output path
- **WHEN** bundle generation is enabled and configured output path is unavailable
- **THEN** readiness preflight returns canonical `diagnostics.bundle.output_unavailable` finding

### Requirement: Observability readiness findings SHALL preserve strict non-strict mapping semantics
Readiness classification for observability and bundle findings MUST follow existing strict/non-strict policy semantics:
- `strict=false` may classify recoverable observability findings as `degraded`,
- `strict=true` MUST escalate equivalent blocking-class findings to `blocked`.

#### Scenario: Non-strict policy with export sink unavailable
- **WHEN** preflight receives `observability.export.sink_unavailable` and `runtime.readiness.strict=false`
- **THEN** readiness status is `degraded` with canonical finding preserved

#### Scenario: Strict policy with same export finding
- **WHEN** equivalent preflight input is evaluated with `runtime.readiness.strict=true`
- **THEN** readiness status escalates to `blocked` with deterministic canonical finding

### Requirement: Readiness preflight SHALL evaluate ReAct prerequisite availability
Runtime readiness preflight MUST evaluate ReAct prerequisite dependencies when `runtime.react.enabled=true`.

Canonical finding codes for this milestone MUST include:
- `react.loop_disabled`
- `react.stream_dispatch_unavailable`
- `react.provider_tool_calling_unsupported`
- `react.tool_registry_unavailable`
- `react.sandbox_dependency_unavailable`

#### Scenario: ReAct enabled but provider lacks tool-calling capability
- **WHEN** preflight evaluates effective config with ReAct enabled and active provider cannot satisfy tool-calling requirements
- **THEN** readiness returns canonical finding `react.provider_tool_calling_unsupported`

#### Scenario: ReAct enabled but Stream dispatch dependency is unavailable
- **WHEN** preflight evaluates managed Stream path and stream dispatch prerequisite is not available
- **THEN** readiness returns canonical finding `react.stream_dispatch_unavailable`

### Requirement: ReAct readiness findings SHALL preserve strict and non-strict mapping semantics
Readiness preflight MUST apply existing strict/non-strict mapping for ReAct findings:
- non-strict mode MAY classify recoverable ReAct findings as `degraded`,
- strict mode MUST escalate equivalent blocking-class findings to `blocked`.

#### Scenario: Non-strict mode with recoverable tool-registry finding
- **WHEN** preflight detects recoverable `react.tool_registry_unavailable` and `runtime.readiness.strict=false`
- **THEN** readiness status is `degraded` with canonical ReAct finding

#### Scenario: Strict mode with equivalent ReAct finding
- **WHEN** preflight evaluates equivalent finding with `runtime.readiness.strict=true`
- **THEN** readiness status escalates to `blocked`

### Requirement: ReAct readiness output SHALL remain deterministic for equivalent snapshots
For equivalent runtime snapshot and dependency state, readiness preflight MUST return semantically equivalent ReAct status and finding codes.

#### Scenario: Repeated preflight with unchanged ReAct dependencies
- **WHEN** host calls readiness preflight repeatedly without config or dependency changes
- **THEN** ReAct-related status and finding-code semantics remain equivalent

#### Scenario: Regression emits non-canonical ReAct finding code
- **WHEN** implementation outputs ReAct finding code outside canonical taxonomy
- **THEN** contract validation fails and blocks merge

