## 1. Runtime Concurrency Control Foundation

- [x] 1.1 Introduce unified concurrency config objects and documented defaults for runner/tool/mcp paths
- [x] 1.2 Implement queue capacity + backpressure modes with deterministic overload behavior
- [x] 1.3 Ensure cancellation propagation and bounded goroutine convergence for timeout/cancel branches
- [x] 1.4 Extend runtime events with queue/fanout/retry/cancel diagnostics

## 2. Async Communication Pipeline

- [x] 2.1 Implement channel-based async dispatch and result collection abstraction for tool and MCP tasks
- [x] 2.2 Add bounded retry handling with retryability-aware fail-fast termination
- [x] 2.3 Preserve run correlation fields (`run_id`, `iteration`, `call_id`, `trace_id`, `span_id`) across async events

## 3. Performance and Concurrency Safety Gates

- [x] 3.1 Add/upgrade benchmark suites for high-fanout, slow-call, and cancel-storm scenarios
- [x] 3.2 Define relative percentage threshold policy for benchmark regression decisions
- [x] 3.3 Integrate mandatory `go test -race ./...` and goroutine leak checks into CI quality gate

## 4. Tutorial Examples Expansion (Phased)

- [x] 4.1 Deliver R2 batch examples: `01-chat-minimal`, `02-tool-loop-basic`, `03-mcp-mixed-call`, `04-streaming-interrupt`
- [x] 4.2 Add TODO extension points for each example (known limits, optimization opportunities, next enhancements)
- [x] 4.3 Document advanced R3 batch plan: `05-parallel-tools-fanout`, `06-async-job-progress`, `07-multi-agent-async-channel`

## 5. Documentation and Roadmap Alignment

- [x] 5.1 Update `docs/development-roadmap.md` with phased concurrency/async execution plan and example milestones
- [x] 5.2 Update relevant docs with relative-percentage performance policy and concurrency safety baseline rules
- [x] 5.3 Ensure proposal/design/specs/tasks remain consistent with scope boundaries and non-goals
