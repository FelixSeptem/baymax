# Legacy TODO Baseline

Scan command:

```bash
rg -n "TODO|TBD|FIXME|待补" examples -g "*.md" -g "*.go" -g "*.yaml" -g "*.yml" -g "*.json"
```

Snapshot date: `2026-04-09`

Result:

- No unresolved `TODO/TBD/FIXME/待补` markers found under `examples/`.

Tracking rule:

- Any future deferred item in examples must be tracked in:
- `examples/agent-modes/MATRIX.md`
- `examples/agent-modes/PLAYBOOK.md`
- `openspec/changes/<agent-mode-example-pack-change>/tasks.md`
