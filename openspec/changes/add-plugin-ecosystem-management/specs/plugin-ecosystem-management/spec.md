## ADDED Requirements

### Requirement: Admin plugin center

The system SHALL provide an administrator-only settings area for managing plugins as first-class extensions.

#### Scenario: Administrator opens plugin center

- **WHEN** an administrator opens the settings plugin center
- **THEN** the system SHALL show registered plugin-backed provider instances, their deployment kind, enabled state, availability status, capabilities, and plugin identity
- **AND** the system SHALL provide actions appropriate to each instance state

#### Scenario: Non-admin cannot access plugin center

- **WHEN** a non-admin user attempts to open the plugin center or call plugin lifecycle management APIs
- **THEN** the system SHALL deny access

### Requirement: Centralized plugin provider instance management

The system SHALL allow administrators to register, edit, enable or disable, and refresh health for plugin-backed provider instances from the plugin center.

#### Scenario: Register remote plugin provider

- **WHEN** an administrator registers a remote plugin endpoint from the plugin center
- **THEN** the system SHALL fetch and validate the plugin manifest
- **AND** the system SHALL render configuration fields from the manifest schema
- **AND** the system SHALL create a plugin-backed provider instance when required configuration is valid

#### Scenario: Refresh plugin health

- **WHEN** an administrator refreshes a plugin provider instance health state
- **THEN** the system SHALL call the plugin health endpoint and update availability status, failure reason, cooldown state, and checked timestamp

### Requirement: Plugin detail and diagnostics

The system SHALL expose detailed plugin diagnostics for administrators.

#### Scenario: View plugin detail

- **WHEN** an administrator opens a plugin detail view
- **THEN** the system SHALL show manifest metadata, protocol version, plugin version, endpoint, deployment kind, declared capabilities, configuration schema summary, enabled state, availability status, failure reason, cooldown state, and last checked timestamp

#### Scenario: Plugin is unhealthy

- **WHEN** a plugin instance is unavailable or cooling down
- **THEN** the plugin center SHALL make the degraded state visible
- **AND** it SHALL show the most recent failure reason when available

### Requirement: Plugin usage references

The system SHALL show where plugin-backed provider instances are used before administrators perform disruptive actions.

#### Scenario: Plugin is used by profiles or media sources

- **WHEN** a plugin-backed provider instance is referenced by metadata profiles or media sources
- **THEN** the plugin center SHALL show those references in the plugin detail view

#### Scenario: Administrator disables a referenced plugin

- **WHEN** an administrator attempts to disable, uninstall, update, or roll back a plugin with active references
- **THEN** the system SHALL warn that dependent metadata profiles or media sources may be affected
- **AND** the action SHALL require explicit confirmation

### Requirement: Local companion plugin lifecycle

The system SHALL support local companion plugins as managed installations that expose the existing provider plugin protocol after endpoint resolution.

#### Scenario: Install local companion plugin

- **WHEN** an administrator installs or registers a local companion plugin from a supported source
- **THEN** the system SHALL persist installation identity, version, source metadata, install state, enabled state, and lifecycle timestamps
- **AND** it SHALL not route provider operations through the companion until a valid endpoint and manifest are resolved

#### Scenario: Start local companion plugin

- **WHEN** an administrator starts a local companion plugin
- **THEN** the system SHALL launch or locate the companion process, resolve its endpoint, fetch and validate its manifest, and update runtime state

#### Scenario: Stop local companion plugin

- **WHEN** an administrator stops a local companion plugin
- **THEN** the system SHALL stop the managed process when possible
- **AND** provider instances depending on that companion SHALL no longer be considered operational

#### Scenario: View local companion logs

- **WHEN** an administrator views local companion diagnostics
- **THEN** the system SHALL provide recent lifecycle or process logs without exposing stored secrets

### Requirement: Catalog and update readiness

The system SHALL model plugin catalog and update metadata so discovery, compatibility checks, trust warnings, upgrades, and rollback can be added safely.

#### Scenario: Catalog entry is available

- **WHEN** a configured plugin catalog source lists a plugin
- **THEN** the system SHALL show plugin identity, version, capabilities, protocol compatibility, platform constraints, source metadata, and trust metadata

#### Scenario: Plugin is incompatible

- **WHEN** a catalog plugin version is incompatible with the current Mibo version, protocol version, platform, or required capabilities
- **THEN** the system SHALL prevent installation or update by default
- **AND** it SHALL explain the compatibility reason to the administrator

#### Scenario: Plugin update is available

- **WHEN** an installed plugin has a compatible update available from a configured source
- **THEN** the system SHALL show the update and release notes
- **AND** it SHALL require administrator confirmation before applying the update
