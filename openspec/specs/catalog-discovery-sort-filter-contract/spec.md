# catalog-discovery-sort-filter-contract Specification

## Purpose

Define the catalog discovery query contract used by library browsing surfaces, including totals, paging, filters, and server-side sort direction.

## Requirements

### Requirement: Discovery browse response includes total and page metadata
The system SHALL return browse response metadata that includes the total matching result count and enough page metadata to determine whether additional results are available.

#### Scenario: Return page metadata
- **WHEN** a client requests a paged library browse result
- **THEN** the response includes `items`, `total`, and page or cursor metadata for continuing the browse

### Requirement: Discovery applies library browse filters
The system SHALL apply the filters accepted by the library browse UI, including library scope, query, type, genre, region, year, minimum rating, watched state, and sort field.

#### Scenario: Apply year and type filters
- **WHEN** a client requests a library browse result with a year filter and a type filter
- **THEN** every returned item matches the requested year and type within the requested library scope

### Requirement: Discovery applies server-side sort direction
The system SHALL accept and apply a safe sort direction value for supported sort fields instead of relying on client-side reversal.

#### Scenario: Sort titles descending
- **WHEN** a client requests title sorting with descending direction
- **THEN** the returned page is ordered by title descending using server-side ordering semantics

### Requirement: Discovery preserves backward-compatible defaults
The system SHALL preserve existing discovery behavior for clients that omit new paging or sort-direction parameters.

#### Scenario: Omit new parameters
- **WHEN** a client sends a discovery request without sort direction or paging metadata
- **THEN** the API returns a valid result using existing default ordering and limit behavior

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
