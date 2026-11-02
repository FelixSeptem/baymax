# Execution Graph Topology Projection

`orchestration/graph` provides a bounded, read-only canonical representation for comparing execution topology. It does not execute nodes, authorize effects, persist state, or restore checkpoints.

## Definition and compilation

A graph definition contains `GraphID`, `Version`, and nodes. Each node has an `ID` and optional `DependsOn` identifiers. `graph.Compile` trims identity and node references, sorts nodes and dependency identifiers, removes repeated dependency declarations, rejects invalid edges and cycles, then returns a deterministic topological order and `sha256:` topology digest.

The v1 limits are 128 UTF-8 bytes for graph and node identifiers, 32 bytes for version, 1,024 nodes, 8,192 unique dependency edges, and 1 MiB for canonical JSON. Compilation returns a typed `*graph.ValidationError`; invalid input returns no partial `graph.Compiled` value. Stable error codes distinguish missing identity/node/dependency, overlong identifiers, duplicate nodes, missing dependency targets, cycles, and each structural limit.

When several nodes are ready at once, the projection uses node identifier order as a deterministic tie-break. This ordering describes the canonical topology only; it is not a global execution or event order across independently running sources.

## Owner projections

- `(*workflow.Engine).ProjectExecutionGraph` validates and expands the definition using that engine's existing workflow capabilities, then projects workflow step dependencies. Workflow still owns conditions, retries, timeouts, step execution, continuation and terminal outcomes.
- `teams.ProjectExecutionGraph` maps `serial` task input order to a dependency chain; `parallel` and `vote` map to independent nodes. Teams still owns dispatch, voting, aggregation, failure policy and continuation.
- `composer.ProjectChildExecutionGraph` maps one child dispatch to one node. It does not infer siblings or claim a complete parent fanout graph. Composer/scheduler still own child attempts and terminal commits.

Equivalent graph IDs, versions, node IDs and edges yield equal topology identities across supported projections. A child-only projection cannot compare equal to a multi-node parent graph unless the topology itself is a single node.

## Digest and restore boundary

`Identity.TopologyDigest` covers only graph ID, version, node IDs and dependency edges. Task payloads, retry/timeout policy, strategy policy, authority, write scope, effect namespace and callbacks are deliberately excluded. A matching topology digest is comparison evidence, not authorization, plan integrity, or restore eligibility.

Keep each owner's existing plan and checkpoint digest for execution-state binding. In particular, do not copy `TopologyDigest` into workflow/team/child continuation requests or source bindings. The new projection API is additive and does not rewrite persisted checkpoints.

## Configuration and diagnostics

This contract adds no environment/file configuration, hot-update keys, QueryRuns fields, RuntimeRecorder payload fields or diagnostic events. Projection errors are returned to the caller as typed API errors; diagnostic write ownership remains with the existing owner and `observability/event.RuntimeRecorder`.

## Offline example

Run `go test ./examples/execution-graph -run TestOfflineExampleEmitsContractMarkers -count=1` to verify workflow/team serial and parallel parity, single-child scope, and digest separation. The example only compiles projections; it does not run a workflow, team, child, scheduler, model or tool.

Rollback requires only stopping calls to the additive projection APIs. No configuration or persistence migration is required.
