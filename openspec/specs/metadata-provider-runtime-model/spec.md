# metadata-provider-runtime-model Specification

## Purpose
TBD - created by archiving change rework-library-metadata-provider-model. Update Purpose after archive.
## Requirements
### Requirement: System-managed metadata provider types include local scan
The system SHALL treat local sidecar metadata as a first-class metadata provider type and SHALL ensure a built-in `local_scan` provider instance exists without requiring manual configuration.

#### Scenario: Startup bootstraps local scan provider
- **WHEN** the application starts and no `local_scan` provider instance exists yet
- **THEN** the system MUST create one enabled, system-managed provider instance that can participate in metadata execution without any operator-supplied credentials or URLs

#### Scenario: Operator views provider instances
- **WHEN** a client lists metadata provider instances
- **THEN** the response MUST include the built-in `local_scan` instance with system-managed semantics and MUST NOT require editable provider configuration fields for it

### Requirement: Library metadata strategy is the executable source of truth
The system SHALL persist each library's executable metadata strategy directly, including ordered provider instance IDs per metadata stage, local evidence behavior, and library-specific language overrides, and all metadata operations SHALL resolve execution from that strategy instead of global TMDB configuration, synthetic local-only profiles, or legacy provider-enable and priority flags.

#### Scenario: Library uses single remote provider
- **WHEN** a library strategy configures one TMDB provider instance for search and detail stages
- **THEN** metadata search, match, refetch, and manual apply operations for that library MUST resolve from that strategy row without consulting legacy TMDB enablement booleans or profile-local-only shortcuts

#### Scenario: Library uses local detail provider only
- **WHEN** a library strategy leaves search unconfigured and sets the built-in `local_scan` instance as the only local evidence or detail-stage executor
- **THEN** strategy-driven local metadata application for that library MUST consume local scan evidence and MUST NOT call a remote metadata provider for that detail stage

#### Scenario: Library uses MetaTube without TMDB
- **WHEN** a library strategy configures an operational MetaTube provider for movie search and detail and no TMDB API key exists
- **THEN** automated movie matching for that library MUST run through the MetaTube provider instead of being skipped by a global TMDB configuration gate

### Requirement: Provider stage assignments follow provider capabilities
The system SHALL validate metadata templates and library metadata strategies against provider-type stage capability rules before accepting the configuration and SHALL execute eligible providers in configured order during metadata operations.

#### Scenario: Unsupported local scan stage is rejected
- **WHEN** a client assigns the built-in `local_scan` provider to the online search stage of a template or library strategy
- **THEN** the system MUST reject the change with a validation error instead of accepting an invalid runtime configuration

#### Scenario: Ordered providers preserve fallback order
- **WHEN** a stage configures two provider instances in an explicit order
- **THEN** execution MUST attempt those provider instances in the configured order and MUST record which provider instance produced the selected result

#### Scenario: Provider fallback is attempted after no result
- **WHEN** the first configured search provider returns no usable candidates and a later configured provider is operational
- **THEN** execution MUST attempt the later provider according to the operation fallback policy and record both attempt outcomes

### Requirement: Metadata templates are reusable defaults rather than runtime sentinels
The system SHALL support reusable metadata templates, but applying a template to a library MUST copy template values into the library's executable strategy rather than making the template row a required runtime dependency.

#### Scenario: Applying template copies strategy values
- **WHEN** a user applies a metadata template to a library
- **THEN** the library's executable strategy MUST receive the template's stage ordering and language defaults as copied values that can later be edited independently

#### Scenario: Template edits do not retroactively change libraries
- **WHEN** a template is edited after it has already been applied to one or more libraries
- **THEN** existing library strategies MUST remain unchanged until a user explicitly reapplies or edits them

### Requirement: Metadata provenance records executing provider instances
The system SHALL record provider-instance provenance and operation attempt evidence for strategy-driven metadata executions, including executions that use the built-in local scan provider.

#### Scenario: Local scan refresh records provider provenance
- **WHEN** a metadata operation applies local scan evidence for a catalog item
- **THEN** the resulting metadata source evidence MUST identify the built-in `local_scan` provider instance and the operation evidence MUST identify local evidence as the selected executor

#### Scenario: Remote fallback records selected provider instance
- **WHEN** execution skips an unavailable primary remote provider and succeeds with a later configured provider instance
- **THEN** the resulting metadata source evidence MUST record the provider instance that actually produced the metadata and the operation evidence MUST retain the execution order that led to that selection
