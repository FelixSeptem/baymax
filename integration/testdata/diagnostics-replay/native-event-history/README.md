# Native event history reference replay

`v1/conformance.json` is an offline, reference-only transcript with schema
`native_event_history.v1`. It contains no message bodies, credentials, model or
tool inputs, raw callback failures, host business-commit assertions or runtime
configuration. The evaluator accepts only public source references, bounded
history identities, declared capability facts and recorded page/cursor evidence.

The matrix records durable mailbox history from a compatible retained backend.
Runner, child, team and workflow remain explicitly unsupported. It is evidence
for the source capability audit, not authority to upgrade any concrete source.
Actual FileStore restart, retention, lease and terminal behavior is exercised by
the mailbox and integration suites in the same contract gate.

Cursor values in this fixture are opaque reference labels. `cursor_source`,
`after`, `floor` and `high_water` are separately recorded owner evidence. The
offline evaluator checks their consistency; it does not decode a source cursor,
authenticate an unsigned transcript, instantiate a source reader, or infer a
global order from separate sources. A restart references the confirmed cursor
and same source. Malformed, foreign, expired, gapped, unsupported and unavailable
evidence must retain its explicit classification and must never contain a page
from latest as a substitute.

The retained `mailbox.ack` terminal has outcome `acked`; this establishes mailbox
lifecycle completion only. It does not establish model execution completion,
checkpoint restore, a resumed graph or a host database commit.

Run the equivalent gates:

```powershell
pwsh -File scripts/check-native-event-history-contract.ps1
```

```bash
bash scripts/check-native-event-history-contract.sh
```

Both invoke the same source and pure replay tests, then reject execution/store
dependencies or workers in the pure evaluator. Negative mutation tests detect
identity, sequence, cursor, retention, capability, terminal and side-effect drift.
Rollback removes history exposure or marks its capability unavailable; fixture
replay itself never writes source state or starts a recovery path.
