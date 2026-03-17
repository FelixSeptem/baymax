## 1. Runtime Config and Validation

- [x] 1.1 Add CA2 external capability-hint config fields with deterministic precedence (`env > file > default`)
- [x] 1.2 Add CA2 template-pack config fields and profile enum support (`graphrag_like|ragflow_like|elasticsearch_like`)
- [x] 1.3 Implement deterministic template resolution precedence (`profile defaults -> explicit overrides`) and explicit-only acceptance path
- [x] 1.4 Add startup/hot-reload fail-fast validation for invalid template profile and malformed hint schema

## 2. Stage2 SPI and Assembler Integration

- [x] 2.1 Extend Stage2 retriever SPI request contract with optional capability-hint extension payload
- [x] 2.2 Wire assembler Stage2 path to forward hints through SPI without introducing provider-specific routing branches
- [x] 2.3 Implement observational-only hint mismatch handling (no automatic provider switch or route mutation)
- [x] 2.4 Ensure existing `fail_fast/best_effort` stage policy semantics remain unchanged

## 3. Diagnostics and Event Contract

- [x] 3.1 Add additive diagnostics fields for template resolution and hint outcomes (`stage2_template_profile`, `stage2_template_resolution_source`, `stage2_hint_applied`, `stage2_hint_mismatch_reason`)
- [x] 3.2 Preserve existing Stage2 layered error semantics (`transport|protocol|semantic`) while supporting additive extension fields
- [x] 3.3 Propagate new fields through diagnostics API and event mapping paths with backward compatibility guarantees

## 4. Tests and Performance Baseline

- [x] 4.1 Add contract tests for template precedence and explicit-only mode semantics
- [x] 4.2 Add contract tests for observational-only hint mismatch behavior (no automatic action)
- [x] 4.3 Add contract tests for Run/Stream semantic equivalence on hint/template success and mismatch paths
- [x] 4.4 Add benchmark baseline for CA2 hint/template resolution overhead and keep existing CA2 trend baseline comparable
- [x] 4.5 Execute and pass `go test ./...`
- [x] 4.6 Execute and pass `go test -race ./...`
- [x] 4.7 Execute and pass `golangci-lint run --config .golangci.yml`

## 5. Documentation Sync

- [x] 5.1 Update `docs/runtime-config-diagnostics.md` with hint/template config and diagnostics field index
- [x] 5.2 Update `docs/ca2-external-retriever-evolution.md` with E3 scope and boundary decisions
- [x] 5.3 Update `docs/development-roadmap.md` and `docs/v1-acceptance.md` for E3 milestones and contract gates
- [x] 5.4 Add minimal YAML integration samples for `graphrag_like`, `ragflow_like`, and `elasticsearch_like` templates (no runnable example code)
