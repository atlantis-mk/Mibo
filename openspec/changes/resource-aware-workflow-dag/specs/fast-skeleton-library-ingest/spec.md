## MODIFIED Requirements

### Requirement: Fast ingest avoids expensive enrichment in its critical path
The system SHALL keep skeleton ingest independent from expensive media enrichment work and SHALL represent enrichment as workflow tasks that are dependent on, but not part of, the critical skeleton visibility path.

#### Scenario: Metadata provider is slow or unavailable
- **WHEN** remote metadata matching cannot complete during or after scan
- **THEN** discovered entries MUST remain visible with their current maturity state
- **AND** scan workflow core synchronization completion MUST NOT depend on the metadata provider result

#### Scenario: ffprobe backlog exists
- **WHEN** media probing is disabled, slow, resource-limited, or queued behind other work
- **THEN** discovered entries MUST remain visible without technical runtime or stream details
- **AND** probing MUST update the entry or linked asset asynchronously when its workflow task completes

### Requirement: Scanner publishes skeleton entries for discovered videos
The system SHALL make newly discovered supported video files visible in library browsing from stable storage facts before final catalog classification, media probing, remote metadata matching, artwork processing, or full enrichment completes. Workflow scheduling MUST NOT delay skeleton visibility solely because unrelated libraries have pending enrichment work.

#### Scenario: Newly discovered video becomes visible before enrichment
- **WHEN** a scan workflow discovers a supported video file that passes scan policy and exclusion filters
- **THEN** the system MUST persist or refresh the file's inventory facts
- **AND** the file MUST be eligible for a user-visible discovered media entry before ffprobe, metadata matching, artwork selection, and final movie or episode classification complete

#### Scenario: Scanner encounters uncertain classification
- **WHEN** the scanner cannot confidently classify a discovered video as a final movie, episode, version, or attachment
- **THEN** the system MUST still preserve the inventory-backed discovered entry
- **AND** it MUST mark the entry as requiring classification or review rather than blocking visibility
