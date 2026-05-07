## ADDED Requirements

### Requirement: Library projection rows
The system SHALL maintain projection rows keyed by library and metadata identity for library-scoped browsing and availability.

#### Scenario: Resource link creates projection
- **WHEN** a resource in a library links to a metadata identity
- **THEN** the system creates or updates a library projection row for that library and metadata identity

#### Scenario: Same metadata appears in two libraries
- **WHEN** resources in two libraries link to the same metadata identity
- **THEN** the system stores separate projection rows for each library and the same metadata identity

### Requirement: Projection availability
The system SHALL compute availability for a metadata identity within a library from resources visible in that library.

#### Scenario: Available projected item
- **WHEN** a library has at least one available playable resource linked to a metadata identity
- **THEN** that library projection reports the metadata identity as available

#### Scenario: Missing projected item
- **WHEN** a library projection exists but no linked playable resource in that library is available
- **THEN** that library projection reports missing or unavailable according to projection policy

### Requirement: Projection rollups
The system SHALL compute series and season rollups from child projections within the same library.

#### Scenario: Series has available episode
- **WHEN** one episode projection under a series is available in a library
- **THEN** the series projection for that library reports available child counts and available status according to rollup policy

### Requirement: Library browsing reads projections
The system SHALL serve library browsing from library projection rows joined to metadata display data.

#### Scenario: Browse library items
- **WHEN** a client requests items for a library
- **THEN** the response contains metadata identities that have projection rows for that library

#### Scenario: Exclude metadata without library resources
- **WHEN** a global metadata identity has no projection row for the requested library
- **THEN** the library browse response does not include that identity

### Requirement: Projection rebuilds
The system SHALL rebuild affected library projections when resource links, metadata links, metadata fields, or resource status change.

#### Scenario: Resource removed from library
- **WHEN** a resource library membership is deleted or marked missing
- **THEN** affected projection rows are recomputed for that library

#### Scenario: Metadata title updated
- **WHEN** the canonical title of a metadata identity changes
- **THEN** all library projection/search rows for that metadata identity are refreshed

### Requirement: Library search documents
The system SHALL provide library-scoped search documents derived from projections and metadata fields.

#### Scenario: Library-scoped search
- **WHEN** a client searches within a library
- **THEN** the search only returns metadata identities projected into that library

### Requirement: Projection visibility controls
The system SHALL allow a metadata identity to be hidden from a specific library without deleting the global metadata identity or resources.

#### Scenario: Hide item from library
- **WHEN** a governance action hides a projected metadata identity in one library
- **THEN** that library no longer shows the projection while other libraries remain unaffected
