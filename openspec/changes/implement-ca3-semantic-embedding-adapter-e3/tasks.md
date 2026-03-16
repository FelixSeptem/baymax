## 1. Config and Validation

- [ ] 1.1 Add CA3 embedding scorer runtime config fields (`enabled`, `provider`, `model`, `timeout_ms`, `rule_weight`, `embedding_weight`, optional independent embedding credentials)
- [ ] 1.2 Implement startup and hot-reload fail-fast validation for embedding scorer config (weights range/sum, required provider/model, timeout bounds, credential precedence)
- [ ] 1.3 Ensure default effective behavior remains rule-only scoring when embedding scorer is not enabled
- [ ] 1.4 Set default hybrid config to cosine metric, `rule_weight=0.7`, `embedding_weight=0.3`, and shared existing quality threshold

## 2. Embedding Adapter and Scoring Pipeline

- [ ] 2.1 Implement OpenAI embedding adapter and internal SPI wiring for CA3 scorer
- [ ] 2.2 Implement Gemini embedding adapter and internal SPI wiring for CA3 scorer
- [ ] 2.3 Implement Anthropic embedding adapter and internal SPI wiring for CA3 scorer
- [ ] 2.4 Integrate hybrid quality score computation (rule score + cosine similarity) with deterministic formula and bounded output
- [ ] 2.5 Add policy-aware failure handling: `best_effort` fallback to rule-only and `fail_fast` terminate on adapter failure
- [ ] 2.6 Keep Run/Stream mode selection and fallback semantics equivalent for identical inputs/config

## 3. Diagnostics and Event Contract

- [ ] 3.1 Add CA3 embedding scoring diagnostics fields (adapter status, similarity contribution, fallback reason)
- [ ] 3.2 Propagate new diagnostics fields through runner/event/diagnostics store pipeline
- [ ] 3.3 Add contract tests for diagnostics presence and semantics under success/fallback/fail-fast scenarios across OpenAI/Gemini/Anthropic adapters

## 4. Testing and Benchmark Gate

- [ ] 4.1 Add contract tests for hybrid scoring gate pass/fail behavior and deterministic tie conditions (cosine-only metric)
- [ ] 4.2 Add Run/Stream equivalence tests for embedding-enabled success and fallback paths across three providers
- [ ] 4.3 Add/extend CA3 semantic benchmark cases for embedding enabled vs rule-only baseline and enforce relative regression policy
- [ ] 4.4 Execute and pass `go test ./...`
- [ ] 4.5 Execute and pass `go test -race ./...`
- [ ] 4.6 Execute and pass `golangci-lint run --config .golangci.yml`

## 5. Docs and Roadmap Sync

- [ ] 5.1 Update `README.md` with CA3 embedding scorer usage, defaults, and rollout guidance
- [ ] 5.2 Update `docs/runtime-config-diagnostics.md` for new config and diagnostics fields
- [ ] 5.3 Update `docs/context-assembler-phased-plan.md` with E3 scope and boundaries
- [ ] 5.4 Update `docs/v1-acceptance.md` and `docs/mainline-contract-test-index.md` for new contracts
- [ ] 5.5 Update `docs/development-roadmap.md` to reflect E3 completion and next TODO
