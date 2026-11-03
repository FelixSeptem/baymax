# Attention diagnostics

Run `go run ./examples/attention-diagnostics` offline. All IDs are synthetic;
there are no credentials, network calls, stores, retries or effect execution.

The example projects a supplied diagnostics snapshot plus explicit pending-effect
and source references. Healthy and automatically retrying work produces no item.
A limit of two makes truncation visible: `total=4 shown=2 truncated=true` counts
only this input snapshot. The two visible items retain their action and owner.

Expected markers: `healthy_retrying_ignored`, `effect_pending_explicit`,
`truncation_visible`. See [the contract guide](../../docs/guides/attention-diagnostics.md)
for all mappings, bounds and owner refresh requirements. Remove the projection
call to roll back; this example never changes owner state.
