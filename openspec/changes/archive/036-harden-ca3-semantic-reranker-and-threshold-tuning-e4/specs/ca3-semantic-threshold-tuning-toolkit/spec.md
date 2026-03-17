## ADDED Requirements

### Requirement: Threshold tuning toolkit SHALL produce reproducible recommendation artifacts
The CA3 threshold tuning toolkit MUST evaluate a supplied corpus and produce deterministic recommendation artifacts for the same input dataset and config.

The minimum outputs MUST include:
- operator-facing summary (`md`)
- recommended threshold range with confidence notes

#### Scenario: Deterministic tuning output
- **WHEN** toolkit runs multiple times with identical corpus, labels, and tuning config
- **THEN** generated threshold recommendation artifacts are semantically equivalent

#### Scenario: Missing required corpus metadata
- **WHEN** toolkit input lacks required fields for labels or sample identity
- **THEN** toolkit returns validation error and does not generate partial recommendation output

#### Scenario: Minimal markdown mode output
- **WHEN** toolkit run succeeds in minimal mode
- **THEN** toolkit emits markdown report as the required artifact without requiring JSON output

### Requirement: Threshold tuning toolkit SHALL support provider/model segmented analysis
Toolkit evaluation MUST support provider/model segmented score distribution analysis so recommendation output can target provider/model-specific profiles.

#### Scenario: Segmented analysis enabled
- **WHEN** corpus samples include provider/model attribution
- **THEN** toolkit emits per-provider/model recommendation sections in addition to global summary

#### Scenario: Segmentation data unavailable
- **WHEN** corpus does not include provider/model attribution
- **THEN** toolkit emits global recommendation only and marks segmented analysis as unavailable

### Requirement: Threshold tuning toolkit SHALL expose quality metrics and acceptance gates
Toolkit output MUST include normalized quality metrics required for operator review, including precision/recall-style trade-off indicators and threshold sweep snapshots.

#### Scenario: Operator reviews recommendation quality
- **WHEN** toolkit completes a threshold sweep
- **THEN** output includes metric table sufficient to justify selected recommendation range

#### Scenario: Recommendation fails acceptance gate
- **WHEN** best candidate threshold does not meet configured minimum quality gate
- **THEN** toolkit marks recommendation as non-accepting and emits explicit reason code

### Requirement: Threshold tuning toolkit SHALL report corpus readiness guidance
Toolkit MUST report corpus readiness guidance and confidence notes in output so operators can judge recommendation reliability.

#### Scenario: Corpus appears limited
- **WHEN** provider+model corpus size or holdout size is low for stable threshold estimation
- **THEN** toolkit emits warning-level guidance and reduced-confidence notes without hard-blocking recommendation output

#### Scenario: Corpus appears sufficient
- **WHEN** provider+model corpus size and holdout size support stable threshold estimation
- **THEN** toolkit emits normal-confidence recommendation notes
