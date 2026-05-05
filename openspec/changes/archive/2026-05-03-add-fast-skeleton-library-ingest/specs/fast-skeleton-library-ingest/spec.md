## ADDED Requirements

### Requirement: Scanner publishes skeleton entries for discovered videos
The system SHALL make newly discovered supported video files visible in library browsing from stable storage facts before final catalog classification, media probing, remote metadata matching, artwork processing, or full enrichment completes.

#### Scenario: Newly discovered video becomes visible before enrichment
- **WHEN** a scan discovers a supported video file that passes scan policy and exclusion filters
- **THEN** the system MUST persist or refresh the file's inventory facts
- **AND** the file MUST be eligible for a user-visible discovered media entry before ffprobe, metadata matching, artwork selection, and final movie or episode classification complete

#### Scenario: Scanner encounters uncertain classification
- **WHEN** the scanner cannot confidently classify a discovered video as a final movie, episode, version, or attachment
- **THEN** the system MUST still preserve the inventory-backed discovered entry
- **AND** it MUST mark the entry as requiring classification or review rather than blocking visibility

### Requirement: Skeleton visibility is anchored to inventory files
The system SHALL use the discovered `inventory_file` or an equivalent file-stable identity as the durable anchor for fast-ingest entries until final catalog graph links are available.

#### Scenario: File is later classified into an episode
- **WHEN** a discovered file is later classified into a series, season, and episode hierarchy
- **THEN** the final catalog graph MUST link back to the same file anchor
- **AND** the user-visible entry MUST upgrade without requiring the physical file record to be recreated

#### Scenario: Classification changes after rescan
- **WHEN** a later scan or classifier update changes the semantic interpretation of a discovered file
- **THEN** the system MUST preserve the file anchor and reconcile catalog links from that anchor
- **AND** it MUST NOT rely on a temporary display title as the durable identity

### Requirement: Discovered media exposes maturity state
The system SHALL expose an explicit maturity state for fast-ingest media so clients can distinguish discovered, classified, enriched, and review-required states.

#### Scenario: Client receives a discovered media card
- **WHEN** a browse API returns a media entry backed by a discovered file that has not completed final classification
- **THEN** the response MUST include a maturity or organizing state equivalent to `discovered`
- **AND** the response MUST include enough title/path-derived display information for the client to render a media card

#### Scenario: Classification requires review
- **WHEN** classifier evidence is conflicting or below the review threshold
- **THEN** the response MUST expose a review-required maturity state or equivalent flag
- **AND** the entry MUST remain visible unless excluded or missing

### Requirement: Discovered entries upgrade without duplicate browse results
The system SHALL avoid showing both a discovered file entry and the final catalog item for the same available file in the same library browse scope once catalog linkage has been established.

#### Scenario: Catalog link is created for discovered file
- **WHEN** asynchronous classification creates or reuses a catalog item and links it to the discovered file's asset
- **THEN** library browse results MUST show the catalog-backed media entry instead of a separate discovered-file duplicate for the same file

#### Scenario: Multiple discovered files become one movie
- **WHEN** classifier reconciliation groups multiple discovered files as versions or assets of one movie
- **THEN** library browse results MUST converge to one movie entry with the related assets
- **AND** the previous file-backed discovered entries MUST NOT remain as separate movie cards in that scope

### Requirement: Fast ingest avoids expensive enrichment in its critical path
The system SHALL keep skeleton ingest independent from expensive media enrichment work.

#### Scenario: Metadata provider is slow or unavailable
- **WHEN** remote metadata matching cannot complete during or after scan
- **THEN** discovered entries MUST remain visible with their current maturity state
- **AND** scan completion MUST NOT depend on the metadata provider result

#### Scenario: ffprobe backlog exists
- **WHEN** media probing is disabled, slow, or queued behind other work
- **THEN** discovered entries MUST remain visible without technical runtime or stream details
- **AND** probing MUST update the entry or linked asset asynchronously when it completes
