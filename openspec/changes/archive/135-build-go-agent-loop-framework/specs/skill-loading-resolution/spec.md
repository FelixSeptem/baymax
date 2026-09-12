## ADDED Requirements

### Requirement: Skill loader SHALL discover AGENTS and SKILL artifacts deterministically
The system MUST load workspace `AGENTS.md` first and MUST discover `SKILL.md` artifacts according to configured search rules.

#### Scenario: Ordered discovery
- **WHEN** run initialization starts
- **THEN** the loader MUST process AGENTS directives before evaluating skill files

#### Scenario: Missing skill file handling
- **WHEN** a referenced skill path is unreadable or missing
- **THEN** the loader MUST emit a skill-load failure event and continue initialization

### Requirement: Skill activation SHALL support explicit and semantic triggers
The system MUST support explicit skill mention and semantic relevance triggers, with explicit triggers taking precedence.

#### Scenario: Explicit trigger wins
- **WHEN** user input explicitly names a skill
- **THEN** the loader MUST enable that skill even if semantic ranking is low

#### Scenario: Semantic trigger fallback
- **WHEN** no explicit skill mention exists
- **THEN** the loader MAY enable semantically matched skills within configured confidence threshold

### Requirement: Instruction conflicts SHALL follow fixed precedence
When instruction conflicts occur, the system MUST resolve with precedence `system built-in > AGENTS > SKILL`.

#### Scenario: AGENTS vs SKILL conflict
- **WHEN** AGENTS and SKILL provide conflicting directives
- **THEN** the loader MUST keep AGENTS directive and record conflict metadata

#### Scenario: System override conflict
- **WHEN** built-in safety constraints conflict with AGENTS or SKILL directives
- **THEN** built-in constraints MUST prevail and lower-priority directives MUST be ignored

### Requirement: Skill output SHALL compile into runtime bundle
The loader MUST compile active skills into a runtime bundle containing prompt fragments, enabled tools, and workflow hints.

#### Scenario: Successful compile
- **WHEN** one or more skills are activated
- **THEN** the loader MUST provide a `SkillBundle` to Runner before first model step

#### Scenario: Partial compile failure
- **WHEN** one skill fails compilation but others succeed
- **THEN** the loader MUST continue with remaining skills and report warning event
