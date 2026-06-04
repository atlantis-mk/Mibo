## ADDED Requirements

### Requirement: Storage browse capability
The system SHALL support plugin-backed storage browsing through a normalized storage browse capability.

#### Scenario: Browse plugin storage path
- **WHEN** an administrator browses a plugin-backed media source path
- **THEN** the system SHALL call a provider instance with `storage.browse` and return normalized storage objects including name, path, directory flag, size, timestamps, thumbnails, stable identity, and provider metadata where available

#### Scenario: Browse response is validated
- **WHEN** a plugin-backed storage browse response contains malformed object paths or unsupported object shapes
- **THEN** the system SHALL reject the response and record a provider failure

### Requirement: Storage resolve capability
The system SHALL support resolving plugin-backed storage objects before creating or updating media sources.

#### Scenario: Resolve root path during media source creation
- **WHEN** an administrator creates a media source using a plugin-backed storage provider
- **THEN** the system SHALL call `storage.resolve` for the requested root path and only persist the media source if the plugin returns a valid resolved storage object and capabilities

#### Scenario: Resolve failure blocks media source activation
- **WHEN** a plugin-backed provider cannot resolve the requested root path
- **THEN** the system SHALL reject media source creation or update with a provider error and SHALL NOT mark the source active

### Requirement: Storage link capability
The system SHALL support plugin-backed playable link creation when the provider declares link capability.

#### Scenario: Link playable object
- **WHEN** playback or probing needs a playable link for an object from a plugin-backed storage provider
- **THEN** the system SHALL call `storage.link` and use the returned link only after validating URL shape, expiration metadata when present, and capability policy

#### Scenario: Missing link capability falls back to existing access behavior
- **WHEN** a plugin-backed storage provider does not declare `storage.link`
- **THEN** the system SHALL avoid calling link operations and SHALL use only supported access paths for that provider

### Requirement: Media source integration
The system SHALL allow media sources to use plugin-backed storage provider instances.

#### Scenario: Plugin-backed media source is listed
- **WHEN** a plugin-backed media source has been created
- **THEN** the system SHALL list it with provider display metadata, root path, redacted configuration, capabilities, health status, and timestamps

#### Scenario: Scanner uses plugin-backed source
- **WHEN** a library scan runs against a plugin-backed media source
- **THEN** the scanner SHALL browse and resolve files through the plugin storage capability contract while preserving existing inventory, recognition, exclusion, and materialization behavior

### Requirement: Storage security boundaries
The system SHALL enforce existing access control and path policy for plugin-backed storage providers.

#### Scenario: Plugin storage does not bypass authorization
- **WHEN** a user browses or plays content from a plugin-backed media source
- **THEN** the system SHALL enforce the same authentication, library visibility, role, and playback access checks used for built-in storage providers

#### Scenario: Plugin provider metadata is sanitized
- **WHEN** storage objects from a plugin-backed provider include provider metadata
- **THEN** the system SHALL store or return only sanitized metadata fields approved by the storage contract
