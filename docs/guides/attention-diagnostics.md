# Attention diagnostics projection

Owner: `runtime/diagnostics` maintainers. `ProjectAttention` is a pure read model
over a caller-selected page of `RunRecord` values, normally from `QueryRuns`.
It does not query, subscribe, write, resume, retry or acknowledge anything.
RuntimeRecorder remains the only diagnostic writer. QueryRuns filters, ordering,
cursor and page defaults are unchanged.

Pass `AttentionRequest{Runs, Evidence, Limit}`. Each input list is limited to
200 entries; zero Limit means 50 and explicit Limit must be 1..200.
RunIDs must be unique within Runs. IDs/references are 1..256 ASCII bytes using
letters, digits, dot, underscore, colon, slash, at-sign or hyphen. They must be
opaque non-secret references; never include raw text, tokens, paths with private
content, tool arguments or credentials. Invalid input returns
`ErrAttentionInput` with no partial result or echoed raw input.

| Evidence | Code | Next action | Owner |
| --- | --- | --- | --- |
| ProtocolCapabilityReason exactly protocol.capability.missing_required, or ProviderCapabilityOutcome exactly blocked | unsupported | inspect_capability | runtime/capability |
| RecoveryConflict or StreamRecoveryTerminalConflictRecorded | recovery-required | inspect_recovery | runtime/recovery |
| GateDeniedCount > 0 or GateTimeoutCount > 0 | gate-failed | inspect_gate | runtime/gate |
| explicit host authorization evidence | authorization-required | revalidate_authorization | host/authorization |
| explicit source owner evidence | source-required | inspect_source | host/source |
| explicit host effect evidence | effect-pending | reconcile_effect | host/effect |

Run-derived items reference RunID. Supplemental `AttentionEvidence` accepts only
the last three codes, with RunID and Reference; its owner/action are fixed by the
code and cannot be supplied by callers. RunID may refer to a run outside the
selected page. The caller must obtain this evidence from the relevant owner.
The projection neither authenticates the caller nor replaces a fresh owner read.
Do not infer effect commit, source progress or authorization from terminal status,
callback acknowledgement, automatic retry, absent fields, or an empty result.
Other classifications are unknown to this version and produce no inferred item.
Provider `blocked` is a coarse capability-unavailable signal and also includes
an unknown catalog model; it does not prove a particular required capability is
missing. Credential-only rejection with accepted capability, provider `degraded`
and protocol `protocol.capability.optional_downgraded` produce no unsupported item.

Items are deduplicated by RunID/code/reference and sorted by that tuple using
bytewise string order before applying Limit. `Total` counts unique candidate
items in this supplied snapshot; `Truncated` means Total exceeds returned Items.
It is not a global backlog count and does not describe further QueryRuns pages.
No opaque continuation cursor is created by this projection.

The standalone example uses synthetic references and no external I/O.
Replay validates a bounded JSON fixture against this same pure function,
rejects duplicate, case-alias and non-ASCII object keys before typed decoding and
never instantiates a runner, source or consumer. Drift classification:
`attention.replay_drift`. Verification:
`pwsh -File scripts/check-attention-diagnostics-contract.ps1` and the shell twin.
Rollback: stop calling the projection; inspect original owner facts directly.
