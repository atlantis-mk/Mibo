## ADDED Requirements

### Requirement: Fast classifier reuses directory plans
The fast classifier SHALL accept directory plan assignments as high-priority classification evidence and SHALL avoid recomputing full filename and sibling classification for files covered by high-confidence plans.

#### Scenario: File is covered by a high-confidence episode assignment
- **WHEN** a file has a high-confidence episode assignment from a directory content shape plan
- **THEN** the fast classifier SHALL return the planned episode candidate with plan evidence and SHALL NOT run independent movie-versus-episode candidate generation for that file

#### Scenario: File is not covered by a plan
- **WHEN** a file has no directory plan assignment or the assignment is below the configured confidence threshold
- **THEN** the fast classifier SHALL fall back to filename signals, bounded sibling evidence, and reviewable candidate generation

### Requirement: Fast classifier distinguishes shape-level evidence from file-level evidence
The fast classifier SHALL preserve whether classification evidence came from a directory shape profile, directory plan assignment, filename token profile, sidecar hint, or per-file fallback candidate.

#### Scenario: Planned episode candidate is emitted
- **WHEN** the classifier emits an episode candidate from a directory plan
- **THEN** the candidate evidence SHALL include the plan shape, plan confidence, numbering mode, directory path, and file assignment source

#### Scenario: Planned candidate conflicts with file evidence
- **WHEN** a high-confidence directory plan assignment conflicts with strong file-level evidence
- **THEN** the classifier SHALL preserve both alternatives and SHALL mark the decision provisional or review-required unless configured thresholds select one candidate safely

### Requirement: Fast classifier avoids repeated directory summary work
The fast classifier SHALL reuse directory shape profiles, directory plans, and filename token profiles when available and SHALL NOT recompute equivalent directory summaries per file or per materialization batch.

#### Scenario: Multiple files in one planned directory are materialized
- **WHEN** multiple files from the same planned directory are materialized across one or more batches
- **THEN** the classifier SHALL reuse the same profile and plan evidence for those files instead of rebuilding sibling summaries for each file
