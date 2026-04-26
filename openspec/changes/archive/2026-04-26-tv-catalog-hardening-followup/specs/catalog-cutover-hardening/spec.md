## ADDED Requirements

### Requirement: Catalog cutover enforces production-grade relational safety
The system SHALL enforce the missing database-level safety constraints, uniqueness guarantees, and lookup indexes needed for stable catalog graph, asset-link, and descendant identity behavior in production.

#### Scenario: Startup encounters duplicate-prone catalog paths
- **WHEN** the service starts or migrates against a database that contains catalog graph or descendant identity data
- **THEN** the schema and migration flow MUST enforce the required relational safety guarantees needed to prevent silent duplicate or orphan-prone catalog behavior

### Requirement: Consistency and rebuild coverage is complete before default-read cutover
The system SHALL provide rebuild and consistency-check coverage for catalog rollups, availability, search documents, and related derived data before catalog reads are treated as the stable default.

#### Scenario: Operator validates catalog derived data before enabling default reads
- **WHEN** an operator prepares to rely on catalog-backed reads by default
- **THEN** the system MUST provide rebuild and verification operations that can detect and repair unsafe projection or availability drift

### Requirement: Catalog reads become default only after combined validation gates pass
The system SHALL enable catalog reads by default only after backend, frontend, playback, hierarchy, and governance validation has succeeded together.

#### Scenario: Validation is incomplete during cutover
- **WHEN** combined validation for hierarchy, playback, governance, or frontend catalog flows has not yet succeeded
- **THEN** the system MUST treat catalog-read-default cutover as incomplete and preserve the bounded migration behavior defined for the current phase

### Requirement: Remaining legacy media routes retire explicitly after validation
The system SHALL remove, retire, or isolate remaining legacy media read and write paths only after cutover validation succeeds, and SHALL keep any recovery behavior explicit and bounded.

#### Scenario: Client calls a retired legacy media endpoint after final cutover
- **WHEN** a client calls a legacy media endpoint after final cutover validation has passed
- **THEN** the server MUST respond with the explicitly defined retirement or bounded fallback behavior instead of silently serving stale legacy-path data
