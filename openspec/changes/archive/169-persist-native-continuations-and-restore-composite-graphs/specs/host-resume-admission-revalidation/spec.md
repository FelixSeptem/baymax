## MODIFIED Requirements

### Requirement: Resume support SHALL distinguish live continuation from metadata recovery

The runtime MUST preserve same-Engine direct dynamic-action Run/Stream resume using their original modes. It SHALL additionally support new-Engine direct restore only through a configured compatible durable checkpoint profile containing actual source continuation state and an atomic source resume claim. Snapshot references, metadata-only checkpoint interfaces, shared Run identifiers and event replay MUST NOT imply durable runner continuation. Plan bridge resumed execution and composed child/team/workflow/mailbox continuations MUST retain their unsupported classifications in this profile. Resume MUST remain distinct from retry, new Run and event replay.

#### Scenario: Live Run and Stream resume
- **WHEN** equivalent active in-process checkpoints receive equivalent admitted decisions in their respective modes
- **THEN** source iteration, saved tool results, resolution and terminal semantics remain equivalent

#### Scenario: Process restarts with metadata only
- **WHEN** a new Engine receives a decision and previous checkpoint metadata without retained execution continuation
- **THEN** it reports unavailable or unknown without invoking provider/tool or synthesizing a new Run

#### Scenario: Plan trigger is labeled resumed
- **WHEN** a planbridge descriptor uses resumed delivery without a supported native bridge continuation path
- **THEN** the existing unsupported result remains and the label is not treated as recovery authorization

#### Scenario: Process restarts with a compatible durable source
- **WHEN** a new Engine loads the original direct checkpoint from a compatible durable store and current host revalidation and source claim succeed
- **THEN** it restores the same Run/session and saved continuation boundary without a new parent Run or completed-tool reexecution

## ADDED Requirements

### Requirement: Host revalidation SHALL precede durable resume claim

When host resume admission is enabled for a durable checkpoint, the verifier SHALL validate the current reference-only authority, revision, plan, scope, confirmation, and effect binding before the source records a resume claim. The verifier SHALL never load or reconstruct source continuation and SHALL run outside source locks; source identity and generation MUST be rechecked before claim commit.

#### Scenario: Revalidation denial preserves checkpoint
- **WHEN** current host authority or revision no longer matches the trusted binding
- **THEN** admission rejects without consuming the checkpoint, emitting resolution/terminal events, or invoking provider/tool/effect

#### Scenario: New Engine passes valid revalidation
- **WHEN** a reopened source checkpoint and current host facts match
- **THEN** exactly one durable claim is committed and continuation retains the original run/session identity
