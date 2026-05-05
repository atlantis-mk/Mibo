## ADDED Requirements

### Requirement: Discovery includes inventory-backed organizing entries
The system SHALL include inventory-backed discovered media entries in library browse responses when those entries are in scope and do not yet have a final catalog-backed browse result.

#### Scenario: Browse library with newly discovered file
- **WHEN** a client requests a library browse result after a scan has discovered a supported video file but before final catalog projection completes
- **THEN** the response MUST include a media entry for the discovered file if it matches the requested library scope and filters
- **AND** the entry MUST expose organizing maturity state

#### Scenario: Discovered entry is linked to catalog item
- **WHEN** a discovered file already contributes to a catalog-backed result in the same browse scope
- **THEN** the discovery response MUST suppress the duplicate inventory-backed organizing entry for that file

### Requirement: Discovery sorting and paging handle organizing entries
The system SHALL apply safe sorting, paging, and total-count semantics consistently when browse results contain both catalog-backed entries and inventory-backed organizing entries.

#### Scenario: Browse mixed mature and organizing results
- **WHEN** a client requests a paged library browse result containing catalog-backed and organizing entries
- **THEN** the response total MUST include all entries that match the request
- **AND** paging MUST NOT duplicate or skip entries because of mixed backing sources

#### Scenario: Sort organizing entries by title
- **WHEN** a client requests title sorting for a result set that includes organizing entries
- **THEN** organizing entries MUST sort using their display title or filename-derived title under the same direction semantics as catalog-backed entries
