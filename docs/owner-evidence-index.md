# Owner Evidence Index

The canonical [versioned index](owner-evidence-index.v1.json) maps explicitly
enrolled public semantics to one owner, an OpenSpec main specification, tests,
replay fixture or justified exemption, dedicated gate, public documentation,
and rollback/non-goal boundaries. It currently enrolls resolver material
admission, provider-neutral model-surface evidence, and source-adapter lifecycle;
it does not claim to cover every repository contract.

Each semantic must be split at an actual ownership boundary. Update its source
map row in [documentation ownership](documentation-ownership.md) together with
the index. Run `pwsh -File scripts/check-owner-evidence-index-contract.ps1` or
`bash scripts/check-owner-evidence-index-contract.sh`; the standard quality
gates run the same checks. Active/archive status remains governed by OpenSpec,
the [roadmap](development-roadmap.md) and archive index, not this artifact.

This is static, offline evidence, not a runtime registry or authority to
execute, authorize, recover, write diagnostics, or commit an effect. Replay
evidence does not replace source-owner tests or prove a business commit. To
roll back, remove the index, validation and links; runtime owners and facts are
unchanged. Narrowing a runtime port requires separate evidence of two callers
crossing an owner boundary through the same interface.
