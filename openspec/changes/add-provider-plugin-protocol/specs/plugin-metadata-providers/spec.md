## ADDED Requirements

### Requirement: Metadata search capability
The system SHALL support plugin-backed metadata candidate search through a normalized metadata search capability.

#### Scenario: Search provider returns normalized candidates
- **WHEN** a metadata profile routes a search request to a plugin-backed provider with `metadata.search`
- **THEN** the system SHALL send the item type, query terms, year hints, external ID hints, preferred language, and library context allowed by policy
- **AND** the plugin response SHALL be validated as normalized metadata candidates before the candidates are returned to the caller

#### Scenario: Invalid search result is rejected
- **WHEN** a plugin-backed metadata search response omits required candidate fields such as provider, media type, external ID, title, or confidence
- **THEN** the system SHALL reject the malformed response, record a provider failure, and continue fallback routing when configured

### Requirement: Metadata detail capability
The system SHALL support plugin-backed metadata detail retrieval through a normalized metadata detail capability.

#### Scenario: Detail provider returns normalized metadata
- **WHEN** a metadata operation applies a candidate from a plugin-backed provider with `metadata.detail`
- **THEN** the system SHALL request detail by provider identity, media type, external ID, language, and relevant context
- **AND** the plugin response SHALL be validated as a normalized metadata detail document before persistence

#### Scenario: Detail response persists canonical fields
- **WHEN** a valid plugin metadata detail response is applied
- **THEN** the system SHALL persist canonical item fields, external IDs, images, people, tags, ratings, hierarchy information, and source attribution that are present and supported by the normalized contract

### Requirement: Metadata profile integration
The system SHALL allow metadata profiles to use plugin-backed metadata provider instances for supported stages.

#### Scenario: Profile selects plugin search and detail providers
- **WHEN** an administrator configures a metadata profile
- **THEN** the system SHALL allow selection of enabled provider instances that declare metadata search and detail capabilities for their respective stages

#### Scenario: Stage fallback uses compatible providers
- **WHEN** a preferred plugin-backed metadata provider fails during a profile stage
- **THEN** the system SHALL record the failure and attempt the next enabled compatible provider instance when fallback is enabled

### Requirement: Metadata source attribution
The system SHALL preserve plugin metadata source attribution.

#### Scenario: Plugin source is recorded
- **WHEN** metadata from a plugin-backed provider is persisted
- **THEN** the system SHALL record the plugin provider identity, provider instance, external ID, language, confidence, fetch timestamp, and triggering library context where applicable

#### Scenario: Attribution is visible in governance workflows
- **WHEN** a user reviews metadata governance details
- **THEN** the system SHALL expose plugin-backed source attribution in the same governance surfaces used for built-in provider sources

### Requirement: Local evidence compatibility
The system SHALL preserve local scan metadata behavior while plugin-backed metadata providers are introduced.

#### Scenario: Local scan remains available as fallback
- **WHEN** a metadata profile relies on local scan detail fallback
- **THEN** the system SHALL continue to support local evidence and local scan detail behavior during the plugin protocol migration

#### Scenario: Plugin providers do not bypass local policy
- **WHEN** plugin metadata is applied to a library item
- **THEN** the system SHALL enforce the same library visibility, metadata policy, governance, and user edit protection rules used for built-in metadata providers
