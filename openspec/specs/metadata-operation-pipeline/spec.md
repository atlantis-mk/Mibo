# metadata-operation-pipeline Specification

## Purpose
Define the unified metadata operation pipeline used for automated matching, refetching, manual application, local evidence application, provider attempts, field ownership, and projection refresh.

## Requirements
### Requirement: Metadata operations execute through a unified pipeline
The system SHALL execute automated match, refetch, manual candidate apply, and local evidence apply through a shared metadata operation pipeline that resolves target item, library strategy, execution plan, provider attempts, metadata decision, field application, and projection refresh as one operation.

#### Scenario: Automated match uses unified pipeline
- **WHEN** a queued catalog item match job runs for a pending catalog item
- **THEN** the system MUST execute a metadata operation of type `match` and return or persist a result that includes the target item, execution plan summary, provider attempts, selected candidate when present, applied fields, skipped fields, resulting governance status, and affected catalog item IDs

#### Scenario: Refetch uses unified pipeline
- **WHEN** a user refetches metadata for an item that already has provider identity
- **THEN** the system MUST execute a metadata operation of type `refetch` using the same execution plan, provider attempt, field application, and projection refresh semantics as automated matching

#### Scenario: Manual candidate apply uses unified pipeline
- **WHEN** a user applies a selected metadata candidate to a catalog item
- **THEN** the system MUST execute a metadata operation of type `manual_apply` and MUST record manual operation status, provider provenance, applied fields, skipped locked fields, and affected catalog item IDs

### Requirement: Metadata operations persist or expose provider attempts
The system SHALL retain enough provider attempt evidence for each metadata operation to explain which configured providers were attempted, skipped, failed, or selected for each metadata stage.

#### Scenario: Primary provider unavailable
- **WHEN** the first configured search provider is disabled, unavailable, or in cooldown and a later provider supplies candidates
- **THEN** the operation evidence MUST record the first provider as skipped with its reason and the later provider as the selected successful attempt

#### Scenario: Provider returns no candidates
- **WHEN** a provider search request succeeds but returns no usable candidates
- **THEN** the operation evidence MUST record a `no_result` attempt instead of treating the operation as an infrastructure failure

#### Scenario: Provider request fails
- **WHEN** a provider request fails with authentication, rate limit, timeout, or remote error
- **THEN** the operation evidence MUST record the failure class and MUST update provider availability when the error maps to unavailable or cooldown state

### Requirement: Metadata operations normalize provider outputs before applying fields
The system SHALL convert provider-specific search, detail, image, people, and hierarchy responses into normalized metadata candidates and detail outputs before applying catalog fields or related records.

#### Scenario: TMDB detail is normalized
- **WHEN** a TMDB detail response is fetched for a movie or series
- **THEN** the system MUST normalize supported title, original title, overview, release date, year, runtime, images, people, external IDs, and hierarchy data before applying catalog updates

#### Scenario: MetaTube detail is normalized
- **WHEN** a MetaTube detail response is fetched for a movie
- **THEN** the system MUST normalize supported title, original title, overview, release date, year, runtime, images, people, and provider-specific external IDs without converting them into fake TMDB semantics

### Requirement: Metadata operations apply fields through ownership policy
The system SHALL apply canonical catalog fields through a field ownership policy that respects locked fields, manual edits, operation type, source provenance, and confidence.

#### Scenario: Automated operation sees locked field
- **WHEN** an automated match or refetch retrieves a value for a locked field
- **THEN** the operation MUST record the field as skipped and MUST NOT overwrite the canonical value

#### Scenario: Manual operation force applies field
- **WHEN** a user manually applies a candidate with permission to override unlocked automated values
- **THEN** the operation MUST apply supported values, mark the resulting governance as manual or equivalent, and record source attribution for the applied fields

### Requirement: Metadata operations refresh projections once per scope
The system SHALL refresh catalog read projections once after a metadata operation has completed its field, identity, source, image, people, and hierarchy writes for the affected scope.

#### Scenario: Movie operation refreshes one item scope
- **WHEN** a metadata operation applies movie fields, images, people, and external IDs
- **THEN** the system MUST refresh projections for the affected movie after all writes are complete rather than once per intermediate write

#### Scenario: TV operation refreshes hierarchy scope
- **WHEN** a metadata operation updates a series and its season or episode descendants
- **THEN** the system MUST refresh projections and rollups for the series hierarchy scope after descendant writes are complete
