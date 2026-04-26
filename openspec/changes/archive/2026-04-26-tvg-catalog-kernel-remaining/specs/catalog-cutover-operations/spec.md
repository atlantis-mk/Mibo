## ADDED Requirements

### Requirement: Cutover state is tracked explicitly
The system SHALL record explicit migration and cutover state that indicates whether backfill has completed, whether catalog reads are enabled, and whether legacy cleanup has completed.

#### Scenario: Startup can determine migration mode
- **WHEN** the application starts against an empty, legacy-populated, or partially migrated database
- **THEN** the system MUST be able to determine from stored migration state whether it should remain in migration mode, allow catalog reads, or consider legacy cleanup complete

### Requirement: Consistency rebuilds and checks are available before cleanup
The system SHALL provide rebuild or verification operations for catalog rollups, search documents, availability, and related projections before legacy cleanup is considered complete.

#### Scenario: Operator validates projections before final cleanup
- **WHEN** an operator prepares to retire legacy paths after migration and read cutover
- **THEN** the system MUST provide a way to rebuild or verify catalog projections and report inconsistencies that would make cleanup unsafe

### Requirement: Legacy cleanup is gated on successful cutover validation
The system SHALL forbid final legacy path removal until backfill, catalog writes, catalog reads, and validation checks have completed successfully.

#### Scenario: Cleanup is blocked while legacy dependencies remain
- **WHEN** catalog reads are not yet enabled by default or verification shows remaining reliance on legacy tables or indexes
- **THEN** the system MUST treat legacy cleanup as incomplete and preserve the legacy compatibility path until the validation gate is satisfied

### Requirement: Cutover preserves startup and migration safety across database states
The system SHALL support repeated startup on empty databases, legacy databases, and already-cut-over databases without corrupting catalog state or requiring destructive resets.

#### Scenario: Repeated startup remains safe across migration phases
- **WHEN** the service starts multiple times across the phases of schema setup, backfill, read cutover, and final cleanup
- **THEN** the startup and migration flow MUST remain idempotent and MUST NOT require operators to manually reset the database to recover from a previously successful phase
