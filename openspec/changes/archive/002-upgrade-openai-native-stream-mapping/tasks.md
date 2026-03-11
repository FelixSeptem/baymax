## 1. OpenAI Native Streaming Adapter

- [x] 1.1 Map OpenAI Responses native streaming events to `types.ModelEvent` with backward-compatible type extensions
- [x] 1.2 Implement complete-tool-call-only emission with internal buffering for partial fragments
- [x] 1.3 Remove compatibility-only stream fallback from `model/openai` and keep native SDK stream as primary path

## 2. Runner Stream Semantics

- [x] 2.1 Align `core/runner.Stream` lifecycle and termination behavior with fail-fast policy
- [x] 2.2 Ensure stream error classification returns `ErrModel` or `ErrPolicyTimeout` consistently by failure context
- [x] 2.3 Verify stream lifecycle events keep correlation fields (`run_id`, `iteration`, `trace_id`, `span_id`) across all branches

## 3. Regression and Golden Tests

- [x] 3.1 Add adapter tests for event mapping, complete tool call emission, and error branches
- [x] 3.2 Add integration tests for event ordering, fail-fast termination, and semantic consistency between `Run` and `Stream`
- [x] 3.3 Add and maintain golden fixtures for canonical stream event sequences

## 4. Lint Quality Gate

- [x] 4.1 Add repository `.golangci.yml` with recommended linter set and stable defaults for this project
- [x] 4.2 Integrate `golangci-lint` command into CI validation flow with failing exit behavior
- [x] 4.3 Document local and CI lint invocation steps in `docs/` and README

## 5. Documentation Alignment

- [x] 5.1 Update `docs/v1-acceptance.md` to reflect removal of compatibility-first stream limitation and new fail-fast semantics
- [x] 5.2 Update `docs/development-roadmap.md` progress markers for this change and remaining R1 items
- [x] 5.3 Update other relevant `docs/` pages and change artifacts so behavior, scope boundaries, and acceptance criteria stay consistent
