# Tool schema pressure and selection audit

`tool/schemaaudit` owns the offline `tool_schema_pressure_selection_audit.v1`
contract. It accepts a bounded snapshot of already-admitted tools and computes
canonical schema facts, provider-neutral pressure metrics, synthetic gold-set
quality, and deterministic comparisons for the five versioned strategies.

The package is audit-only: it does not call tools or models, use provider SDKs
or tokenizers, access network/filesystem/clock/credentials, mutate registries,
or wire a subset into `ModelRequest`. Raw schemas are accepted only in-memory
by `CanonicalSchemaFacts`; normalized fixtures retain digest, byte count,
estimate version, bounded labels, metrics, and reason codes.

Synthetic cases are mandatory evidence. Eval corpus/Badcase metadata is an
optional advisory and cannot change the synthetic conclusion. A
`projection_candidate` is only offline evidence requiring both pressure and
quality gaps plus a deterministic improvement with no forbidden hit.

Example Impact Assessment: 无需示例变更（附理由）。The package does not
change `examples/agent-modes` runtime paths, configuration, or markers.
