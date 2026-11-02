# Execution Graph Topology Projection

This offline example demonstrates the public `execution_graph.v1` topology projection contract.

## Semantic anchor

Equivalent workflow/team topology and a single workflow/child node produce the same topology identity. The topology digest covers graph identity, node IDs, and dependency edges only; it is not an owner plan digest, authorization, checkpoint digest, or restore token.

## Runtime path

`main.go` calls `workflow.Engine.ProjectExecutionGraph`, `teams.ProjectExecutionGraph`, and `composer.ProjectChildExecutionGraph`. Those APIs only normalize and compile topology; the example does not call Run/Stream, dispatch a child, start a scheduler, invoke a tool/model, write diagnostics, or restore a checkpoint.

## Expected markers

Run `go test ./examples/execution-graph -run TestOfflineExampleEmitsContractMarkers -count=1`. It verifies:

- `execution_graph.workflow_team_serial_equal=true`
- `execution_graph.workflow_team_parallel_equal=true`
- `execution_graph.workflow_child_single_node_equal=true`
- `execution_graph.owner_digest_unchanged=true`
- `execution_graph.no_execution_performed=true`

## Rollback notes

The projection API is additive and has no persisted state or configuration. Rollback means stopping use of the projection calls; existing owner digests and checkpoint records remain unchanged.
