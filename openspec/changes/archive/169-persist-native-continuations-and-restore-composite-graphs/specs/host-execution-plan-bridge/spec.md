## ADDED Requirements

### Requirement: Direct checkpoint restore SHALL NOT imply resumed plan support

A direct runner durable checkpoint capability SHALL NOT enable `host_execution_plan.v1` resumed-trigger dispatch or composite continuation by itself. The bridge MUST require a separately declared compatible bridge binding and source-native restore capability. In this profile, resumed plans and composite pause requirements SHALL reject as `unsupported_capability` before source activation, even when a direct Engine supports restore.

#### Scenario: Direct store is configured but plan uses resumed trigger
- **WHEN** a host compiles a resumed-trigger plan against an Engine with a durable direct checkpoint store
- **THEN** compile rejects as unsupported before model/tool/handler/enqueue/publish/claim and does not reinterpret it as a fresh direct Run

#### Scenario: Composite plan declares a pause requirement
- **WHEN** a child, team, workflow or mailbox plan requires any unverified pause/restore cell
- **THEN** admission rejects the entire plan before any member or step starts
