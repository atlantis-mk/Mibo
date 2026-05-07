## MODIFIED Requirements

### Requirement: Metadata operations execute through a unified pipeline
The system SHALL execute automated match, refetch, manual candidate apply, and local evidence apply through a shared metadata operation pipeline that resolves target item, library strategy, execution plan, provider attempts, metadata decision, field application, and projection refresh as one operation. Automated scanner-triggered matching SHALL be queued per recognized movie or series work group rather than per source file or per version asset.

#### Scenario: Automated match uses unified pipeline
- **WHEN** a queued catalog item match job runs for a pending catalog item
- **THEN** the system MUST execute a metadata operation of type `match` and return or persist a result that includes the target item, execution plan summary, provider attempts, selected candidate when present, applied fields, skipped fields, resulting governance status, and affected catalog item IDs

#### Scenario: Refetch uses unified pipeline
- **WHEN** a user refetches metadata for an item that already has provider identity
- **THEN** the system MUST execute a metadata operation of type `refetch` using the same execution plan, provider attempt, field application, and projection refresh semantics as automated matching

#### Scenario: Manual candidate apply uses unified pipeline
- **WHEN** a user applies a selected metadata candidate to a catalog item
- **THEN** the system MUST execute a metadata operation of type `manual_apply` and MUST record manual operation status, provider provenance, applied fields, skipped locked fields, and affected catalog item IDs

#### Scenario: Movie version work group is scanned
- **WHEN** the scanner materializes one movie item with multiple version assets
- **THEN** the metadata pipeline SHALL queue at most one automated match for the movie work group and SHALL NOT queue separate remote searches for each version file

#### Scenario: Series work group is scanned
- **WHEN** the scanner materializes a series with season or episode descendants
- **THEN** the metadata pipeline SHALL queue matching for the series root and SHALL NOT match each episode independently in the fast path
