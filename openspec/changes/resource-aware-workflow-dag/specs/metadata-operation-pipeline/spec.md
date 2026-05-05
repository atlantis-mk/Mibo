## MODIFIED Requirements

### Requirement: Metadata operations execute through a unified pipeline
The system SHALL execute automated match, refetch, manual candidate apply, and local evidence apply through a shared metadata operation pipeline that resolves target item, library strategy, execution plan, provider attempts, metadata decision, field application, and projection refresh as one operation. Automated workflow-driven metadata operations MUST be scheduled under dependency and resource-budget controls.

#### Scenario: Automated match uses unified pipeline
- **WHEN** a queued metadata workflow task runs for a pending catalog item
- **THEN** the system MUST execute a metadata operation of type `match` and return or persist a result that includes the target item, execution plan summary, provider attempts, selected candidate when present, applied fields, skipped fields, resulting governance status, and affected catalog item IDs

#### Scenario: Refetch uses unified pipeline
- **WHEN** a user refetches metadata for an item that already has provider identity
- **THEN** the system MUST execute a metadata operation of type `refetch` using the same execution plan, provider attempt, field application, and projection refresh semantics as automated matching

#### Scenario: Manual candidate apply uses unified pipeline
- **WHEN** a user applies a selected metadata candidate to a catalog item
- **THEN** the system MUST execute a metadata operation of type `manual_apply` and MUST record manual operation status, provider provenance, applied fields, skipped locked fields, and affected catalog item IDs

### Requirement: Metadata operations persist or expose provider attempts
The system SHALL retain enough provider attempt evidence for each metadata operation to explain which configured providers were attempted, skipped, failed, or selected for each metadata stage. Provider attempts initiated by workflow tasks MUST also expose resource wait, retry, and cooldown state when those states delay execution.

#### Scenario: Primary provider unavailable
- **WHEN** the first configured search provider is disabled, unavailable, or in cooldown and a later provider supplies candidates
- **THEN** the operation evidence MUST record the first provider as skipped with its reason and the later provider as the selected successful attempt

#### Scenario: Provider returns no candidates
- **WHEN** a provider search request succeeds but returns no usable candidates
- **THEN** the operation evidence MUST record a `no_result` attempt instead of treating the operation as an infrastructure failure

#### Scenario: Provider request fails
- **WHEN** a provider request fails with authentication, rate limit, timeout, or remote error
- **THEN** the operation evidence MUST record the failure class and MUST update provider availability when the error maps to unavailable or cooldown state
