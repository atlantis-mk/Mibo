## ADDED Requirements

### Requirement: Plugin manifest discovery
The system SHALL discover provider plugins by fetching and validating a manifest from a plugin endpoint.

#### Scenario: Register remote plugin endpoint
- **WHEN** an administrator registers a remote plugin endpoint URL
- **THEN** the system SHALL fetch the endpoint manifest, validate its protocol version, identity, capabilities, configuration schema, and declared operation paths
- **AND** the system SHALL persist a manifest snapshot for the provider instance

#### Scenario: Reject invalid manifest
- **WHEN** a plugin endpoint returns a missing, malformed, unsupported, or capability-incomplete manifest
- **THEN** the system SHALL reject registration with a validation error
- **AND** the system SHALL NOT create an enabled provider instance

### Requirement: Unified deployment shapes
The system SHALL support remote services and local companion processes as deployment shapes for the same provider protocol.

#### Scenario: Remote and local instances expose equivalent protocol surface
- **WHEN** a remote plugin service and a local companion plugin both declare the same capability in their manifests
- **THEN** the system SHALL route capability calls through the same internal provider dispatch path after endpoint resolution

#### Scenario: Local companion endpoint is resolved before use
- **WHEN** a local companion plugin instance is enabled
- **THEN** the system SHALL start or locate the companion process, resolve its callable local endpoint, fetch its manifest, and only then mark the instance available

### Requirement: Provider instance state
The system SHALL store plugin-backed provider instances with deployment kind, endpoint, plugin identity, manifest snapshot, configuration, capabilities, health status, and enabled state.

#### Scenario: Instance state is visible without exposing secrets
- **WHEN** the frontend requests plugin-backed provider instance settings
- **THEN** the system SHALL return display metadata, capabilities, health state, deployment kind, endpoint summary, and redacted configuration state
- **AND** the system SHALL NOT return stored secret values

#### Scenario: Disabled instance is not selected for execution
- **WHEN** a plugin-backed provider instance is disabled
- **THEN** the system SHALL exclude it from capability routing, metadata profile execution, and media source execution

### Requirement: Capability routing
The system SHALL route provider operations by declared capability rather than hard-coded provider type.

#### Scenario: Capability-compatible provider is selected
- **WHEN** a metadata profile or media source operation requires a capability
- **THEN** the system SHALL select only provider instances whose manifest declares that capability and whose instance is enabled and healthy enough for execution

#### Scenario: Missing capability prevents execution
- **WHEN** an operation targets a provider instance that does not declare the required capability
- **THEN** the system SHALL reject the operation with a capability mismatch error

### Requirement: Health and failure handling
The system SHALL track plugin health and provider operation failures.

#### Scenario: Health check updates provider status
- **WHEN** the system performs a health check against a plugin endpoint
- **THEN** the system SHALL update the provider instance availability status, failure reason, and checked timestamp

#### Scenario: Rate limited plugin enters cooldown
- **WHEN** a plugin operation returns a rate limit response
- **THEN** the system SHALL mark the provider instance as cooling down for a bounded interval
- **AND** fallback-capable routing SHALL attempt the next compatible provider instance

### Requirement: Schema-driven configuration
The system SHALL render and validate plugin provider configuration from the plugin manifest configuration schema.

#### Scenario: Frontend renders plugin configuration form
- **WHEN** the frontend receives a plugin manifest with supported configuration fields
- **THEN** it SHALL render a provider configuration form using the manifest schema and shared settings components

#### Scenario: Backend validates configuration
- **WHEN** an administrator saves plugin provider configuration
- **THEN** the backend SHALL validate required fields, field types, defaults, and secret handling according to the manifest schema before persisting the instance

### Requirement: Built-in provider compatibility
The system SHALL keep existing built-in providers available while plugin-backed providers are introduced.

#### Scenario: Existing provider behavior remains available
- **WHEN** a user has existing TMDB, MetaTube, local scan, local storage, or OpenList configuration
- **THEN** the system SHALL continue supporting those providers through compatibility adapters or existing flows during migration

#### Scenario: Built-in adapter participates in capability routing
- **WHEN** a built-in provider is represented in the new routing layer
- **THEN** it SHALL declare equivalent capabilities and execute through the same capability selection path as plugin-backed instances where practical
