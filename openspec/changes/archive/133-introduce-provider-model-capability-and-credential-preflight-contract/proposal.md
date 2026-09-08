## Why

Baymax has provider clients, adapter capability negotiation, and runtime readiness
findings, but it has no single contract that answers whether a configured model
can satisfy a requested workload before Run or Stream begins. Hosts therefore
cannot deterministically distinguish an unknown model, an unsupported required
capability, a missing credential, or an optional fallback without provider-
specific inspection.

The Pi comparison identified model metadata and credential preflight as a useful
embedded-runtime seam. This is timely after extension lifecycle governance: a
host can now validate extension requirements, but still needs a source-owned
provider/model admission result before enabling an extension or dispatching a
model action.

## What Changes

- Add a library-owned, host-injected provider/model capability catalog with
  stable provider/model identity, catalog version, context window, and declared
  capabilities. The catalog is bounded, deterministic, and has no remote
  discovery or background refresh behavior.
- Add credential preflight contracts that evaluate credential presence and
  optional host-supplied validation evidence without recording secrets,
  endpoints, tokens, or raw provider responses.
- Reuse `adapter/capability` required/optional negotiation and existing
  readiness status mapping to project model selection as `ready`, `degraded`,
  or `blocked`. Required capability or credential failures are side-effect-free;
  optional gaps may use only configured, declared fallbacks.
- Add `runtime.provider_catalog.*` configuration with `env > file > default`,
  fail-fast validation, and atomic hot-reload rollback. Catalog/config changes
  apply only to later admissions and must not alter facts already recorded for
  an in-flight Run.
- Project bounded catalog identity, model identity, capability outcome,
  credential preflight status, fallback, and canonical reason fields through
  `RuntimeRecorder` into additive nullable diagnostics fields.
- Add deterministic replay fixtures, Run/Stream parity coverage, dedicated
  shell/PowerShell contract gates, and an agent-mode documentation update for
  supported, degraded, denied, and rollback paths.

## Capabilities

### New Capabilities

- `provider-model-capability-and-credential-preflight`: host-injected model
  catalog validation, required/optional capability negotiation, credential
  preflight, deterministic fallback, and replayable admission decisions.

### Modified Capabilities

- `runtime-readiness-preflight-contract`: incorporate provider/model catalog
  and credential findings into canonical readiness aggregation and strict-mode
  escalation without adding a second admission state machine.
- `runtime-config-and-diagnostics-api`: define additive runtime configuration
  and diagnostics projections for catalog and credential-preflight outcomes.

## Example Impact Assessment

修改示例

Update the existing `adapter-onboarding-manifest-capability` agent-mode README
and expected markers to show catalog admission and credential-preflight
outcomes; no new example pattern is needed.

## Impact

- Affected packages: `model/<provider>`, a new library-owned model catalog
  package, `adapter/capability`, `runtime/config`, `runtime/diagnostics`, and
  `observability/event`.
- Affected contracts: runtime readiness, configuration precedence/reload,
  diagnostics compatibility, capability negotiation, Run/Stream parity, and
  diagnostics replay.
- Affected documentation: `README.md`, `docs/development-roadmap.md`,
  `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`,
  `docs/mainline-contract-test-index.md`, and the selected agent-mode README.
- No Provider SDK is introduced into `context/*`; no remote model registry,
  credential store, hosted control plane, or breaking public API is included.
