## 1. Multi-Agent Benchmark Matrix

- [ ] 1.1 Add integration benchmarks for synchronous invocation mainline path.
- [ ] 1.2 Add integration benchmarks for async reporting mainline path.
- [ ] 1.3 Add integration benchmarks for delayed dispatch mainline path.
- [ ] 1.4 Add integration benchmarks for recovery replay mainline path.
- [ ] 1.5 Ensure benchmark outputs include `ns/op`, `p95-ns/op`, and `allocs/op` for each required path.

## 2. Baseline and Regression Scripts

- [ ] 2.1 Add `scripts/multi-agent-benchmark-baseline.env` with required baseline and threshold keys.
- [ ] 2.2 Add `scripts/check-multi-agent-performance-regression.sh` with default `benchtime=200ms` and `count=5`.
- [ ] 2.3 Add `scripts/check-multi-agent-performance-regression.ps1` with parity to shell script semantics.
- [ ] 2.4 Implement fail-fast validation for missing baseline, invalid thresholds, and parse failures in both scripts.
- [ ] 2.5 Implement relative degradation checks for `ns/op`, `p95-ns/op`, and `allocs/op` in both scripts.

## 3. Quality Gate and CI Integration

- [ ] 3.1 Integrate multi-agent performance regression script into `scripts/check-quality-gate.sh`.
- [ ] 3.2 Integrate multi-agent performance regression script into `scripts/check-quality-gate.ps1`.
- [ ] 3.3 Ensure default CI quality path preserves the same blocking semantics as local quality-gate scripts.

## 4. Documentation and Traceability

- [ ] 4.1 Update `docs/performance-policy.md` with multi-agent benchmark matrix and threshold defaults.
- [ ] 4.2 Update `docs/mainline-contract-test-index.md` with benchmark/gate traceability rows for A19.
- [ ] 4.3 Update `docs/development-roadmap.md` with A19 proposal and status.
- [ ] 4.4 Update `README.md` performance section with multi-agent regression gate entry.

## 5. Validation

- [ ] 5.1 Run `go test ./integration -run '^$' -bench '^BenchmarkMultiAgent' -benchmem -benchtime=200ms -count=5`.
- [ ] 5.2 Run `bash scripts/check-multi-agent-performance-regression.sh`.
- [ ] 5.3 Run `pwsh -File scripts/check-multi-agent-performance-regression.ps1`.
- [ ] 5.4 Run `bash scripts/check-quality-gate.sh` and `pwsh -File scripts/check-quality-gate.ps1`.
- [ ] 5.5 Run `go test ./...`.
- [ ] 5.6 Run `go test -race ./...`.
- [ ] 5.7 Run `golangci-lint run --config .golangci.yml`.
- [ ] 5.8 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [ ] 5.9 Run `openspec validate introduce-multi-agent-mainline-performance-baseline-gate-a19 --strict`.

