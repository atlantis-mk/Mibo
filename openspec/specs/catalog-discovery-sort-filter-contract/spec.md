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
