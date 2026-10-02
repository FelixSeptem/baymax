## 1. Capability adapter contract

- [x] 1.1 Add failing unit tests for explicit adapter construction, normalized/deduplicated capabilities, invalid configuration, discovery reports, and Generate/Stream error delegation; verified the new package tests failed before implementation.
- [x] 1.2 Implement `adapter/modelcapability` with validated config, transparent model delegation, `ProviderName`, and deterministic `DiscoverCapabilities`; verified `go test ./adapter/modelcapability -count=1` passes.

## 2. Runner diagnostics and integration

- [x] 2.1 Add failing runner tests for a strict Stream request against a minimal client without `ModelCapabilityDiscovery`, asserting the missing interface details, and for an explicit adapter that passes streaming preflight; verified the tests failed before runner changes.
- [x] 2.2 Update runner capability selection diagnostics to distinguish unavailable discovery from discovered capability mismatch while preserving classified error and fallback behavior; verified focused and full `core/runner` tests pass.
- [x] 2.3 Add an integration regression covering equivalent Run/Stream capability declaration and no provider invocation when discovery is absent; verified focused integration tests pass.

## 3. Documentation and examples

- [x] 3.1 Update the model adapter template to wrap the minimal model with the explicit capability adapter and declare `streaming`; verified `go run ./examples/templates/model-adapter-template` emits the expected stream answer.
- [x] 3.2 Update adapter and external-template documentation with the optional discovery interface, adapter usage, error guidance, and Example Impact Assessment; verified docs consistency passes.

## 4. Contract validation

- Example Impact Assessment: 修改示例。

- [x] 4.1 Run OpenSpec validation and confirm proposal/design/spec/tasks include the legal Example Impact Assessment value; `openspec validate --all` passed.
- [x] 4.2 Run affected package tests, `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `scripts/check-quality-gate.ps1`, and `scripts/check-docs-consistency.ps1`; implementation verification is complete, the isolated agent-mode smoke stability rerun passed, and this task is now closed before archival.
